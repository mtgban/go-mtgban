package toamagic

import (
	"os"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/matcher"

	_ "github.com/mtgban/go-mtgban/matcher/magic"
)

var (
	datastoreOnce    sync.Once
	datastoreErr     error
	datastoreBackend *matcher.Backend
)

// realDatastore reads the Magic datastore the first time a test asks for it,
// and skips where the run carries none.
func realDatastore(t *testing.T) *matcher.Backend {
	t.Helper()
	datastoreOnce.Do(func() {
		path := os.Getenv("ALLPRINTINGS5_PATH")
		if path == "" {
			return
		}
		backend, err := datastore.Read(matcher.GameMagic, path)
		if err != nil {
			datastoreErr = err
			return
		}
		datastoreBackend = backend
	})
	if datastoreErr != nil {
		t.Fatal(datastoreErr)
	}
	if datastoreBackend == nil {
		t.Skip("Need ALLPRINTINGS5_PATH set to run this test")
	}
	return datastoreBackend
}

// TestListingResolvesToItsPrinting pins listings whose edition, finish or
// emblem wording is the store's own, each to the printing it sells.
func TestListingResolvesToItsPrinting(t *testing.T) {
	b := realDatastore(t)

	tests := []struct {
		cardName string
		edition  string
		variant  string
		set      string
		number   string
		foil     bool
	}{
		{"Scythecat Cub", "Jumpstart Foundations", "", "J25", "24", false},
		{"Mana Drain", "OTJ: Breaking News", "Foil", "OTP", "11", true},
		{"Chrome Mox", "MTG: Special Guests", "Borderless", "SPG", "92", false},
		{"City of Brass", "Teenage Mutant Ninja Turtles: Eternal-Legal", "", "TMC", "62", false},
		{"Niko, Light of Hope", "Duskmourn: House of Horror", "Showcase", "DSK", "355", false},
		{"Niko, Light of Hope", "Duskmourn: House of Horror", "Foil Showcase", "DSK", "355", true},
		{"Niko, Light of Hope", "Duskmourn: House of Horror", "Textured Foil Showcase", "DSK", "410", true},
		{"Kona, Rescue Beastie", "Duskmourn: House of Horror", "Showcase", "DSK", "358", false},
		{"Come Back Wrong", "Duskmourn: House of Horror", "Showcase", "DSK", "309", false},
		{"The Aetherspark", "Aetherdrift", "First-Place Foil", "DFT", "496", true},
		{"Valkyrie's Call", "Foundations", "Mana Foil Borderless", "FDN", "371", true},
		{"Swiftfoot Boots", "The Brothers' War", "Retro Schematic", "BRR", "121", false},
		{"Demonic Tutor (DVD)", "Mystery Booster Cards", "", "PLST", "DDC-49", false},
		{"Krosan Tusker (ONS)", "Mystery Booster Cards", "", "PLST", "ONS-272", false},
		{"Wrenn and Six", "Secret Lair Drop Series", "", "SLP", "15", false},
		{"Sleight of Hand", "Secret Lair Drop Series", "Foil Pro Tour Promo", "SLP", "25", true},
		{"Emberheart Challenger", "Secret Lair Drop Series", "Regional Championships Promo", "SLP", "48", false},
		{"Necropotence", "Secret Lair Drop Series", "Foil", "SLC", "1995", true},
		{"Lim-Dul's Vault", "Secret Lair Drop Series", "Borderless", "SLC", "1996", false},
		{"Gray Merchant of Asphodel", "Secret Lair Drop Series", "", "SLC", "7", false},
		{"Murktide Regent", "Secret Lair Drop Series", "", "SLD", "1878", false},
		{"Emblem", "Kaldheim", "Kaya the Inexorable", "TKHM", "20", false},
		{"Emblem", "Shadows Over Innistrad", "Jace", "TSOI", "17", false},
		{"Emblem", "Battle for Zendikar", "Nixilis", "TBFZ", "13", false},
		{"Wolf (0018) // Emblem", "Innistrad Remastered", "Arlinn, Embraces by the Moon Double-Sided Token", "TINR", "18//23", false},
		{"Eldrazi (0001) // Emblem", "Commander Masters", "Ob Nixilis of the Black Oath (0052) Double-Sided Token", "TCMM", "1//52", false},
		{"Emblem", "Foundations", "Kaito, Cunning Infiltrator // Ninja Doubled-Sided Token", "TFDN", "12//24", false},
		{"Elk // Emblem", "Lorwyn Eclipsed", "Oko, Shadowmoor Scion Double-Sided Token", "TECL", "2//12", false},
		{"Emblem", "Final Fantasy", "Sephiroth, One-Winged Angel // Robot Warrior Double-Sided Token Foil", "TFIN", "12//24", true},
		{"Emblem", "Commander 2014", "Teferi // Zombie Token Blue", "TC14", "11//34", false},
	}
	for _, test := range tests {
		theCard, err := preprocess(b, test.cardName, test.edition, test.variant)
		if err != nil {
			t.Fatalf("%s %s: preprocess: %v", test.cardName, test.variant, err)
		}

		cardID, err := b.Match(theCard)
		if err != nil {
			t.Fatalf("%s %s: Match: %v", test.cardName, test.variant, err)
		}
		co, err := b.GetUUID(cardID)
		if err != nil {
			t.Fatalf("%s %s: GetUUID: %v", test.cardName, test.variant, err)
		}
		if co.SetCode != test.set || co.Number != test.number || co.Foil != test.foil {
			t.Errorf("%s %s: matched %s %s foil=%t, want %s %s foil=%t", test.cardName, test.variant, co.SetCode, co.Number, co.Foil, test.set, test.number, test.foil)
		}
	}
}
