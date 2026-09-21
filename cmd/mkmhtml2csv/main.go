// Command mkmhtml2csv parses a saved Cardmarket seller offers HTML page
// (view-source or raw) and produces a CSV of matched cards.
//
// Usage:
//
//	mkmhtml2csv -datastore ~/allprintings5.json offers.html > cards.csv
//	mkmhtml2csv -datastore ~/allprintings5.json offers1.html offers2.html
package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"html"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

var conditionMap = map[string]string{
	"Mint":         "NM",
	"Near Mint":    "NM",
	"Excellent":    "SP",
	"Good":         "MP",
	"Light Played": "MP",
	"Played":       "HP",
	"Poor":         "PO",
}

type cardEntry struct {
	ArticleID string
	McmID     string
	SetSlug   string
	CardSlug  string
	Condition string
	Foil      bool
	Qty       string
	Price     string
}

var (
	stockRowRE = regexp.MustCompile(`stockRow(\d+)`)
	productRE  = regexp.MustCompile(`/Products/Singles/([^/?\"<& ]+/[^/?\"<& ]+)(?:\?language=(\d+))?`)
	priceRE    = regexp.MustCompile(`(\d+[.,]\d{2})\s*€`)
	condRE     = regexp.MustCompile(`(?:data-bs-original-title|title)="(Mint|Near Mint|Excellent|Good|Light Played|Played|Poor)"`)
	qtyRE      = regexp.MustCompile(`item-count[^"]*">\s*(\d+)`)
	foilRE     = regexp.MustCompile(`(?:data-bs-original-title|title)="Foil"`)
	skipRE     = regexp.MustCompile(`(?:data-bs-original-title|title)="(?:Altered|Signed|Inked)"`)
	mcmImgRE   = regexp.MustCompile(`/items/\d+/(\d+)/`)
	mcmS3ImgRE = regexp.MustCompile(`product-images\.s3\.cardmarket\.com/\d+/[^/]+/(\d+)/`)
)

func cleanHTML(raw string) string {
	// Only strip view-source wrapper tags if this is a view-source dump
	if strings.Contains(raw, `class="start-tag"`) {
		s := regexp.MustCompile(`<span[^>]*>`).ReplaceAllString(raw, "")
		s = strings.ReplaceAll(s, "</span>", "")
		s = regexp.MustCompile(`<a[^>]*class="attribute-value"[^>]*>`).ReplaceAllString(s, "")
		s = strings.ReplaceAll(s, "</a>", "")
		return html.UnescapeString(s)
	}
	return raw
}

func parseHTML(content string, langFilter string) []cardEntry {
	cleaned := cleanHTML(content)

	// Split into stockRow sections by finding all positions
	rowStarts := stockRowRE.FindAllStringIndex(cleaned, -1)
	var sections []string
	for i, loc := range rowStarts {
		end := len(cleaned)
		if i+1 < len(rowStarts) {
			end = rowStarts[i+1][0]
		}
		sections = append(sections, cleaned[loc[0]:end])
	}

	seen := map[string]bool{}
	var entries []cardEntry

	for _, section := range sections {
		article := stockRowRE.FindStringSubmatch(section)
		if article == nil {
			continue
		}
		articleID := article[1]
		if seen[articleID] {
			continue
		}
		seen[articleID] = true

		product := productRE.FindStringSubmatch(section)
		if product == nil {
			continue
		}
		parts := strings.SplitN(product[1], "/", 2)
		if len(parts) != 2 {
			continue
		}

		// Skip cards that don't match the language filter
		if langFilter != "" && product[2] != "" && product[2] != langFilter {
			continue
		}

		// Skip altered, signed, or inked cards
		if skipRE.MatchString(section) {
			continue
		}

		price := priceRE.FindStringSubmatch(section)
		cond := condRE.FindStringSubmatch(section)
		qty := qtyRE.FindStringSubmatch(section)
		// Unescape HTML entities in section for image URL matching
		unescaped := html.UnescapeString(section)
		mcmImg := mcmImgRE.FindStringSubmatch(unescaped)
		foil := foilRE.MatchString(section)

		entry := cardEntry{
			ArticleID: articleID,
			SetSlug:   parts[0],
			CardSlug:  parts[1],
		}

		if mcmImg != nil {
			entry.McmID = mcmImg[1]
		} else if mcmS3 := mcmS3ImgRE.FindStringSubmatch(unescaped); mcmS3 != nil {
			entry.McmID = mcmS3[1]
		}
		if price != nil {
			entry.Price = strings.ReplaceAll(price[1], ",", ".")
		}
		if cond != nil {
			entry.Condition = cond[1]
		}
		if qty != nil {
			entry.Qty = qty[1]
		} else {
			entry.Qty = "1"
		}
		entry.Foil = foil

		entries = append(entries, entry)
	}

	return entries
}

// slugToName converts a Cardmarket URL slug back to a card name.
// "Mirri-s-Guile" → "Mirri's Guile", "Jace-the-Mind-Sculptor" → "Jace the Mind Sculptor"
// Strips trailing version suffixes like "-V1", "-V2".
func slugToName(slug string) string {
	// Remove version suffixes
	slug = regexp.MustCompile(`-V\d+$`).ReplaceAllString(slug, "")
	// Restore apostrophes: "-s-" between words → "'s "
	slug = strings.ReplaceAll(slug, "-s-", "'s-")
	slug = strings.ReplaceAll(slug, "-", " ")
	return slug
}

func readClipboard() ([]byte, error) {
	cmd := exec.Command("pbpaste")
	return cmd.Output()
}

func findDatastore() string {
	candidates := []string{
		"allprintings5.json",
		filepath.Join(os.Getenv("HOME"), "allprintings5.json"),
		filepath.Join(os.Getenv("HOME"), "AllPrintings.json"),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func processEntries(backend *mtgmatcher.Backend, entries []cardEntry, w *csv.Writer, eurRate float64, mcmIDMap map[string]string) (matched, unmatched int) {
	for _, entry := range entries {
		cond := conditionMap[entry.Condition]
		if cond == "" {
			cond = entry.Condition
		}

		foilStr := ""
		if entry.Foil {
			foilStr = "foil"
		}

		// Convert EUR to USD
		priceUSD := entry.Price
		eur, parseErr := strconv.ParseFloat(entry.Price, 64)
		if parseErr == nil && eurRate > 0 {
			priceUSD = fmt.Sprintf("%.2f", eur*eurRate)
		}

		// Try to match via mcmId lookup
		var cardID string
		var err error
		if entry.McmID != "" {
			if uuid, ok := mcmIDMap[entry.McmID]; ok {
				cardID, err = backend.MatchID(uuid, entry.Foil)
				if err != nil {
					log.Printf("MatchID(%s, foil=%v) failed for %s/%s: %v",
						uuid, entry.Foil, entry.SetSlug, entry.CardSlug, err)
					cardID = ""
				}
			}
		}

		// Fall back to name-based matching
		if cardID == "" {
			cardName := slugToName(entry.CardSlug)
			setName := strings.ReplaceAll(entry.SetSlug, "-", " ")

			// SplitVariants splits on a parenthetical, not on the "-V1"
			// suffix Cardmarket spells a variant with - slugToName has
			// already stripped that one.
			variation := cond
			vars := mtgmatcher.SplitVariants(cardName)
			if len(vars) > 1 {
				cardName = vars[0]
				variation = strings.Join(vars[1:], " ") + " " + cond
			}

			theCard := &mtgmatcher.InputCard{
				Name:      cardName,
				Edition:   setName,
				Foil:      entry.Foil,
				Variation: strings.TrimSpace(variation),
			}
			cardID, err = backend.Match(theCard)
			if err != nil {
				log.Printf("Match failed for %s/%s (mcm=%s): %v",
					entry.SetSlug, entry.CardSlug, entry.McmID, err)
				unmatched++

				w.Write([]string{
					"", entry.CardSlug, "", "",
					cond, foilStr, entry.Qty, priceUSD,
					entry.McmID, entry.ArticleID, entry.SetSlug, entry.CardSlug,
				})
				continue
			}
		}

		// A uuid Match just handed back should resolve, but a miss here
		// would nil-deref the fields below.
		co, err := backend.GetUUID(cardID)
		if err != nil {
			log.Printf("GetUUID(%s) failed for %s/%s: %v",
				cardID, entry.SetSlug, entry.CardSlug, err)
			unmatched++

			w.Write([]string{
				"", entry.CardSlug, "", "",
				cond, foilStr, entry.Qty, priceUSD,
				entry.McmID, entry.ArticleID, entry.SetSlug, entry.CardSlug,
			})
			continue
		}
		matched++

		w.Write([]string{
			cardID, co.Name, co.SetCode, co.Number,
			cond, foilStr, entry.Qty, priceUSD,
			entry.McmID, entry.ArticleID, entry.SetSlug, entry.CardSlug,
		})
	}
	return
}

var csvHeader = []string{
	"uuid", "card_name", "set_code", "number",
	"condition", "foil", "qty", "price_usd",
	"mcm_id", "article_id", "mkm_set", "mkm_card",
}

func main() {
	datastoreOpt := flag.String("datastore", "", "Path to AllPrintings JSON (auto-detected if omitted)")
	outOpt := flag.String("o", "", "Output CSV path (default: stdout)")
	langOpt := flag.String("lang", "", "Filter by Cardmarket language ID (1=English, 7=Japanese, empty=all)")
	flag.Parse()

	// Auto-find datastore
	dsPath := *datastoreOpt
	if dsPath == "" {
		dsPath = findDatastore()
		if dsPath == "" {
			log.Fatal("Cannot find allprintings5.json — use -datastore flag")
		}
	}

	log.Printf("Loading datastore %s (please wait)...", dsPath)
	f, err := os.Open(dsPath)
	if err != nil {
		log.Fatalf("open datastore: %v", err)
	}
	t0 := time.Now()
	backend, err := magic.Load(f)
	f.Close()
	if err != nil {
		log.Fatalf("load datastore: %v", err)
	}
	log.Printf("Loaded datastore in %s (%s)", time.Since(t0).Round(time.Millisecond), dsPath)

	// Fetch EUR→USD exchange rate
	eurRate, err := mtgban.GetExchangeRate(context.Background(), "EUR")
	if err != nil {
		log.Printf("WARNING: could not fetch exchange rate: %v (using 1.0)", err)
		eurRate = 1.0
	} else {
		log.Printf("EUR→USD rate: %.4f", eurRate)
	}

	// Build mcmId → base UUID lookup, skipping _f/_e suffixed duplicates
	mcmIDMap := map[string]string{}
	for _, uuid := range backend.GetUUIDs() {
		if strings.HasSuffix(uuid, "_f") || strings.HasSuffix(uuid, "_e") {
			continue
		}
		co, err := backend.GetUUID(uuid)
		if err != nil {
			continue
		}
		mcmID := co.Identifiers["mcmId"]
		if mcmID != "" {
			mcmIDMap[mcmID] = uuid
		}
	}
	log.Printf("Built mcmId map: %d entries", len(mcmIDMap))

	// File mode: process files from args (supports glob patterns)
	if flag.NArg() > 0 {
		out := os.Stdout
		if *outOpt != "" {
			var err error
			out, err = os.Create(*outOpt)
			if err != nil {
				log.Fatalf("create %s: %v", *outOpt, err)
			}
			defer out.Close()
		}
		w := csv.NewWriter(out)
		w.Write(csvHeader)

		// Expand glob patterns
		var files []string
		for _, pattern := range flag.Args() {
			matches, err := filepath.Glob(pattern)
			if err != nil {
				log.Fatalf("bad glob %q: %v", pattern, err)
			}
			if len(matches) == 0 {
				files = append(files, pattern)
			} else {
				files = append(files, matches...)
			}
		}

		var totalMatched, totalUnmatched int
		for _, path := range files {
			data, err := os.ReadFile(path)
			if err != nil {
				log.Fatalf("read %s: %v", path, err)
			}
			entries := parseHTML(string(data), *langOpt)
			log.Printf("%s: found %d cards", path, len(entries))
			m, u := processEntries(backend, entries, w, eurRate, mcmIDMap)
			totalMatched += m
			totalUnmatched += u
		}
		w.Flush()
		log.Printf("Matched %d/%d cards (%d total files)",
			totalMatched, totalMatched+totalUnmatched, len(files))
		return
	}

	// Pipe mode: read stdin, write to stdout or -o
	stat, _ := os.Stdin.Stat()
	if stat.Mode()&os.ModeCharDevice == 0 {
		out := os.Stdout
		if *outOpt != "" {
			var err error
			out, err = os.Create(*outOpt)
			if err != nil {
				log.Fatalf("create %s: %v", *outOpt, err)
			}
			defer out.Close()
		}
		w := csv.NewWriter(out)
		w.Write(csvHeader)

		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("read stdin: %v", err)
		}
		entries := parseHTML(string(data), *langOpt)
		log.Printf("stdin: found %d cards", len(entries))
		m, u := processEntries(backend, entries, w, eurRate, mcmIDMap)
		w.Flush()
		log.Printf("Matched %d/%d cards (%d unmatched)", m, m+u, u)
		return
	}

	// Interactive mode: accumulate all entries, dump CSV on exit
	log.Println("Interactive mode — commands:")
	log.Println("  paste    — read HTML from clipboard (pbpaste)")
	log.Println("  <path>   — read HTML from a file path")
	log.Println("  quit     — exit and save CSV")

	var allEntries []cardEntry
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Fprint(os.Stderr, "\n> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())

		switch {
		case input == "":
			continue
		case input == "quit" || input == "exit":
			goto dump
		case input == "paste":
			data, err := readClipboard()
			if err != nil {
				log.Printf("clipboard: %v", err)
				continue
			}
			entries := parseHTML(string(data), *langOpt)
			if len(entries) == 0 {
				log.Println("No cards found in clipboard")
				continue
			}
			allEntries = append(allEntries, entries...)
			log.Printf("Found %d cards from clipboard (%d total)", len(entries), len(allEntries))
		default:
			input = strings.ReplaceAll(input, `\`, "")
			input = strings.Trim(input, `'" `)
			data, err := os.ReadFile(input)
			if err != nil {
				log.Printf("read %s: %v", input, err)
				continue
			}
			entries := parseHTML(string(data), *langOpt)
			if len(entries) == 0 {
				log.Printf("No cards found in %s", input)
				continue
			}
			allEntries = append(allEntries, entries...)
			log.Printf("Found %d cards from %s (%d total)", len(entries), input, len(allEntries))
		}
	}

dump:
	if len(allEntries) == 0 {
		log.Println("No cards collected")
		return
	}

	outPath := *outOpt
	if outPath == "" {
		outPath = fmt.Sprintf("dump-%s.csv", time.Now().Format("2006-01-02"))
	}
	out, err := os.Create(outPath)
	if err != nil {
		log.Fatalf("create %s: %v", outPath, err)
	}
	defer out.Close()

	w := csv.NewWriter(out)
	w.Write(csvHeader)
	m, u := processEntries(backend, allEntries, w, eurRate, mcmIDMap)
	w.Flush()
	log.Printf("Wrote %s: %d matched, %d unmatched (%d total cards)", outPath, m, u, m+u)
}
