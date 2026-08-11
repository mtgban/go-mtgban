package starcitygames

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// scgCatalogURL is the HawkSearch catalog export. It returns the full product
// catalog (both retail price/qty and buylist sell_list_price per variant) as a
// single JSON array, authenticated with an x-api-key header.
const scgCatalogURL = "https://api.starcitygames.com/hawksearch/catalog/download/json"

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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, scgCatalogURL, http.NoBody)
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
// MatchId, which associates the foil id with its etched sibling. Every other
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
			return mtgmatcher.MatchId(out[0].UUID, foil, etched)
		}
	}

	// Duel Decks: Anthology reprints four earlier duel decks, and mtgjson
	// keeps them under their original codes. The product's set name says
	// only "Anthology", so the deck it belongs to is read from the sku.
	if game == GameMagic && p.Set == "Duel Decks: Anthology" {
		number := strings.TrimLeft(p.CollectorNumber, "0")
		if out := mtgmatcher.MatchWithNumber(p.Name, skuSetCode(p.SKU), number); len(out) == 1 {
			return mtgmatcher.MatchId(out[0].UUID, foil, etched)
		}
	}

	// The authoritative identifiers resolve directly through the identifier
	// index, regardless of game: Scryfall id first, then the TCGplayer id
	// (MatchId resolves a bare product id through the external-id index and
	// applies the finish exactly like the scryfall path). Etched is the only
	// alt-foil that changes the printing; every other alt-foil shares the plain
	// foil's id. (SCG sends null ids today, so in practice this fires only once
	// they start populating them.)
	for _, id := range []string{p.ScryfallID, p.TCGPlayerID} {
		if id == "" {
			continue
		}
		if out, err := mtgmatcher.MatchId(id, foil, etched); err == nil {
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
				if id, err := mtgmatcher.MatchId(out[0].UUID, foil, false); err == nil {
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
		if card.Id != "" {
			if co, e := mtgmatcher.GetUUID(card.Id); e == nil && co.Language != "" && co.Language != "English" {
				return mtgmatcher.MatchId(card.Id, foil, etched)
			}
		}
		return mtgmatcher.Match(card)
	}

	// Flesh and Blood reads its number off the sku instead: the segment
	// keeps the fused-card pair ("077_112"), the promo-pack prefix
	// ("JDG_001") and the variant letter ("155b") that the product's bare
	// number field drops. The underscores become spaces so the matcher's
	// number extraction reads the leading code and the rest stays wording.
	// The finish is inert there - one product is one printing - but the
	// flag rides along like everywhere else.
	if game == GameFleshAndBlood {
		return mtgmatcher.Match(&mtgmatcher.InputCard{
			Name:      p.Name,
			Edition:   p.Set,
			Variation: strings.ReplaceAll(skuNumber(p.SKU), "_", " "),
			Foil:      foil,
		})
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
