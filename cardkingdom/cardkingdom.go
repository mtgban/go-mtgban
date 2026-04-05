package cardkingdom

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-cardkingdom"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type Cardkingdom struct {
	LogCallback mtgban.LogCallbackFunc
	Partner     string
	PreserveOOS bool

	localPath     string
	inventoryDate time.Time
	buylistDate   time.Time

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord
}

func NewScraperLocal(localPath string) *Cardkingdom {
	ck := Cardkingdom{}
	ck.inventory = mtgban.InventoryRecord{}
	ck.buylist = mtgban.BuylistRecord{}
	ck.localPath = localPath
	return &ck
}

func NewScraper() *Cardkingdom {
	ck := Cardkingdom{}
	ck.inventory = mtgban.InventoryRecord{}
	ck.buylist = mtgban.BuylistRecord{}
	return &ck
}

func (ck *Cardkingdom) printf(format string, a ...interface{}) {
	if ck.LogCallback != nil {
		ck.LogCallback("[CK] "+format, a...)
	}
}

func (ck *Cardkingdom) Load(ctx context.Context) error {
	link := ck.localPath
	if link == "" {
		link = cardkingdom.PricelistURL
	}
	pricelist, _, err := cardkingdom.Pricelist(ctx, nil, link)
	if err != nil {
		return err
	}

	ck.printf("Found %d prices", len(pricelist))

	start := time.Now()

	for _, card := range pricelist {
		skipErrors := card.Edition == "Mystery Booster/The List"

		theCard, err := Preprocess(card)
		if err != nil {
			if !errors.Is(err, mtgmatcher.ErrUnsupported) {
				ck.printf("%v", err)
			}
			continue
		}

		cardId, err := mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			continue
		} else if err != nil {
			ogErr := err
			cardId, err = mtgmatcher.MatchId(card.ScryfallID, theCard.Foil, strings.Contains(card.Variation, "Etched"))
			if err != nil {
				if skipErrors {
					continue
				}
				ck.printf("%v", ogErr)
				ck.printf("%q", theCard)
				ck.printf("%q", card)

				var alias *mtgmatcher.AliasingError
				if errors.As(err, &alias) {
					probes := alias.Probe()
					for _, probe := range probes {
						card, _ := mtgmatcher.GetUUID(probe)
						ck.printf("- %s", card)
					}
				}
				continue
			}
		}

		u, _ := url.Parse("https://www.cardkingdom.com/")

		u.Path = card.URL
		if ck.Partner != "" {
			q := u.Query()
			q.Set("partner", ck.Partner)
			q.Set("utm_source", ck.Partner)
			q.Set("utm_medium", "affiliate")
			q.Set("utm_campaign", ck.Partner)
			u.RawQuery = q.Encode()
		}
		link := u.String()

		retailPrices := []float64{
			card.ConditionValues.NmPrice, card.ConditionValues.ExPrice, card.ConditionValues.VgPrice, card.ConditionValues.GPrice,
		}
		// For newly added cards the nmPrice may not be initialized, so we the
		// root price value that is known to be valid (and skip card if data is
		// missing, since it gets used in the buylist step).
		if retailPrices[0] == 0 {
			retailPrices[0] = card.PriceRetail
		}
		if retailPrices[0] == 0 {
			continue
		}

		if card.QtyRetail > 0 && card.PriceRetail > 0 {
			qtys := []int{
				card.ConditionValues.NmQty, card.ConditionValues.ExQty, card.ConditionValues.VgQty, card.ConditionValues.GQty,
			}

			for i, cond := range mtgban.DefaultGradeTags {
				if qtys[i] == 0 {
					continue
				}

				out := &mtgban.InventoryEntry{
					Conditions: cond,
					Price:      retailPrices[i],
					Quantity:   qtys[i],
					URL:        link,
					OriginalId: strconv.Itoa(card.ID),
					InstanceId: card.Sku,
				}
				if cond != "NM" {
					out.CustomFields = map[string]string{
						"RetailPrice": fmt.Sprint(card.PriceRetail),
					}
				}
				err = ck.inventory.AddUnique(cardId, out)
			}
		} else if ck.PreserveOOS {
			// Only save URL information
			out := &mtgban.InventoryEntry{
				URL: link,
			}
			err = ck.inventory.AddUnique(cardId, out)
		}
		if err != nil && !skipErrors {
			ck.printf("%v", err)
		}

		u, _ = url.Parse("https://www.cardkingdom.com/purchasing/mtg_singles")
		if card.QtyBuying > 0 {
			q := u.Query()
			q.Set("filter[search]", "mtg_advanced")

			// Include as much detail as possible in the name
			cardName := card.Name
			if card.Variation != "" {
				cardName = fmt.Sprintf("%s (%s)", card.Name, card.Variation)
			}
			q.Set("filter[name]", strings.ReplaceAll(cardName, "\"", ""))

			// Always show both non-foil and foil cards, the filtering
			// on the website accurate enough (ie for Prerelease)
			q.Set("filter[singles]", "1")
			q.Set("filter[foils]", "1")

			// Edition is derived from the url itself, not the edition field
			urlPaths := strings.Split(card.URL, "/")
			if len(urlPaths) > 2 {
				q.Set("filter[edition]", urlPaths[1])
			}
			if ck.Partner != "" {
				q.Set("partner", ck.Partner)
				q.Set("utm_source", ck.Partner)
				q.Set("utm_medium", "affiliate")
				q.Set("utm_campaign", ck.Partner)
			}
			u.RawQuery = q.Encode()

			for i, grade := range mtgban.DefaultGradeTags {
				// Apply the same downgrade present in the retail prices since
				// there are a class of cards where CK downgrades price differently,
				// as if they were nonfoil
				buyPrice := card.PriceBuy * retailPrices[i] / retailPrices[0]

				var priceRatio float64

				if card.PriceRetail > 0 {
					priceRatio = card.PriceBuy / card.PriceRetail * 100
				}

				out := &mtgban.BuylistEntry{
					Conditions: grade,
					BuyPrice:   buyPrice,
					Quantity:   card.QtyBuying,
					PriceRatio: priceRatio,
					URL:        u.String(),
					OriginalId: strconv.Itoa(card.ID),
					InstanceId: card.Sku,
				}
				// Add the line entry as needed by the csv import
				if grade == "NM" {
					out.CustomFields = map[string]string{
						"CKTitle":   cardName,
						"CKEdition": card.Edition,
						"CKFoil":    strconv.FormatBool(card.IsFoil),
						"CKSKU":     card.Sku,
						"CKID":      strconv.Itoa(card.ID),
					}
				}
				err = ck.buylist.Add(cardId, out)
				if err != nil && !skipErrors {
					ck.printf("%v", err)
				}
			}
		}
	}

	ck.printf("Took %v", time.Since(start))

	ck.inventoryDate = time.Now()
	ck.buylistDate = time.Now()

	return nil
}

func (ck *Cardkingdom) Inventory() mtgban.InventoryRecord {
	return ck.inventory
}

func (ck *Cardkingdom) Buylist() mtgban.BuylistRecord {
	return ck.buylist
}

func (ck *Cardkingdom) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Kingdom"
	info.Shorthand = "CK"
	info.InventoryTimestamp = &ck.inventoryDate
	info.BuylistTimestamp = &ck.buylistDate
	info.CreditMultiplier = 1.3
	return
}
