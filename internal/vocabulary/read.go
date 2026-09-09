package vocabulary

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// ErrNotDatastore says a file is not a built datastore. The games whose
// upstream is itself a JSON document keep that document under the same
// name, and reading one reports an empty vocabulary rather than saying it
// read the wrong thing.
var ErrNotDatastore = errors.New("no cards: this is not a built datastore")

// aside are the fields a loader may not make a token out of.
//
// The variant is prose - the label the catalog wrote, which the promo types
// and the mark are distilled out of - so a token derived from it is a token
// the datastore never stated. That derivation is the bug this is looking
// for: it has been written from scratch in five loaders. Everything else a
// datastore states is a fact a printing can answer a listing with, and a
// loader is free to carry it without declaring it.
var aside = map[string]bool{"variant": true, "id": true, "image": true, "images": true}

// marking is the field whose presence says a builder has taken the facts out
// of the variant and put them in fields of their own. Until it does, a
// loader deriving a token from the variant is falling back the way it is
// documented to, and the variant counts as something the datastore states.

// ReadPublished reads what a datastore states, without the loader's help.
func ReadPublished(path string) (Published, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Published{}, err
	}
	var payload struct {
		Cards []map[string]any `json:"cards"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return Published{}, err
	}
	if len(payload.Cards) == 0 {
		return Published{}, fmt.Errorf("%s: %w", path, ErrNotDatastore)
	}
	var stated Published
	for _, card := range payload.Cards {
		if _, marked := card["watermark"]; marked {
			stated.Marked = true
		}
		walk(card, &stated)
	}
	if !stated.Marked {
		for _, card := range payload.Cards {
			if variant, prose := card["variant"].(string); prose {
				stated.Facts = append(stated.Facts, mtgmatcher.PromoTypeSlug(variant))
			}
		}
	}
	stated.Tokens, stated.Facts = sorted(stated.Tokens), sorted(stated.Facts)
	return stated, nil
}

// walk reads every fact a card states, however deep it states it. A game
// says its finishes in a nested array and its promotions with them, so a
// reader that stops at the top of the card sees neither.
func walk(held any, stated *Published) {
	switch value := held.(type) {
	case map[string]any:
		for key, one := range value {
			if aside[key] {
				continue
			}
			if key == "promoTypes" {
				for _, token := range scalars(one) {
					stated.Tokens = append(stated.Tokens, mtgmatcher.PromoTypeSlug(token))
				}
				continue
			}
			if key == "originalReleaseDate" {
				stated.Facts = append(stated.Facts, dateSaid(fmt.Sprint(one))...)
			}
			walk(one, stated)
		}
	case []any:
		for _, one := range value {
			walk(one, stated)
		}
	default:
		for _, said := range scalars(held) {
			if slug := mtgmatcher.PromoTypeSlug(said); slug != "" {
				stated.Facts = append(stated.Facts, slug)
			}
		}
	}
}

// scalars reads a field as the values it holds, which is one for a scalar
// and several for a list.
func scalars(value any) []string {
	switch held := value.(type) {
	case nil, map[string]any:
		return nil
	case []any:
		out := make([]string, 0, len(held))
		for _, one := range held {
			out = append(out, scalars(one)...)
		}
		return out
	case float64:
		if held == float64(int64(held)) {
			return []string{fmt.Sprintf("%d", int64(held))}
		}
	}
	return []string{fmt.Sprint(value)}
}

// dateSaid are the ways a published date can be written back into a
// printing's wording: the bare year where the label named only a year, and
// the month beside it where it named a month.
func dateSaid(published string) []string {
	if len(published) != 10 {
		return nil
	}
	year := published[:4]
	month := time.Month(int(published[5]-'0')*10 + int(published[6]-'0'))
	return []string{year, mtgmatcher.PromoTypeSlug(month.String() + " " + year)}
}

// ReadLoaded reads what a loader made of the same file.
func ReadLoaded(game, path string) (Backend, error) {
	b, err := datastore.Read(game, path)
	if err != nil {
		return Backend{}, err
	}
	loaded := Backend{
		Declared: b.AllPromoTypes,
		Labels:   map[string]string{},
	}
	for _, token := range b.AllPromoTypes {
		loaded.Labels[token] = b.PromoTypeLabels[token]
	}
	for _, uuid := range b.AllUUIDs {
		co, err := b.GetUUID(uuid)
		if err != nil || co.Sealed {
			continue
		}
		loaded.Worn = append(loaded.Worn, co.PromoTypes...)
	}
	loaded.Worn = sorted(loaded.Worn)
	return loaded, nil
}

// Games are the games built by the datastore generator, keyed by the
// variable naming where each one's file is. Magic is not among them: its
// datastore is MTGJSON's and its promo types are MTGJSON's too.
var Games = map[string]string{
	"fleshandblood": "FLESHANDBLOOD_PATH",
	"gundam":        "GUNDAM_PATH",
	"lorcana":       "LORCANA_PATH",
	"onepiece":      "ONEPIECE_PATH",
	"palworld":      "PALWORLD_PATH",
	"pokemon":       "POKEMON_PATH",
	"riftbound":     "RIFTBOUND_PATH",
	"yugioh":        "YUGIOH_PATH",
}

// GameNames are the games in a settled order, for a test to run through.
func GameNames() []string {
	names := make([]string, 0, len(Games))
	for game := range Games {
		names = append(names, game)
	}
	return sorted(names)
}

// PathOf is where a game's datastore is, empty where the run carries none.
func PathOf(game string) string {
	return strings.TrimSpace(os.Getenv(Games[game]))
}
