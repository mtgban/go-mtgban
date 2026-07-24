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

// CatalogProduct is a single card printing in the catalog export.
type CatalogProduct struct {
	ID              int              `json:"id"`
	SKU             string           `json:"sku"`
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
func gameFromCatalog(game string) int {
	switch game {
	case "Magic: The Gathering":
		return GameMagic
	case "Lorcana":
		return GameLorcana
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
	foil := catalogFoil(p)

	switch game {
	case GameMagic:
		etched := strings.Contains(strings.ToLower(p.Finish), "etched")
		if p.ScryfallID != "" {
			if id, err := mtgmatcher.MatchId(p.ScryfallID, foil, etched); err == nil {
				return id, nil
			}
		}

		// The TCGplayer id is the next authoritative identifier: MatchId resolves
		// a bare product id through the external-id index and applies the finish,
		// exactly like the scryfall path. Use it whenever the scryfall id is
		// absent or didn't resolve. (SCG sends null today; this future-proofs it.)
		if p.TCGPlayerID != "" {
			if id, err := mtgmatcher.MatchId(p.TCGPlayerID, foil, etched); err == nil {
				return id, nil
			}
		}

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
	case GameLorcana:
		return mtgmatcher.SimpleSearch(p.Name, strings.TrimLeft(p.CollectorNumber, "0"), foil)
	default:
		return "", mtgmatcher.ErrUnsupported
	}
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
