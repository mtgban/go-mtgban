package abugames

import (
	"errors"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPromoShelf pins the promos this storefront files under one edition and
// one number apiece, where only the wording says which programme handed the
// card out.
func TestPromoShelf(t *testing.T) {
	for _, test := range []struct {
		desc    string
		card    ABUCard
		wantSet string
	}{
		{"a bundle promo, numbered in the set it comes with", ABUCard{
			DisplayTitle: "Frodo, Sauron's Bane (Bundle Promo) - FOIL",
			Edition:      "The Lord of the Rings: Tales of Middle-earth", Number: "448"}, "LTR"},
		{"the showcase it was answering with", ABUCard{
			DisplayTitle: "Frodo, Sauron's Bane (Showcase) - FOIL",
			Edition:      "The Lord of the Rings: Tales of Middle-earth", Number: "304"}, "LTR"},
		{"a nationals promo shelved with the judge cards", ABUCard{
			DisplayTitle: "Flooded Strand (Nationals) - FOIL",
			Edition:      "Judge Gift Cards 2009", Number: "7"}, "PNAT"},
		{"a regional championship qualifier", ABUCard{
			DisplayTitle: "Snapcaster Mage (RCQ) - FOIL", Edition: "Promo", Number: "2"}, "PR23"},
		{"and one from another year of the same programme", ABUCard{
			DisplayTitle: "Selfless Spirit (RCQ) - FOIL", Edition: "Promo", Number: "2"}, "PRCQ"},
		{"a standard showdown promo", ABUCard{
			DisplayTitle: "Squall, SeeD Mercenary (Standard Showdown) - FOIL",
			Edition:      "Promo", Number: "2"}, "PSS5"},
		{"a Love Your LGS promo", ABUCard{
			DisplayTitle: "Chromatic Lantern (Love Your LGS) - FOIL", Edition: "Promo", Number: "1"}, "PLG25"},
		{"a textless player reward shelved with the MagicFest ones", ABUCard{
			DisplayTitle: "Lightning Bolt (Player Rewards Textless) - FOIL",
			Edition:      "MagicFest 2019", Number: "1"}, "P10"},
		{"the MagicFest one beside it", ABUCard{
			DisplayTitle: "Lightning Bolt (MagicFest Textless) - FOIL",
			Edition:      "MagicFest 2019", Number: "1"}, "PF19"},
	} {
		t.Run(test.desc, func(t *testing.T) {
			card := test.card
			in, err := preprocess(&card)
			if err != nil {
				t.Fatalf("preprocess(%q) = %v", card.DisplayTitle, err)
			}
			id, err := mtgmatcher.Match(in)
			if err != nil {
				t.Fatalf("Match(%q) = %v", in, err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != test.wantSet {
				t.Errorf("Match(%q) = %s|%s, want a %s printing", in, co.SetCode, co.Number, test.wantSet)
			}
		})
	}
}

// TestEtchedForFoil pins the Secret Lair cards sold etched and never in plain
// foil, which this storefront calls FOIL like any other.
func TestEtchedForFoil(t *testing.T) {
	for _, test := range []struct{ desc, title, number string }{
		{"a basic land sold etched", "Mountain (Secret Lair 49) - FOIL", "49"},
		{"and a spell", "Temur Sabertooth (Secret Lair) - FOIL", "308"},
	} {
		t.Run(test.desc, func(t *testing.T) {
			card := ABUCard{DisplayTitle: test.title, Edition: "Secret Lair Drop", Number: test.number}
			in, err := preprocess(&card)
			if err != nil {
				t.Fatalf("preprocess(%q) = %v", test.title, err)
			}
			id, err := mtgmatcher.Match(in)
			if err != nil {
				t.Fatalf("Match(%q) = %v", in, err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if !co.Etched {
				t.Errorf("Match(%q) = %s|%s, foil=%v etched=%v, want the etched printing",
					in, co.SetCode, co.Number, co.Foil, co.Etched)
			}
		})
	}
}

// TestLanguageFromTitle pins the rows whose language field says English and
// whose own title says otherwise. There is no Japanese 30th Anniversary Serra
// Angel, so that listing answered with the English one and priced beside it.
func TestLanguageFromTitle(t *testing.T) {
	card := ABUCard{DisplayTitle: "Serra Angel (30th Anniversary History Retro Frame JP) - FOIL",
		Edition: "Promo", Number: "1★", Title: "Non-English - Japanese", Language: []string{"English"}}
	in, err := preprocess(&card)
	if err == nil {
		_, err = mtgmatcher.Match(in)
	}
	if err == nil {
		t.Errorf("preprocess(%q) = %v, want a refusal", card.DisplayTitle, in)
		return
	}
	if !errors.Is(err, errForeignListing) {
		t.Logf("refused as %v", err)
	}
}
