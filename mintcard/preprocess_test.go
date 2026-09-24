package mintcard

import (
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/tcgplayer"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

var (
	datastoreOnce    sync.Once
	datastoreErr     error
	datastoreBackend *mtgmatcher.Backend
)

// realDatastore loads the Magic datastore the first time a test asks for it,
// and skips where the run carries none.
func realDatastore(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	datastoreOnce.Do(func() {
		path := os.Getenv("ALLPRINTINGS5_PATH")
		if path == "" {
			return
		}
		b, err := datastore.Read("magic", path)
		if err != nil {
			datastoreErr = err
			return
		}
		datastoreBackend = b
	})
	if datastoreErr != nil {
		t.Fatal(datastoreErr)
	}
	if datastoreBackend == nil {
		t.Skip("Need ALLPRINTINGS5_PATH set to run this test")
	}
	return datastoreBackend
}

// TestPreprocessInserts pins that the inserts a booster carries beside its
// cards - theme cards, the helper cards of a set the datastore files none
// for - and the emblems and signed cards are refused quietly: 364 of the
// 411 lines of the night of 2026-09-06.
func TestPreprocessInserts(t *testing.T) {
	// No datastore lookup is reached on this path, so an empty backend is
	// enough.
	b := &mtgmatcher.Backend{}
	for _, name := range []string{
		"Angels Theme Card",
		"Helper Card",
		"Emblem Garruk, Caller of Beasts",
		"Chrome Mox (Manuel Bevand Signature)",
	} {
		_, err := preprocess(b, name, "", "Regular", "English", "Foundations Jumpstart", "J25")
		if !errors.Is(err, mtgmatcher.ErrUnsupported) {
			t.Errorf("preprocess(%q) = %v, want ErrUnsupported", name, err)
		}
	}
}

// TestPreprocessShelves pins the storefront's own spellings against the
// datastore's: a duel deck code of its own, a Final Fantasy buy-a-box
// promo listed under its flavor name with the card's own in parentheses,
// the helper card of a set whose substitute cards the datastore files, the
// two cards whose names read like inserts, a Phyrexian printing filed as
// English beside a List card that only names New Phyrexia, and a catch-all
// promo shelf numbering its cards by TCGplayer's position, not the card's.
func TestPreprocessShelves(t *testing.T) {
	b := realDatastore(t)
	for _, tt := range []struct {
		name, edition, code, language string
		wantSet, wantNumber           string
	}{
		{"Coalition Relic", "Duel Decks: Phyrexia vs. Coalition", "PVC", "English", "DDE", "54"},
		{"Fatalism (Arcane Denial) (Final Fantasy Buy-a-Box)", "Promo: Buy-A-Box", "PBAB", "Japanese", "RFIN", "J2"},
		{"Helper Card (9/9)", "Kaldheim", "KHM", "English", "SKHM", "9"},
		{"Signature Slam", "Modern Horizons 3", "MH3", "English", "MH3", "168"},
		{"Emblem of the Warmind", "Future Sight", "FUT", "English", "FUT", "112"},
		{"Plains (267) (Phyrexian)", "Phyrexia: All Will Be One", "ONE", "English", "ONE", "267"},
		{"Beast Within (New Phyrexian)", "Mystery Booster/The List", "MYS", "English", "PLST", "NPH-103"},
		{"Lightning Bolt (0002)", "Promo:\u00a0Unique and Miscellaneous", "PMSC", "English", "PW26", "5"},
	} {
		theCard, err := preprocess(b, tt.name, "", "Regular", tt.language, tt.edition, tt.code)
		if err != nil {
			t.Fatalf("preprocess(%q) = %v", tt.name, err)
		}
		cardID, err := b.Match(theCard)
		if err != nil {
			t.Fatalf("Match(%q) = %v", theCard, err)
		}
		co, _ := b.GetUUID(cardID)
		if co.SetCode != tt.wantSet || co.Number != tt.wantNumber {
			t.Errorf("Match(%q) = %s %s, want %s %s", theCard, co.SetCode, co.Number, tt.wantSet, tt.wantNumber)
		}
	}
}

// TestPreprocessLanguage pins that the feed's blanket Language "English" is
// dropped before it reaches mtgmatcher: an Arabic prerelease foil shelved
// under that tag, with no language of its own beside "English" to pass,
// would otherwise demand an English candidate and refuse.
func TestPreprocessLanguage(t *testing.T) {
	b := realDatastore(t)
	theCard, err := preprocess(b, "Stone-Tongue Basilisk (Odyssey Prerelease)(Arabic)", "", "Foil Prerelease", "English", "Odyssey", "ODY")
	if err != nil {
		t.Fatalf("preprocess() = %v", err)
	}
	cardID, err := b.Match(theCard)
	if err != nil {
		t.Fatalf("Match(%q) = %v", theCard, err)
	}
	co, _ := b.GetUUID(cardID)
	if co.SetCode != "PODY" || co.Number != "276" || !co.Foil {
		t.Errorf("Match(%q) = %s %s foil=%v, want PODY 276 foil", theCard, co.SetCode, co.Number, co.Foil)
	}
}

// TestPreprocessSignatureSpellbook pins that a Signature Spellbook card, a
// real set, escapes the insert guard the word "Signature" otherwise trips.
func TestPreprocessSignatureSpellbook(t *testing.T) {
	b := realDatastore(t)
	theCard, err := preprocess(b, "Brainstorm (Signature Spellbook: Jace)", "", "Regular", "English", "Mystery Booster/The List", "MYS")
	if err != nil {
		t.Fatalf("preprocess() = %v", err)
	}
	cardID, err := b.Match(theCard)
	if err != nil {
		t.Fatalf("Match(%q) = %v", theCard, err)
	}
	co, _ := b.GetUUID(cardID)
	if co.SetCode != "PLST" || co.Number != "SS1-3" {
		t.Errorf("Match(%q) = %s %s, want PLST SS1-3", theCard, co.SetCode, co.Number)
	}
}

// TestPreprocessTokenFlavorSwap pins that a token's own parenthetical names
// the set it comes from, not a flavor: "Goblin Soldier Token (Apocalypse)"
// keeps its own name rather than swapping to Apocalypse, also a card.
func TestPreprocessTokenFlavorSwap(t *testing.T) {
	b := realDatastore(t)
	theCard, err := preprocess(b, "Goblin Soldier Token (Apocalypse) (Player Rewards)", "", "Regular", "English", "Promo: Magic Player Rewards", "PMPR")
	if err != nil {
		t.Fatalf("preprocess() = %v", err)
	}
	if theCard.Name != "Goblin Soldier Token" {
		t.Errorf("preprocess() name = %q, want %q", theCard.Name, "Goblin Soldier Token")
	}
	cardID, err := b.Match(theCard)
	if err != nil {
		t.Fatalf("Match(%q) = %v", theCard, err)
	}
	co, _ := b.GetUUID(cardID)
	if co.SetCode != "MPR" || co.Number != "6" {
		t.Errorf("Match(%q) = %s %s, want MPR 6", theCard, co.SetCode, co.Number)
	}
}

// TestBuildSkuToUUID pins buildSku2UUID's tie-break on a hand-built
// fixture: two different uuids rivaling one sku id is dropped unless
// exactly one resolves to the finish the sku names, and one uuid's own
// sku list repeating that id - not a rival claim - is kept regardless.
func TestBuildSkuToUUID(t *testing.T) {
	const uuidA, uuidB, uuidC, uuidD, uuidE, uuidF, uuidG, uuidH = "uuid-a", "uuid-b", "uuid-c", "uuid-d", "uuid-e", "uuid-f", "uuid-g", "uuid-h"
	newCard := func(uuid, name, number string, foil bool, finishes ...string) *mtgmatcher.CardObject {
		return &mtgmatcher.CardObject{
			Card: mtgmatcher.Card{Name: name, SetCode: "TST", Number: number, UUID: uuid, Finishes: finishes},
			Foil: foil,
		}
	}
	b := &mtgmatcher.Backend{UUIDs: map[string]*mtgmatcher.CardObject{
		uuidA: newCard(uuidA, "Card A", "1", false, "nonfoil"),
		uuidB: newCard(uuidB, "Card B", "2", false, "nonfoil"),
		uuidC: newCard(uuidC, "Card C", "3", false, "nonfoil"),
		uuidD: newCard(uuidD, "Card D", "4", false, "nonfoil"),
		uuidE: newCard(uuidE, "Card E", "5", true, "foil"),
		uuidG: newCard(uuidG, "Card G", "6", false, "nonfoil"),
		uuidH: newCard(uuidH, "Card H", "8", true, "foil"),
	}}
	etchedF := newCard(uuidF, "Card F", "7", false, "etched")
	etchedF.Etched = true
	b.UUIDs[uuidF] = etchedF

	skus := tcgplayer.SKUMap{}
	// Id 100 is claimed by two different, equally nonfoil cards: still
	// ambiguous once the finish check runs, so it is dropped.
	skus[uuidA] = append(skus[uuidA], tcgplayer.TCGSku{SkuID: 100, Language: "ENGLISH", Printing: "NON FOIL"})
	skus[uuidB] = append(skus[uuidB], tcgplayer.TCGSku{SkuID: 100, Language: "ENGLISH", Printing: "NON FOIL"})
	// Id 200 is repeated within card C's own sku list - a catalog
	// duplicate, not a rival claim - and is kept.
	skus[uuidC] = append(skus[uuidC], tcgplayer.TCGSku{SkuID: 200, Language: "ENGLISH", Printing: "NON FOIL"})
	skus[uuidC] = append(skus[uuidC], tcgplayer.TCGSku{SkuID: 200, Language: "ENGLISH", Printing: "NON FOIL"})
	// Id 300 names only card D: unambiguous outright.
	skus[uuidD] = append(skus[uuidD], tcgplayer.TCGSku{SkuID: 300, Language: "ENGLISH", Printing: "NON FOIL"})
	// Id 400 is claimed by both D (nonfoil only, so a foil request falls
	// back to its own nonfoil id - a mismatch) and E (sold foil outright,
	// a match): the mismatch breaks the tie in E's favor.
	skus[uuidD] = append(skus[uuidD], tcgplayer.TCGSku{SkuID: 400, Language: "ENGLISH", Printing: "FOIL"})
	skus[uuidE] = append(skus[uuidE], tcgplayer.TCGSku{SkuID: 400, Language: "ENGLISH", Printing: "FOIL"})
	// Id 600 is repeated within card G's own sku list, both mistagged
	// FOIL for a card G only sells nonfoil: still one uuid, so the
	// mismatch must not drop it the way it would if two rival uuids gave
	// the same wrong finish.
	skus[uuidG] = append(skus[uuidG], tcgplayer.TCGSku{SkuID: 600, Language: "ENGLISH", Printing: "FOIL"})
	skus[uuidG] = append(skus[uuidG], tcgplayer.TCGSku{SkuID: 600, Language: "ENGLISH", Printing: "FOIL"})

	// Id 500 is an etched sku claimed by F (sold etched) and H (sold only
	// foil, so the etched request falls back to its foil id): an etched
	// sku is never foil, so F alone matches.
	skus[uuidF] = append(skus[uuidF], tcgplayer.TCGSku{SkuID: 500, Language: "ENGLISH", Printing: "FOIL", Finish: "ETCHED"})
	skus[uuidH] = append(skus[uuidH], tcgplayer.TCGSku{SkuID: 500, Language: "ENGLISH", Printing: "FOIL", Finish: "ETCHED"})

	mint := &MTGMintCard{backend: b, skusData: skus}
	got := mint.buildSku2UUID()

	want := map[int]string{200: uuidC, 300: uuidD, 400: uuidE, 500: uuidF, 600: uuidG}
	for sku, id := range want {
		if got[sku] != id {
			t.Errorf("buildSku2UUID()[%d] = %q, want %q", sku, got[sku], id)
		}
	}
	if id, found := got[100]; found {
		t.Errorf("buildSku2UUID()[100] = %q, want unmapped (still ambiguous)", id)
	}
	if len(got) != len(want) {
		t.Errorf("buildSku2UUID() = %v, want exactly %v", got, want)
	}
}
