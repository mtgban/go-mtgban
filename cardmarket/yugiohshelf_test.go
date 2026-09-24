package cardmarket

import (
	"errors"
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/yugioh"
)

// yugiohShelfDatastore is the published Yu-Gi-Oh datastore cut down to the
// printings these tests turn on: Tri-Horned Dragon in both runs of the set
// the storefront names without its article, its European print in the set
// the catalog keeps for that run, and a Mega-Tin card numbered with the
// region infix the storefront leaves out.
//
// The two runs carry the product id they are two runs of, the way every
// entry the real datastore publishes does: it is what pairs them.
const yugiohShelfDatastore = `{"data": {
 "game": "yugioh",
 "sets": {
  "LOB": {"abbreviation": "LOB", "name": "The Legend of Blue Eyes White Dragon", "releaseDate": "2002-03-08"},
  "LOB-EN": {"abbreviation": "LOB-EN", "name": "Legend of Blue Eyes White Dragon (Worldwide English)", "releaseDate": "2002-03-08"},
  "MP18": {"abbreviation": "MP18", "name": "2018 Mega-Tins Mega Pack", "releaseDate": "2018-08-30"}
 },
 "cards": [
  {"externalLinks": {"tcgPlayerId": 22538}, "finish": "1st Edition", "id": "lob-000_22538_1stedition", "name": "Tri-Horned Dragon", "number": "LOB-000", "rarity": "Secret Rare", "setCode": "LOB"},
  {"externalLinks": {"tcgPlayerId": 22538}, "finish": "Unlimited", "id": "lob-000_22538_unlimited", "name": "Tri-Horned Dragon", "number": "LOB-000", "rarity": "Secret Rare", "setCode": "LOB"},
  {"finish": "Unlimited", "id": "lob-en000_1", "name": "Tri-Horned Dragon", "number": "LOB-EN000", "rarity": "Secret Rare", "setCode": "LOB-EN"},
  {"finish": "1st Edition", "id": "mp18-en065_1", "name": "Topologic Bomber Dragon", "number": "MP18-EN065", "rarity": "Prismatic Secret Rare", "setCode": "MP18"}
 ]
}}`

// TestMatchYugiohShelves pins that the name path reaches the sets the
// storefront names its own way, reads the run off the version index, puts
// the region infix back on a number, follows a European number to the set
// kept for that print, and says which kind of miss a miss is.
func TestMatchYugiohShelves(t *testing.T) {
	b := datastoreBackend(t, "yugioh", yugiohShelfDatastore)

	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}
	for _, tt := range []struct {
		expansion, name, number, want string
		err                           error
	}{
		{"Legend of Blue Eyes White Dragon", "Tri-Horned Dragon (V.1 - Secret Rare)", "000", "lob-000_22538_unlimited", nil},
		{"Legend of Blue Eyes White Dragon", "Tri-Horned Dragon (V.2 - Secret Rare)", "000", "lob-000_22538_unlimited", nil},
		{"Legend of Blue Eyes White Dragon", "Tri-Horned Dragon (V.4 - Secret Rare)", "EN000", "lob-en000_1", nil},
		{"Legend of Blue Eyes White Dragon", "Tri-Horned Dragon (V.3 - Secret Rare)", "A000", "", errForeign},
		{"2018 Mega-Tin Mega Pack", "Topologic Bomber Dragon", "065", "mp18-en065_1", nil},
		{"2018 Mega-Tin Mega Pack", "Gouki Re-Match", "070", "", errNoPrinting},
		{"Legend of Blue Eyes White Dragon (LDD)", "Tri-Horned Dragon", "000", "", errForeign},
	} {
		product := cm.Product{Name: tt.name, Number: tt.number, ExpansionName: tt.expansion}
		got, err := mkm.matchYugioh(&product)
		if got != tt.want || !errors.Is(err, tt.err) {
			t.Errorf("%q in %q (%s) = %q, %v; want %q, %v", tt.name, tt.expansion, tt.number, got, err, tt.want, tt.err)
		}
	}
}

// yugiohInfixDatastore is the published datastore cut down to two
// region-prefixed promos, each filed in a set other than the one its
// expansion names: Rise of Destiny's special edition and a Sneak Preview
// print of a Force of the Breaker card.
const yugiohInfixDatastore = `{"data": {
 "game": "yugioh",
 "sets": {
  "RDS": {"abbreviation": "RDS", "name": "Rise of Destiny", "releaseDate": "2004-12-03"},
  "RDS-275": {"abbreviation": "RDS-275", "name": "Rise of Destiny Special Edition", "releaseDate": "2005-02-01"},
  "FOTB": {"abbreviation": "FOTB", "name": "Force of the Breaker", "releaseDate": "2007-05-16"},
  "G284": {"abbreviation": "G284", "name": "Sneak Preview Series 3", "releaseDate": "2006-02-28"}
 },
 "cards": [
  {"externalLinks": {"tcgPlayerId": 23472}, "finish": "Limited", "id": "rds-ense1_23472_limited", "name": "Diffusion Wave-Motion", "number": "RDS-ENSE1", "rarity": "Ultra Rare", "setCode": "RDS-275"},
  {"externalLinks": {"tcgPlayerId": 26592}, "finish": "Limited", "id": "fotb-ensp1_26592_limited", "name": "Volcanic Rocket", "number": "FOTB-ENSP1", "rarity": "Super Rare", "setCode": "G284"}
 ]
}}`

// TestMatchYugiohInfixedPromos pins the region-infix retry that reaches a
// promo the datastore numbers "EN"+region+tail, in whichever set of the
// shelf actually carries the row.
func TestMatchYugiohInfixedPromos(t *testing.T) {
	b := datastoreBackend(t, "yugioh", yugiohInfixDatastore)

	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}
	for _, tt := range []struct {
		expansion, name, number, want string
		err                           error
	}{
		{"Rise of Destiny", "Diffusion Wave-Motion", "SE1", "rds-ense1_23472_limited", nil},
		{"Force of the Breaker", "Volcanic Rocket (V.2 - Super Rare)", "SP1", "fotb-ensp1_26592_limited", nil},
	} {
		product := cm.Product{Name: tt.name, Number: tt.number, ExpansionName: tt.expansion}
		got, err := mkm.matchYugioh(&product)
		if got != tt.want || !errors.Is(err, tt.err) {
			t.Errorf("%q in %q (%s) = %q, %v; want %q, %v", tt.name, tt.expansion, tt.number, got, err, tt.want, tt.err)
		}
	}
}

// yugiohEuropeanDatastore is the published datastore cut down to an OTS
// Tournament Pack 9 card and the number just past where its English pack
// ends, and Spell Ruler's last SRL number beside the Magic Ruler European
// print of the number that follows it.
const yugiohEuropeanDatastore = `{"data": {
 "game": "yugioh",
 "sets": {
  "OP09": {"abbreviation": "OP09", "name": "OTS Tournament Pack 9", "releaseDate": "2018-12-08"},
  "SRL": {"abbreviation": "SRL", "name": "Spell Ruler", "releaseDate": "2002-09-16"},
  "MRL": {"abbreviation": "MRL", "name": "Magic Ruler", "releaseDate": "2002-09-16"}
 },
 "cards": [
  {"externalLinks": {"tcgPlayerId": 181765}, "finish": "Unlimited", "id": "op09-en008_181765_unlimited", "name": "Card Destruction", "number": "OP09-EN008", "rarity": "Super Rare", "setCode": "OP09"},
  {"externalLinks": {"tcgPlayerId": 181783}, "finish": "Unlimited", "id": "op09-en026_181783_unlimited", "name": "Token: Mecha Phantom Beast - Dracossack", "number": "OP09-EN026", "rarity": "Super Rare", "setCode": "OP09"},
  {"externalLinks": {"tcgPlayerId": 120691}, "finish": "Unlimited", "id": "srl-103_120691_unlimited", "name": "Serpent Night Dragon", "number": "SRL-103", "rarity": "Secret Rare", "setCode": "SRL"},
  {"externalLinks": {"tcgPlayerId": 229417}, "finish": "Unlimited", "id": "mrl-e129_229417_unlimited", "name": "Pot of Greed", "number": "MRL-E129", "rarity": "Rare", "setCode": "MRL"}
 ]
}}`

// TestMatchYugiohEuropeanPrints pins the OTS Tournament Pack rule that
// tells the Portuguese packs' extra numbers apart from a real gap, and the
// Spell Ruler rule that tells its European renumbering apart from one.
func TestMatchYugiohEuropeanPrints(t *testing.T) {
	b := datastoreBackend(t, "yugioh", yugiohEuropeanDatastore)

	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}
	for _, tt := range []struct {
		expansion, name, number, want string
		err                           error
	}{
		{"OTS Tournament Pack 9", "Card Destruction", "008", "op09-en008_181765_unlimited", nil},
		{"OTS Tournament Pack 9", "Scrap Chimera", "028", "", errForeign},
		{"Spell Ruler", "Pot of Greed (V.1 - Rare)", "129", "", errTwin},
		{"Spell Ruler", "Beaver Warrior (V.1 - Common)", "103", "", errNoPrinting},
	} {
		product := cm.Product{Name: tt.name, Number: tt.number, ExpansionName: tt.expansion}
		got, err := mkm.matchYugioh(&product)
		if got != tt.want || !errors.Is(err, tt.err) {
			t.Errorf("%q in %q (%s) = %q, %v; want %q, %v", tt.name, tt.expansion, tt.number, got, err, tt.want, tt.err)
		}
	}
}

func TestYugiohSameProduct(t *testing.T) {
	for _, tt := range []struct {
		a, b cm.Product
		want bool
	}{
		{cm.Product{Name: "Feral Imp (V.1 - Common)", Number: "001"}, cm.Product{Name: "Feral Imp (V.2 - Common)", Number: "001"}, true},
		{cm.Product{Name: "Feral Imp (V.3 - Common)", Number: "EN001"}, cm.Product{Name: "Feral Imp (V.2 - Common)", Number: "001"}, true},
		{cm.Product{Name: "Tri-Horned Dragon", Number: ""}, cm.Product{Name: "Tri-Horned Dragon", Number: ""}, true},
		{cm.Product{Name: "Harpie Lady (V.1 - Common)", Number: "008"}, cm.Product{Name: "Harpie Lady Sisters (V.1 - Super Rare)", Number: "009"}, false},
		{cm.Product{Name: "Griggle (V.2 - Common)", Number: "016"}, cm.Product{Name: "Rescue-ACE Monitor", Number: "279"}, false},
	} {
		if got := yugiohSameProduct(&tt.a, &tt.b); got != tt.want {
			t.Errorf("%q/%q vs %q/%q = %v, want %v", tt.a.Name, tt.a.Number, tt.b.Name, tt.b.Number, got, tt.want)
		}
	}
}

func TestYugiohEditions(t *testing.T) {
	for _, tt := range []struct {
		expansion string
		want      string
	}{
		{"Duelist League 18", "DL18"},
		{"Champion Pack: Game Three", "CP03"},
		{"Astral Pack Eight", "AP08"},
		{"Collector's Tins 2010", "2010 Collectors Tin"},
		{"2017 Mega-Tin Mega Pack", "2017 Mega-Tins Mega Pack"},
		{"Legend of Blue Eyes White Dragon", "LOB"},
		{"Spell Ruler (SDM)", "Spell Ruler (SDM)"},
	} {
		if got := yugiohEditions(tt.expansion)[0]; got != tt.want {
			t.Errorf("%q -> %q, want %q", tt.expansion, got, tt.want)
		}
	}
}
