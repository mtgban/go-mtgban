package starcitygames

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"unicode"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// scgCatalogURL is the HawkSearch catalog export. It returns the full product
// catalog (both retail price/qty and buylist sell_list_price per variant) as a
// single JSON array, authenticated with an x-api-key header.
const scgCatalogURL = "https://api.starcitygames.com/hawksearch/catalog/download/json"

// catalogAttempts bounds the replays of a broken export, matching the retry
// budget the client gives an ordinary request.
const catalogAttempts = 10

// Product types the catalog reports. Everything SCG sells shares one
// export, so this is what separates cards from boxes from playmats.
const (
	ProductTypeSingles = "Singles"
	ProductTypeSealed  = "Sealed"
)

// CatalogProduct is a single card printing in the catalog export.
type CatalogProduct struct {
	ID              int              `json:"id"`
	SKU             string           `json:"sku"`
	ProductType     string           `json:"product_type"`
	ScryfallID      string           `json:"scryfall_id"`
	TCGPlayerID     string           `json:"tcgplayer_id"`
	URL             string           `json:"url"`
	Name            string           `json:"name"`
	Game            string           `json:"game"`
	Set             string           `json:"set"`
	Finish          string           `json:"finish"`
	FinishGroup     string           `json:"finish_group"`
	Language        string           `json:"language"`
	CollectorNumber string           `json:"collector_number"`
	Variants        []CatalogVariant `json:"variants"`
}

// CatalogVariant is a product in a specific condition, with its own retail
// price/quantity and buylist (sell_list) price.
type CatalogVariant struct {
	ID            int    `json:"id"`
	SKU           string `json:"sku"`
	URL           string `json:"url"`
	Condition     string `json:"condition"`
	Qty           int    `json:"qty"`
	Price         string `json:"price"`
	IsOnDiscount  bool   `json:"is_on_discount"`
	SellListPrice string `json:"sell_list_price"`
}

// DownloadCatalog fetches the catalog export stream. The caller must close the
// returned reader.
func (scg *SCGClient) DownloadCatalog(ctx context.Context) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, scg.catalogURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", scg.apiKey)

	resp, err := scg.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		resp.Body.Close()
		return nil, fmt.Errorf("catalog download failed: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return resp.Body, nil
}

// StreamCatalog hands every product in the catalog export to fn, restarting
// the download if the stream breaks partway through.
//
// The export is one long-lived response of a hundred-odd megabytes, and the
// client's own retry can only replay a request that never returned a status.
// A connection dropped after the header - "stream error: stream ID 1;
// INTERNAL_ERROR" is the one seen in practice - lands past it, and used to end
// the run with an empty catalog. Replaying from the top costs a second
// download and is the only recovery available, since the export is not
// resumable; reset undoes whatever the abandoned pass accumulated.
func (scg *SCGClient) StreamCatalog(ctx context.Context, reset func(), fn func(CatalogProduct) error) error {
	var err error
	for attempt := 0; attempt < catalogAttempts; attempt++ {
		if attempt > 0 {
			reset()
		}
		err = scg.streamCatalogOnce(ctx, fn)
		if err == nil {
			return nil
		}
		// A cancelled context is the caller giving up, not a flaky peer.
		if ctx.Err() != nil {
			return err
		}
	}
	return err
}

func (scg *SCGClient) streamCatalogOnce(ctx context.Context, fn func(CatalogProduct) error) error {
	body, err := scg.DownloadCatalog(ctx)
	if err != nil {
		return err
	}
	defer body.Close()
	return decodeCatalog(body, fn)
}

// decodeCatalog streams the catalog array, invoking fn for each product without
// buffering the whole (large) response in memory.
func decodeCatalog(r io.Reader, fn func(CatalogProduct) error) error {
	dec := json.NewDecoder(r)

	// Opening '['
	if _, err := dec.Token(); err != nil {
		return err
	}
	for dec.More() {
		var p CatalogProduct
		if err := dec.Decode(&p); err != nil {
			return err
		}
		if err := fn(p); err != nil {
			return err
		}
	}
	// Closing ']'
	_, err := dec.Token()
	return err
}

// gameFromCatalog maps the catalog game string to the internal game constant.
// An unknown string maps to 0, which matches no configured scraper, so those
// products are skipped.
func gameFromCatalog(game string) int {
	switch game {
	case "Magic: The Gathering":
		return GameMagic
	case "Flesh and Blood":
		return GameFleshAndBlood
	case "Lorcana":
		return GameLorcana
	case "Riftbound: League of Legends TCG":
		return GameRiftbound
	default:
		return 0
	}
}

// catalogFoil reports the foil flag from the broad finish grouping. "Non-foil"
// is plain; "Foil" and "Alt Foil" (etched, surge, rainbow, cold, …) are foil.
func catalogFoil(p CatalogProduct) bool {
	return p.FinishGroup != "Non-foil"
}

// catalogHit synthesizes the minimal Hit that preprocess needs from a catalog
// product, used as the fallback when the Scryfall shortcut doesn't apply.
func catalogHit(p CatalogProduct, foil bool) Hit {
	finishType := 1
	if foil {
		finishType = 2
	}
	return Hit{
		Name:                p.Name,
		SetName:             p.Set,
		Language:            p.Language,
		CollectorNumber:     p.CollectorNumber,
		FinishPricingTypeID: finishType,
		Variants:            []Variant{{Sku: p.SKU}},
	}
}

// resolveProduct returns the mtgban card id for a catalog product.
//
// The Scryfall id is authoritative: when present it resolves directly through
// the identifier index, skipping preprocess entirely. Etched is the only
// alt-foil that changes the printing (and only for two sets); it shares the
// plain foil's Scryfall id, so it is detected from the finish name and handed to
// MatchID, which associates the foil id with its etched sibling. Every other
// alt-foil (surge/rainbow/cold) resolves to the plain foil. When the id is
// missing or unresolved, it falls back to the SKU-driven preprocess path.
func resolveProduct(game int, p CatalogProduct) (string, error) {
	id, err := resolveProductID(game, p)
	if err != nil {
		return "", err
	}

	// The inherently foreign sets hold a single printing each - FBB is
	// Italian and 4BB Japanese - while SCG sells them in six or seven
	// languages. Every language other than the one the printing actually
	// is collapses onto that uuid, and the products then fight over the
	// same key. Keep only the language that matches.
	co, cerr := mtgmatcher.GetUUID(id)
	if cerr == nil && !languageMatches(p.Language, co.Language) {
		return "", mtgmatcher.ErrUnsupported
	}
	return id, nil
}

// languageMatches reports whether the language a product is sold in is
// the language of the printing it resolved to. The catalog spells the
// two-part languages with a dash that mtgjson does not use.
func languageMatches(catalogLanguage, cardLanguage string) bool {
	if catalogLanguage == "" {
		catalogLanguage = "English"
	}
	if cardLanguage == "" {
		cardLanguage = "English"
	}
	return strings.EqualFold(strings.ReplaceAll(catalogLanguage, " - ", " "), cardLanguage)
}

// skuSetCode returns the set segment of a catalog sku, which carries
// detail the product's own set name has lost.
func skuSetCode(sku string) string {
	fields := strings.Split(sku, "-")
	if len(fields) < 3 {
		return ""
	}
	return fields[2]
}

// skuNumber returns the collector number segment of a catalog sku,
// which keeps the variant letter the product's number field drops.
func skuNumber(sku string) string {
	fields := strings.Split(sku, "-")
	if len(fields) < 4 {
		return ""
	}
	return fields[3]
}

// fabPrintRunMarkers are the print-run suffixes Star City Games glues onto a
// Flesh and Blood set code in its skus, listed longest first so the trim takes
// the whole marker. The datastore numbers a card with the bare code and crosses
// the run with the treatment instead, so "ARC1036" is its ARC036.
var fabPrintRunMarkers = []string{"12", "1", "2", "U"}

// fabNumbers returns the collector numbers to try for a Flesh and Blood sku,
// most specific first.
//
// The sku carries the datastore's own number split across two segments:
// "SGL-FAB-BVO-005-ENN" is BVO005, which names exactly one printing, while the
// bare "005" is every set's fifth card. That matters because the sets Star City
// Games sells these under are decks and blister packs whose names match no
// datastore set, so nothing else narrows the candidates and the reprints alias.
//
// A number segment that opens with a code of its own ("HER_001" under the
// catch-all PRM) already names its set, and the sku's set segment is only the
// shelf it sits on. The fused pairs join with the separator the datastore
// spells them with, and the letter parts ("019_CC") stay wording.
//
// The bare number stays last rather than being dropped: it is what resolves the
// listings whose set segment is neither a datastore code nor a marked-up one,
// and keeping it last costs nothing since a prefixed number that resolves is
// always the more specific answer.
func fabNumbers(sku string) []string {
	number := skuNumber(sku)
	bare := strings.ReplaceAll(number, "_", " ")
	if number == "" {
		return []string{bare}
	}

	code := skuSetCode(sku)
	parts := strings.Split(number, "_")
	if len(parts) > 1 && isAllLetters(parts[0]) {
		code, parts = parts[0], parts[1:]
	}

	var candidates []string
	for _, prefix := range []string{code, trimPrintRun(code)} {
		candidate := prefixNumber(prefix, parts)
		if candidate != "" && candidate != bare && !slices.Contains(candidates, candidate) {
			candidates = append(candidates, candidate)
		}
	}
	return append(candidates, bare)
}

// prefixNumber glues the set code onto every numeric part of a sku's number
// segment, pairing them the way the datastore does ("MST002//MST158") and
// leaving the non-numeric parts as the wording they are.
func prefixNumber(code string, parts []string) string {
	if code == "" {
		return ""
	}
	var numbers, words []string
	for _, part := range parts {
		if part != "" && part[0] >= '0' && part[0] <= '9' {
			numbers = append(numbers, code+part)
			continue
		}
		words = append(words, part)
	}
	if len(numbers) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.Join(numbers, " // ") + " " + strings.Join(words, " "))
}

// trimPrintRun removes a print-run marker from a sku's set code. The trim is
// only ever a second guess, and it never shortens a code past the three
// letters every Flesh and Blood code is at least: "KSU", "NUU" and "UZU" are
// whole codes that end in a marker letter, so the untrimmed code has to be
// tried first and what the trim leaves has to still look like a code.
func trimPrintRun(code string) string {
	for _, marker := range fabPrintRunMarkers {
		trimmed := strings.TrimSuffix(code, marker)
		if trimmed != code && len(trimmed) >= 3 {
			return trimmed
		}
	}
	return code
}

func isAllLetters(field string) bool {
	if field == "" {
		return false
	}
	for _, r := range field {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func resolveProductID(game int, p CatalogProduct) (string, error) {
	// Duel Masters crossover promos are catalogued under Magic but aren't Magic
	// cards, so there's nothing to match; discard them.
	if strings.Contains(p.Name, "(Duel Masters)") {
		return "", mtgmatcher.ErrUnsupported
	}

	foil := catalogFoil(p)
	etched := strings.Contains(strings.ToLower(p.Finish), "etched")

	// Portal printed two versions of six cards. SCG marks them a and b in
	// the sku while mtgjson numbers the second with a d suffix, and sends
	// the same collector number and the same scryfall id for both - an id
	// that names the first version - so the b product has to be steered by
	// its number before the identifiers get a say.
	if game == GameMagic && p.Set == "Portal" && strings.HasSuffix(skuNumber(p.SKU), "b") {
		number := strings.TrimSuffix(skuNumber(p.SKU), "b") + "d"
		if out := mtgmatcher.MatchWithNumber(p.Name, "POR", number); len(out) == 1 {
			return mtgmatcher.MatchID(out[0].UUID, foil, etched)
		}
	}

	// Duel Decks: Anthology reprints four earlier duel decks, and mtgjson
	// keeps them under their original codes. The product's set name says
	// only "Anthology", so the deck it belongs to is read from the sku.
	if game == GameMagic && p.Set == "Duel Decks: Anthology" {
		number := strings.TrimLeft(p.CollectorNumber, "0")
		if out := mtgmatcher.MatchWithNumber(p.Name, skuSetCode(p.SKU), number); len(out) == 1 {
			return mtgmatcher.MatchID(out[0].UUID, foil, etched)
		}
	}

	// The authoritative identifiers resolve directly through the identifier
	// index, regardless of game: Scryfall id first, then the TCGplayer id
	// (MatchID resolves a bare product id through the external-id index and
	// applies the finish exactly like the scryfall path). Etched is the only
	// alt-foil that changes the printing; every other alt-foil shares the plain
	// foil's id. (SCG sends null ids today, so in practice this fires only once
	// they start populating them.)
	for _, id := range []string{p.ScryfallID, p.TCGPlayerID} {
		if id == "" {
			continue
		}
		if out, err := mtgmatcher.MatchID(id, foil, etched); err == nil {
			return out, nil
		}
	}

	// Magic needs catalog-specific fixups before the generic matcher.
	if game == GameMagic {
		// SCG's "-WAR2-" is the War of the Spark Japanese planeswalker
		// (jpwalker), whose Japanese-language Scryfall id isn't in the index and
		// which preprocess rejects as non-english. It maps to WAR #NNN★.
		if strings.Contains(p.SKU, "-WAR2-") {
			num := strings.TrimLeft(p.CollectorNumber, "0") + "★"
			if out := mtgmatcher.MatchWithNumber(p.Name, "WAR", num); len(out) == 1 {
				if id, err := mtgmatcher.MatchID(out[0].UUID, foil, false); err == nil {
					return id, nil
				}
			}
		}

		card, err := preprocess(catalogHit(p, foil))
		if err != nil {
			return "", err
		}
		// Inherently foreign sets (Foreign Black Border, Rinascimento, ...)
		// store the foreign printing as their canonical card, so a resolved id
		// whose primary language isn't English is the right match. Match's
		// English-only language validation would reject it, so use it directly.
		// English-primary cards fall through so a foreign single isn't wrongly
		// collapsed onto the English printing.
		if card.ID != "" {
			if co, e := mtgmatcher.GetUUID(card.ID); e == nil && co.Language != "" && co.Language != "English" {
				return mtgmatcher.MatchID(card.ID, foil, etched)
			}
		}
		return mtgmatcher.Match(card)
	}

	// Flesh and Blood reads its number off the sku instead: the segments
	// keep the set code, the fused-card pair ("077_112"), the promo-pack
	// prefix ("JDG_001") and the variant letter ("155b") that the product's
	// bare number field drops. The candidates run most specific first, so
	// the bare number only decides what the set-prefixed one could not.
	// The catalog's own finish name rides beside the flag: a product is one
	// printing in one treatment, and only the name says which.
	if game == GameFleshAndBlood {
		edition, finish := fabPrintRun(p.Set, p.Finish)
		var err error
		for _, number := range fabNumbers(p.SKU) {
			var id string
			id, err = mtgmatcher.Match(&mtgmatcher.InputCard{
				Name:      p.Name,
				Edition:   edition,
				Variation: number,
				Finish:    finish,
				Foil:      foil,
			})
			if err == nil {
				return id, nil
			}
		}
		return "", err
	}

	// The other games (Lorcana, Riftbound) identify a card by name +
	// collector number + finish; the catalog set narrows
	// same-name-and-number collisions across sets.
	return mtgmatcher.Match(&mtgmatcher.InputCard{
		Name:      p.Name,
		Edition:   p.Set,
		Variation: p.CollectorNumber,
		Foil:      foil,
	})
}

// fabPrintRun moves the print run out of a Flesh and Blood set name and into
// the finish. The catalog spells the run as part of the set, "Tales of Aria
// (1st Edition)", where the datastore keeps one set and crosses the run with
// the treatment, so the two runs of a card are two printings and the set name
// alone reaches neither in particular. A set without a run is left as it is.
func fabPrintRun(set, finish string) (string, string) {
	var run string
	switch {
	case strings.HasSuffix(set, " (1st Edition)"):
		set, run = strings.TrimSuffix(set, " (1st Edition)"), "1st Edition"
	case strings.HasSuffix(set, " (Unlimited)"):
		set, run = strings.TrimSuffix(set, " (Unlimited)"), "Unlimited Edition"
	default:
		return set, finish
	}
	// The catalog names the plain treatment for what it is not; the
	// datastore names it for what it is.
	if finish == "" || finish == "Non-foil" {
		finish = "Normal"
	}
	return set, run + " " + finish
}

// catalogCondition maps a catalog condition string to an mtgban grade.
func catalogCondition(condition string) (string, error) {
	switch condition {
	case "Near Mint":
		return "NM", nil
	case "Played":
		return "SP", nil
	case "Heavily Played":
		return "HP", nil
	default:
		return "", fmt.Errorf("unknown condition %q", condition)
	}
}
