package hareruya

import (
	"os"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

var (
	hareruyaDatastoreOnce sync.Once
	hareruyaDatastore     *mtgmatcher.Backend
	hareruyaDatastoreErr  error
)

func realDatastore(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	hareruyaDatastoreOnce.Do(func() {
		path := os.Getenv("ALLPRINTINGS5_PATH")
		if path == "" {
			return
		}
		hareruyaDatastore, hareruyaDatastoreErr = datastore.Read("magic", path)
	})
	if hareruyaDatastoreErr != nil {
		t.Fatal(hareruyaDatastoreErr)
	}
	if hareruyaDatastore == nil {
		t.Skip("Need ALLPRINTINGS5_PATH set to run this test")
	}
	return hareruyaDatastore
}

// Hareruya's Secret Lair Commander Deck shelf uses the source set's number
// with the edition label "SLD Commander Deck". Those printings are catalogued
// under The List (PLST), not under the Secret Lair set itself.
func TestSecretLairCommanderDeck(t *testing.T) {
	b := realDatastore(t)
	for _, tt := range []struct {
		name, rawNumber, number string
	}{
		{"Bramble Sovereign", "BBD-065", "BBD-65"},
		{"Boon Reflection", "2XM-010", "2XM-10"},
		{"Angelic Chorus", "BBD-087", "BBD-87"},
		{"Ancient Cornucopia", "BIG-016", "BIG-16"},
		{"Resplendent Angel", "LCI-032", "LCI-32"},
		{"Rhys the Redeemed", "2XM-213", "2XM-213"},
		{"Lazotep Quarry", "M3C-131", "M3C-131"},
		{"Mirari's Wake", "MH2-291", "MH2-291"},
		{"Shamanic Revelation", "BLC-237", "BLC-237"},
		{"Suture Priest", "MOC-210", "MOC-210"},
		{"Storm Herd", "KHC-033", "KHC-33"},
		{"Rootborn Defenses", "RVR-026", "RVR-26"},
		{"Invincible Hymn", "ALA-014", "ALA-14"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			product := Product{
				ProductName:   "(" + tt.rawNumber + ")《" + tt.name + "/" + tt.name + "》(SLD構築済み)[PWシンボル付き再版]",
				ProductNameEN: "(" + tt.rawNumber + ")《" + tt.name + "》[SLD Commander Deck]",
				CardName:      tt.name,
			}
			card, err := Preprocess(b, product)
			if err != nil {
				t.Fatalf("Preprocess = %v", err)
			}
			id, err := b.Match(card)
			if err != nil {
				t.Fatalf("Match(%#v) = %v", card, err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != "PLST" || co.Number != tt.number {
				t.Errorf("Match(%#v) = %s|%s, want PLST|%s", card, co.SetCode, co.Number, tt.number)
			}
		})
	}
}
