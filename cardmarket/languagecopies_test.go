package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// A product's printing in another language at the same number is priced
// from its own listings; one an upstream id names at another number is not.
func TestLanguageCopies(t *testing.T) {
	b := realDatastore(t)

	product := &cm.Product{IDProduct: 680699, Name: "Carrion Feeder"}
	cardID, cardIDFoil := Fallback(b, product)
	copies := languageCopies(b, product.IDProduct, cardID, cardIDFoil)
	if len(copies) != 1 {
		t.Fatalf("Carrion Feeder has %d language copies, want 1", len(copies))
	}
	co, err := b.GetUUID(copies[0][0])
	if err != nil {
		t.Fatal(err)
	}
	if co.SetCode != "SLD" || co.Number != "1114jpn" {
		t.Errorf("Carrion Feeder's copy is %s %s, want SLD 1114jpn", co.SetCode, co.Number)
	}

	// MTGJSON files this 2024 promo's product on the 2019 Japanese one too
	product = &cm.Product{IDProduct: 382566, Name: "Shock"}
	cardID, cardIDFoil = Fallback(b, product)
	copies = languageCopies(b, product.IDProduct, cardID, cardIDFoil)
	if len(copies) != 0 {
		t.Errorf("Shock has language copies %v, want none", copies)
	}
}

// TestLanguageCopiesUnfiltered pins that a copy in a language the listings
// filter cannot name is left out, rather than priced from English listings.
func TestLanguageCopiesUnfiltered(t *testing.T) {
	card := func(uuid, language string) *mtgmatcher.CardObject {
		return &mtgmatcher.CardObject{Card: mtgmatcher.Card{
			UUID: uuid, Name: "Swords", SetCode: "SLD", Language: language, PlainNumber: "1",
			Identifiers: map[string]string{"mcmId": "7"},
			FoilUUIDs:   map[string]string{mtgmatcher.FinishNonfoil: uuid},
			Finish:      mtgmatcher.FinishNonfoil,
		}}
	}
	b := &mtgmatcher.Backend{
		UUIDs: map[string]*mtgmatcher.CardObject{
			"en": card("en", "English"), "ja": card("ja", "Japanese"), "ph": card("ph", "Phyrexian"),
		},
		Hashes: map[string][]string{mtgmatcher.Normalize("Swords"): {"en", "ja", "ph"}},
	}
	copies := languageCopies(b, 7, "en", "")
	if len(copies) != 1 || copies[0][0] != "ja" {
		t.Errorf("language copies = %v, want the Japanese one alone", copies)
	}
}

// An English Legends or The Dark product resolves to its English printing:
// the Italian copies are Cardmarket's own Legends Italian and The Dark
// Italian products, and carry none of the English ids.
func TestFallbackKeepsEnglishBesideItalianCopies(t *testing.T) {
	b := realDatastore(t)

	for _, product := range []*cm.Product{
		{IDProduct: 7171, Name: "Moat", ExpansionName: "Legends", Number: "28"},
		{IDProduct: 7009, Name: "Vampire Bats", ExpansionName: "Legends", Number: "125"},
		{IDProduct: 7312, Name: "Merfolk Assassin", ExpansionName: "The Dark", Number: "32"},
	} {
		cardID, _ := Fallback(b, product)
		co, err := b.GetUUID(cardID)
		if err != nil {
			t.Errorf("%s (%d) did not resolve: %v", product.Name, product.IDProduct, err)
			continue
		}
		if co.Language != "English" || co.Number != product.Number {
			t.Errorf("%s (%d) resolved to %s %s #%s", product.Name, product.IDProduct, co.Language, co.SetCode, co.Number)
		}
	}
}
