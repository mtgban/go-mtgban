package cardmarket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// IDMapProduct is one product of the published catalog: what the marketplace
// calls it, where it files it, and the printings it stands for.
type IDMapProduct struct {
	ExpansionID int      `json:"expansionId"`
	Name        string   `json:"name"`
	Number      string   `json:"number,omitempty"`
	UUIDs       []string `json:"uuids,omitempty"`
}

// IDMapExpansion names one expansion of the catalog and the set codes it
// covers.
type IDMapExpansion struct {
	Name     string   `json:"name"`
	SetCodes []string `json:"setCodes,omitempty"`
}

// IDMap is the published Cardmarket catalog: every singles product mapped to
// the printings it sells, which replaces walking the API for the same
// answer. MTGJSON publishes the Magic one; the other games' are built by the
// same crawl this map retires, run once against each game's datastore.
type IDMap struct {
	Products   map[int]IDMapProduct
	Expansions map[int]IDMapExpansion
}

// LoadIDMap reads a published catalog. The file keys both tables by the
// stringified ids JSON forces on it; the map keys them by the numbers they
// are.
func LoadIDMap(reader io.Reader) (*IDMap, error) {
	var payload struct {
		Data struct {
			Expansions map[string]IDMapExpansion `json:"expansions"`
			Products   map[string]IDMapProduct   `json:"products"`
		} `json:"data"`
	}
	err := json.NewDecoder(reader).Decode(&payload)
	if err != nil {
		return nil, err
	}
	if len(payload.Data.Products) == 0 {
		return nil, errors.New("empty id map")
	}

	out := IDMap{
		Products:   make(map[int]IDMapProduct, len(payload.Data.Products)),
		Expansions: make(map[int]IDMapExpansion, len(payload.Data.Expansions)),
	}
	for key, product := range payload.Data.Products {
		id, err := strconv.Atoi(key)
		if err != nil {
			return nil, fmt.Errorf("product id %q: %w", key, err)
		}
		out.Products[id] = product
	}
	for key, expansion := range payload.Data.Expansions {
		id, err := strconv.Atoi(key)
		if err != nil {
			return nil, fmt.Errorf("expansion id %q: %w", key, err)
		}
		out.Expansions[id] = expansion
	}
	return &out, nil
}

// resolveUUIDs answers a product from the printings its map entry lists,
// splitting them by finish the way the guide's columns are split. Ids the
// datastore does not carry are passed over - a double-faced card lists its
// back face too, and the index knows only fronts. Within a finish the
// printing whose number agrees with the product's wins, the way Fallback
// already prefers it; a pick between printings the number cannot settle is
// said out loud. Both ids empty means the entry decided nothing.
func (mkm *Index) resolveUUIDs(product *MKMProduct, uuids []string) (string, string) {
	var plain, foil []string
	var plainMatched, foilMatched bool
	for _, uuid := range uuids {
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil {
			continue
		}
		sameNumber := strings.EqualFold(co.OriginalNumber, product.Number)
		if co.Foil || co.Etched {
			if sameNumber && !foilMatched {
				foil = append([]string{uuid}, foil...)
				foilMatched = true
			} else {
				foil = append(foil, uuid)
			}
		} else {
			if sameNumber && !plainMatched {
				plain = append([]string{uuid}, plain...)
				plainMatched = true
			} else {
				plain = append(plain, uuid)
			}
		}
	}

	if len(plain) > 1 && !plainMatched {
		mkm.printf("id %d %q lists %d plain printings and the number settles none; keeping %s",
			product.IDProduct, product.Name, len(plain), plain[0])
	}
	if len(foil) > 1 && !foilMatched {
		mkm.printf("id %d %q lists %d foil printings and the number settles none; keeping %s",
			product.IDProduct, product.Name, len(foil), foil[0])
	}

	var cardID, cardIDFoil string
	switch {
	case len(plain) > 0:
		cardID = plain[0]
		if len(foil) > 0 {
			cardIDFoil = foil[0]
		} else {
			// The entry lists no foil printing, but the datastore may
			// still carry one, the way the crawl probes for it.
			cardIDFoil, _ = mtgmatcher.MatchID(cardID, true)
		}
	case len(foil) > 0:
		// A foil-only product prices through its own columns; both ids
		// point to it, the way Fallback answers a single printing.
		cardID = foil[0]
		cardIDFoil = foil[0]
	}
	return cardID, cardIDFoil
}

// processMapped prices one product of the id map. The map answers first;
// what it left unmapped goes down the same ladder the crawl uses, so a
// product the file does not know yet is matched rather than lost.
func (mkm *Index) processMapped(channel chan<- responseChan, id int, mapped IDMapProduct, expansionName string) error {
	product := &MKMProduct{
		IDProduct:     id,
		Name:          mapped.Name,
		Number:        mapped.Number,
		ExpansionName: expansionName,
		// The map carries no listing counts, so each column reports one
		// copy, the way the TCGplayer index does; the crawl's counts were
		// the one thing the API alone knew.
		CountArticles: 2,
		CountFoils:    1,
	}

	cardID, cardIDFoil := mkm.resolveUUIDs(product, mapped.UUIDs)
	if cardID == "" {
		if mkm.gameID != GameMagic {
			return errNoPrinting
		}
		var err error
		cardID, cardIDFoil, err = mkm.resolveMagic(product)
		if err != nil || cardID == "" {
			return err
		}
	}

	return mkm.emitPrices(channel, product, cardID, cardIDFoil, false)
}

// loadOffline continues Load with the crawl replaced by the id map: the
// same tallies, the same output, and no API. The expansions come from the
// map too, so nothing from here on needs a credential.
func (mkm *Index) loadOffline(ctx context.Context) error {
	byExpansion := map[int][]int{}
	for id, product := range mkm.IDMap.Products {
		byExpansion[product.ExpansionID] = append(byExpansion[product.ExpansionID], id)
	}

	var items []MKMExpansion
	for expansionID := range byExpansion {
		name := mkm.IDMap.Expansions[expansionID].Name
		if name == "" {
			name = fmt.Sprintf("expansion %d", expansionID)
		}
		if mkm.TargetEdition != "" && name != mkm.TargetEdition {
			continue
		}
		items = append(items, MKMExpansion{IDExpansion: expansionID, Name: name})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].IDExpansion < items[j].IDExpansion })

	mkm.printf("Parsing %d expansion ids from the id map", len(items))

	walked, refused, unread := mkm.collectPrices(ctx, items,
		func(ctx context.Context, exp MKMExpansion, channel chan<- responseChan) error {
			mkm.printf("Processing %s (%d)", exp.Name, exp.IDExpansion)
			ids := byExpansion[exp.IDExpansion]
			sort.Ints(ids)

			var refused []string
			for _, id := range ids {
				mapped := mkm.IDMap.Products[id]
				err := mkm.processMapped(channel, id, mapped, exp.Name)
				switch {
				case errors.Is(err, errNoPrinting):
					refused = append(refused, fmt.Sprintf("%d %q (%s) in %s",
						id, mapped.Name, mapped.Number, exp.Name))
				case err != nil:
					mkm.printf("product id %d returned %s", id, err)
				}
			}

			mkm.reportRefused(exp.Name, len(ids), refused)
			channel <- responseChan{tally: true, walked: len(ids), refused: len(refused)}
			return nil
		})

	mkm.printf("Walked %d products, %d of which named no printing of ours", walked, refused)
	if unread > 0 {
		mkm.printf("%d of %d expansions never answered, and none of their products is in that count", unread, len(items))
	}
	mkm.inventoryDate = time.Now()
	return nil
}
