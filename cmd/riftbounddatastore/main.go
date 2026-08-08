// Command riftbounddatastore builds the Riftbound datastore file: it
// downloads the official card-gallery payload (resolving the current site
// build id) and appends the promotional printings TCGplayer carries but the
// gallery does not, as separate promo-typed sets, so promo listings resolve
// to their own uuids instead of polluting the main printings.
//
// The output is the gallery payload itself with the extra sets and cards
// merged into the gallery blade, so mtgmatcher/riftbound loads it unchanged.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

const (
	galleryPageURL = "https://riftbound.leagueoflegends.com/en-us/card-gallery/"
	galleryDataURL = "https://riftbound.leagueoflegends.com/_next/data/%s/en-us/card-gallery.json"

	// tcgcsv republishes the TCGplayer catalog daily; category 89 is
	// Riftbound.
	tcgcsvGroupsURL   = "https://tcgcsv.com/tcgplayer/89/groups"
	tcgcsvProductsURL = "https://tcgcsv.com/tcgplayer/89/%d/products"
)

var buildIdRe = regexp.MustCompile(`"buildId":"([^"]+)"`)

func fetch(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

type tcgGroup struct {
	GroupID      int    `json:"groupId"`
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
}

type tcgProduct struct {
	ProductID    int    `json:"productId"`
	Name         string `json:"name"`
	ImageURL     string `json:"imageUrl"`
	ExtendedData []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"extendedData"`
}

func (p tcgProduct) extended(name string) string {
	for _, e := range p.ExtendedData {
		if e.Name == name {
			return e.Value
		}
	}
	return ""
}

// isPromoGroup reports whether a TCGplayer group holds promotional printings
// rather than a main set the gallery already covers (or will cover once
// published, like preview-season sets).
func isPromoGroup(g tcgGroup) bool {
	return strings.Contains(g.Name, "Promotional") || strings.Contains(g.Name, "Bundle")
}

func main() {
	output := flag.String("o", "", "output file (default stdout)")
	flag.Parse()

	page, err := fetch(galleryPageURL)
	if err != nil {
		log.Fatalln("gallery page:", err)
	}
	m := buildIdRe.FindSubmatch(page)
	if m == nil {
		log.Fatalln("no buildId found in the gallery page")
	}
	payload, err := fetch(fmt.Sprintf(galleryDataURL, m[1]))
	if err != nil {
		log.Fatalln("gallery payload:", err)
	}

	// Decode the payload generically so everything the loader does not care
	// about survives the round trip untouched.
	var doc map[string]any
	if err := json.Unmarshal(payload, &doc); err != nil {
		log.Fatalln("gallery payload:", err)
	}
	blades, _ := doc["pageProps"].(map[string]any)["page"].(map[string]any)["blades"].([]any)
	var gallery map[string]any
	for _, b := range blades {
		blade, ok := b.(map[string]any)
		if ok && blade["type"] == "riftboundCardGallery" {
			gallery = blade
			break
		}
	}
	if gallery == nil {
		log.Fatalln("no card gallery blade in the payload")
	}
	sets := gallery["sets"].(map[string]any)
	cards := gallery["cards"].(map[string]any)
	setItems := sets["items"].([]any)
	cardItems := cards["items"].([]any)

	var groupsResp struct {
		Results []tcgGroup `json:"results"`
	}
	groupsData, err := fetch(tcgcsvGroupsURL)
	if err != nil {
		log.Fatalln("tcgcsv groups:", err)
	}
	if err := json.Unmarshal(groupsData, &groupsResp); err != nil {
		log.Fatalln("tcgcsv groups:", err)
	}

	for _, group := range groupsResp.Results {
		if !isPromoGroup(group) {
			continue
		}

		productsData, err := fetch(fmt.Sprintf(tcgcsvProductsURL, group.GroupID))
		if err != nil {
			log.Fatalln(group.Name, ":", err)
		}
		var productsResp struct {
			Results []tcgProduct `json:"results"`
		}
		if err := json.Unmarshal(productsData, &productsResp); err != nil {
			log.Fatalln(group.Name, ":", err)
		}

		var added, maxNum int
		for _, product := range productsResp.Results {
			number := product.extended("Number")
			if number == "" {
				// Not a single (sealed, accessories, the odd unnumbered
				// promo): nothing to identify it by.
				continue
			}
			collector := 0
			fmt.Sscanf(strings.TrimLeft(number, "0"), "%d", &collector)
			if collector > maxNum {
				maxNum = collector
			}

			cardItems = append(cardItems, map[string]any{
				// The TCGplayer product id is the stable identity of a
				// promo printing; group-prefixed for readability.
				"id":              fmt.Sprintf("%s-%d", strings.ToLower(group.Abbreviation), product.ProductID),
				"collectorNumber": collector,
				"name":            product.Name,
				"publicCode":      fmt.Sprintf("%s-%s", group.Abbreviation, number),
				"orientation":     "portrait",
				"set": map[string]any{
					"value": map[string]any{
						"id":    group.Abbreviation,
						"label": group.Name,
					},
				},
				"rarity": map[string]any{
					"value": map[string]any{
						"id": strings.ToLower(product.extended("Rarity")),
					},
				},
				"cardImage": map[string]any{
					"url": product.ImageURL,
				},
			})
			added++
		}
		if added == 0 {
			continue
		}

		setItems = append(setItems, map[string]any{
			"id":                 group.Abbreviation,
			"name":               group.Name,
			"collectorNumberMax": maxNum,
			"type":               "promo",
		})
		log.Printf("%s (%s): %d promo printings", group.Name, group.Abbreviation, added)
	}

	sets["items"] = setItems
	cards["items"] = cardItems

	out := os.Stdout
	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			log.Fatalln(err)
		}
		defer f.Close()
		out = f
	}
	enc := json.NewEncoder(out)
	if err := enc.Encode(doc); err != nil {
		log.Fatalln(err)
	}
}
