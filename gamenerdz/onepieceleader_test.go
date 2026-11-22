package gamenerdz

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"

	_ "github.com/mtgban/go-mtgban/matcher/onepiece"
)

// codelessLeaderDatastore holds the published rows of two promo leaders the
// catalog gives no collector number and of the trophy card it gives none
// either, of a leader Extra Booster Vol. 2 reprints at its own code, and
// of a starter deck card with a promo printing at the same code.
// NUMBER and STEM spell the leaders in the datastore's two shapes:
// filed under "LEADER" (leader_552137), or with no number on the bare
// product id (552137).
const codelessLeaderDatastore = `{"data": {
 "game": "onepiece",
 "sets": {
  "EB05": {"name": "Extra Booster: One Piece Heroines Edition Vol. 2", "releaseDate": "2026-10-30"},
  "OP11": {"name": "A Fist of Divine Speed", "releaseDate": "2025-06-06"},
  "OP-PR": {"name": "One Piece Promotion Cards", "releaseDate": "2022-09-30", "type": "promo"},
  "ST-03": {"name": "ST-03: Starter Deck 3 The Seven Warlords of The Sea", "releaseDate": "2022-12-02"},
  "OP13-ANN": {"name": "Carrying On His Will: 3rd Anniversary Tournament Cards", "releaseDate": "2025-10-31"}
 },
 "cards": [
  {"color": "Blue;Green;Purple;Red;Black;Yellow", "colors": ["Blue", "Green", "Purple", "Red", "Black", "Yellow"], "externalLinks": {"tcgPlayerId": 552137}, "finish": "Normal", "id": "STEM552137", NUMBER"name": "Monkey.D.Luffy", "originalReleaseDate": "2024-01-01", "promoTypes": ["sealedbattle"], "rarity": "Leader", "setCode": "OP-PR", "type": "Leader", "variant": "Sealed Battle 2024 Vol. 2", "watermark": "vol. 2"},
  {"color": "Blue;Green;Purple;Red;Black;Yellow", "colors": ["Blue", "Green", "Purple", "Red", "Black", "Yellow"], "externalLinks": {"tcgPlayerId": 552137}, "finish": "Foil", "id": "STEM552137_foil", NUMBER"name": "Monkey.D.Luffy", "originalReleaseDate": "2024-01-01", "promoTypes": ["sealedbattle"], "rarity": "Leader", "setCode": "OP-PR", "type": "Leader", "variant": "Sealed Battle 2024 Vol. 2", "watermark": "vol. 2"},
  {"color": "Blue;Green;Purple;Red;Black;Yellow", "colors": ["Blue", "Green", "Purple", "Red", "Black", "Yellow"], "externalLinks": {"tcgPlayerId": 661879}, "finish": "Normal", "id": "STEM661879", NUMBER"name": "Monkey.D.Luffy", "promoTypes": ["anniversarytournament", "3brotherspack"], "rarity": "Leader", "setCode": "OP13-ANN", "type": "Leader", "variant": "3rd Anniversary Tournament 3 Brothers Pack", "watermark": "3rd"},
  {"externalLinks": {"tcgPlayerId": 632406}, "finish": "Foil", "id": "op11-041_632406_foil", "name": "Nami", "number": "OP11-041", "promoTypes": ["alternateart"], "rarity": "Leader", "setCode": "OP11", "type": "Leader", "variant": "Alternate Art"},
  {"color": "Blue;Yellow", "colors": ["Blue", "Yellow"], "externalLinks": {"tcgPlayerId": 724683}, "finish": "Foil", "id": "op11-041_724683_foil", "name": "Nami", "number": "OP11-041", "promoTypes": ["heroineleader"], "rarity": "Leader", "setCode": "EB05", "type": "Leader", "variant": "Heroine Leader"},
  {"color": "Blue", "colors": ["Blue"], "externalLinks": {"bandaiId": "ST03-004", "tcgPlayerId": 288270}, "finish": "Normal", "id": "st03-004_288270", "image": "https://static.dotgg.gg/onepiece/card/ST03-004.webp", "name": "Gecko Moria", "number": "ST03-004", "rarity": "Common", "setCode": "ST-03", "type": "Character"},
  {"color": "Blue", "colors": ["Blue"], "externalLinks": {"tcgPlayerId": 503022}, "finish": "Foil", "id": "st03-004_503022_foil", "image": "https://tcgplayer-cdn.tcgplayer.com/product/503022_400w.jpg", "name": "Gecko Moria", "number": "ST03-004", "originalReleaseDate": "2023-01-01", "promoTypes": ["storechampionship", "participationpack"], "rarity": "Promo", "setCode": "OP-PR", "type": "Character", "variant": "Store Championship 2023 Participation Pack Vol. 1", "watermark": "vol. 1"},
  {"externalLinks": {"tcgPlayerId": 719668}, "finish": "Foil", "id": "719668_foil", "name": "Flame-Flame Fruit Trophy Card", "promoTypes": ["flameflamefruitcoliseum"], "rarity": "None", "setCode": "OP-PR", "variant": "Flame-Flame Fruit Coliseum"}
 ]
}}`

// TestCodelessLeader pins that a listing with no card code is kept where it
// lands on a promo leader with no collector number, in either shape the
// datastore has filed them, and refused where it lands on any other card
// with none, or where its wording lands outside a shelf that prints the card
// at its code, unless its retail id placed it there.
func TestCodelessLeader(t *testing.T) {
	for _, shape := range []struct{ desc, number, stem string }{
		{"filed under LEADER", `"number": "LEADER", `, "leader_"},
		{"filed with no number", "", ""},
	} {
		raw := strings.NewReplacer("NUMBER", shape.number, "STEM", shape.stem).Replace(codelessLeaderDatastore)
		b, err := matcher.Open("onepiece", strings.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		gn, err := NewScraper(b)
		if err != nil {
			t.Fatal(err)
		}
		for _, tt := range []struct {
			name, shelf, finish, want string
			tcgID                     int64
		}{
			{"Monkey.D.Luffy (Sealed Battle 2024 Vol. 2)", "One Piece Promotion Cards", "Normal", shape.stem + "552137", 0},
			{"Monkey.D.Luffy (3rd Anniversary Tournament 3 Brothers Pack)", "Carrying On His Will: 3rd Anniversary Tournament Cards", "Normal", shape.stem + "661879", 0},
			{"Flame-Flame Fruit Trophy Card (Flame-Flame Fruit Coliseum)", "One Piece Promotion Cards", "Foil", "", 0},
			{"Nami (OP11-041) (Alternate Art) (OP11-041) Extra Booster: One Piece Heroines Edition Vol. 2 Foil", "Extra Booster: One Piece Heroines Edition Vol. 2", "Foil", "", 0},
			{"Gecko Moria (ST03-004) Starter Deck 3: The Seven Warlords of The Sea Foil", "Starter Deck 3: The Seven Warlords of The Sea", "Foil", "st03-004_503022_foil", 503022},
		} {
			product := GNProduct{
				DisplayName:    tt.name,
				SelectedFinish: tt.finish,
				ProductData:    GNProductData{SetName: tt.shelf},
			}
			got, err := resolveOnePiece(gn, "", product, tt.tcgID)
			if err != nil || got != tt.want {
				t.Errorf("%s: resolveOnePiece(%q) = (%q, %v), want %q", shape.desc, tt.name, got, err, tt.want)
			}
		}
	}
}
