package coolstuffinc

import (
	"testing"
)

// TestPreprocessBuylistResolvesTokenPairing pins a two-sided token buylist
// row resolving to the combined entity matcher/magic derives for it,
// anchored by the row's own Code and Number - CSI's buylist JSON carries
// both, and PreprocessBuylist routes a "X (Token) // Y (Token)" name through
// them rather than matching the whole two-face string as one card name.
// "Angel // Cat" is Commander 2014's own derived pairing (no native mtgjson
// entity, no usable TCGplayer id).
func TestPreprocessBuylistResolvesTokenPairing(t *testing.T) {
	b := readGameDatastore(t, "magic", "ALLPRINTINGS5_PATH")

	card := CSIPriceEntry{
		PID: "209796", Name: "Angel // Cat (Token)", ItemSet: "Commander 2014 Edition",
		Price: "0.05", Number: "001/002", RarityName: "Fixed", Code: "C14",
	}
	out, err := PreprocessBuylist(b, card)
	if err != nil {
		t.Fatalf("PreprocessBuylist(%s) = %v", card.PID, err)
	}
	co, err := b.GetUUID(out.ID)
	if err != nil {
		t.Fatalf("GetUUID(%s) = %v", out.ID, err)
	}
	if co.Identifiers["derivedTokenPair"] != "true" {
		t.Errorf("resolved to %q, want a derived token pairing", co.Name)
	}
	if co.Name != "Angel // Cat" {
		t.Errorf("resolved to %q, want \"Angel // Cat\"", co.Name)
	}
}

// TestPreprocessBuylistResolvesCompoundNumberPairing pins the same anchor
// working off a compound "020/023" Number - one collector number per face,
// CSI's own convention for most two-sided token rows - trying the first
// half against the pairing's first face.
func TestPreprocessBuylistResolvesCompoundNumberPairing(t *testing.T) {
	b := readGameDatastore(t, "magic", "ALLPRINTINGS5_PATH")

	card := CSIPriceEntry{
		PID: "209802", Name: "Beast (Token) // Elf Druid (Token)", ItemSet: "Commander 2014 Edition",
		Price: "0.50", Number: "020/023", RarityName: "Fixed", Code: "C14",
	}
	out, err := PreprocessBuylist(b, card)
	if err != nil {
		t.Fatalf("PreprocessBuylist(%s) = %v", card.PID, err)
	}
	co, err := b.GetUUID(out.ID)
	if err != nil {
		t.Fatalf("GetUUID(%s) = %v", out.ID, err)
	}
	if co.Identifiers["derivedTokenPair"] != "true" {
		t.Errorf("resolved to %q, want a derived token pairing", co.Name)
	}
	if co.Name != "Beast // Elf Druid" {
		t.Errorf("resolved to %q, want \"Beast // Elf Druid\"", co.Name)
	}
}

// TestPreprocessBuylistRefusesUnresolvedTokenPairing pins the reverse:
// "Horror // Zombie" is a real Archenemy: Nicol Bolas buylist row, but
// mtgjson carries no Zombie token under E01 (or its token sheet) at all -
// no tokenProducts record links the two faces, so there is nothing to
// anchor. A refusal here is correct per this project's own standing rule
// (drop what mtgjson has no record of, without logging it) rather than a
// gap to chase further.
func TestPreprocessBuylistRefusesUnresolvedTokenPairing(t *testing.T) {
	b := readGameDatastore(t, "magic", "ALLPRINTINGS5_PATH")

	card := CSIPriceEntry{
		PID: "243749", Name: "Horror // Zombie (Token)", ItemSet: "Archenemy: Nicol Bolas",
		Price: "0.02", Number: "003", RarityName: "Fixed", Code: "E01",
	}
	_, err := PreprocessBuylist(b, card)
	if err == nil {
		t.Fatal("PreprocessBuylist succeeded, want a refusal: no mtgjson record links Horror to Zombie in this edition")
	}
}

// TestPreprocessBuylistIgnoresSingleFaceTokenDash pins that a single-faced
// token whose own name happens to carry " - " (CSI's own way of telling
// two same-name token variants apart, e.g. Core Set 2021's two different
// Cat tokens) is never routed through the two-sided pairing path: the
// trigger requires " // ", not just a hyphen, and preprocessTokenPairBuylist
// would refuse this name outright since it never splits on " - " without a
// leading "(Token)" name on both sides.
func TestPreprocessBuylistIgnoresSingleFaceTokenDash(t *testing.T) {
	b := readGameDatastore(t, "magic", "ALLPRINTINGS5_PATH")

	card := CSIPriceEntry{
		PID: "299568", Name: "Cat (Token) - #20", ItemSet: "Core Set 2021",
		Price: "0.05", Number: "020", RarityName: "Fixed", Code: "M21",
	}
	_, err := PreprocessBuylist(b, card)
	if err != nil {
		t.Fatalf("PreprocessBuylist(%s) = %v, want the ordinary single-token path to still resolve it", card.PID, err)
	}
}

// TestRetailMagicResolvesTokenPairing pins a two-sided token sale listing,
// which names no set code or number, landing on the pair its buylist row
// lands on rather than on its first face.
func TestRetailMagicResolvesTokenPairing(t *testing.T) {
	b := readGameDatastore(t, "magic", "ALLPRINTINGS5_PATH")
	csi, err := NewScraper(b)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		listing sellListing
		buylist CSIPriceEntry
		want    string
	}{
		{
			listing: sellListing{pid: "262201", name: "Dragon Egg (Token) // Dragon (Token)", edition: "Commander 2018 Edition"},
			buylist: CSIPriceEntry{PID: "262201", Name: "Dragon Egg (Token) // Dragon (Token)", ItemSet: "Commander 2018 Edition", Number: "010", Code: "C18"},
			want:    "Dragon Egg Token // Dragon",
		},
		{
			listing: sellListing{pid: "209810", name: "Myr (Token) // Pentavite (Token)", edition: "Commander 2014 Edition"},
			buylist: CSIPriceEntry{PID: "209810", Name: "Myr (Token) // Pentavite (Token)", ItemSet: "Commander 2014 Edition", Number: "028/029", Code: "C14"},
			want:    "Myr // Pentavite",
		},
		{
			listing: sellListing{pid: "339971", name: "Warrior (Token) // Inkling (Token)", edition: "Commander Legends: Battle for Baldur's Gate"},
			buylist: CSIPriceEntry{PID: "339971", Name: "Warrior (Token) // Inkling (Token)", ItemSet: "Commander Legends: Battle for Baldur's Gate", Number: "032/045", Code: "CLB"},
			want:    "Warrior // Inkling",
		},
		// The Cat face is Amonkhet's, filed on another set than the shelf.
		{
			listing: sellListing{pid: "357757", name: "Earthshaker Khenra (Token) // Cat (Token)", edition: "Hour of Devastation"},
			buylist: CSIPriceEntry{PID: "357757", Name: "Earthshaker Khenra (Token) // Cat (Token)", ItemSet: "Hour of Devastation", Number: "004/016", Code: "HOU"},
			want:    "Earthshaker Khenra Token // Cat",
		},
		{
			listing: sellListing{pid: "319246", name: "Dungeon of the Mad Mage (Token) // Goblin (Token)", edition: "D&D: Adventures in the Forgotten Realms", foil: true},
			want:    "Goblin // Dungeon of the Mad Mage",
		},
		// mtgjson's own combined printing, not a derived pair.
		{
			listing: sellListing{pid: "324205", name: "Day // Night (Token)", edition: "Innistrad: Midnight Hunt "},
			want:    "Day // Night",
		},
	}
	for _, test := range tests {
		t.Run(test.listing.pid, func(t *testing.T) {
			_, cardID, err := retailMagic(csi, test.listing)
			if err != nil {
				t.Fatalf("retailMagic(%s) = %v", test.listing.pid, err)
			}
			co, err := b.GetUUID(cardID)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", cardID, err)
			}
			if co.Name != test.want || co.Foil != test.listing.foil {
				t.Errorf("landed on %s %q #%s foil=%v, want %q foil=%v", co.SetCode, co.Name, co.Number, co.Foil, test.want, test.listing.foil)
			}
			if test.buylist.PID == "" {
				return
			}
			_, buyID, err := buylistMagic(csi, test.buylist)
			if err != nil {
				t.Fatalf("buylistMagic(%s) = %v", test.buylist.PID, err)
			}
			if buyID != cardID {
				t.Errorf("retail landed on %s, buylist on %s", cardID, buyID)
			}
		})
	}
}

// TestRetailMagicRefusesUnanchoredTokenPairing pins the listings no face
// anchors to one pair. Commander 2017 files two Dragons that each pair with
// Gold, and the note's "6/6" would read as Dragon 6 where it means 7;
// Commander 2015's Shapeshifter pairs with no second Shapeshifter.
func TestRetailMagicRefusesUnanchoredTokenPairing(t *testing.T) {
	b := readGameDatastore(t, "magic", "ALLPRINTINGS5_PATH")
	csi, err := NewScraper(b)
	if err != nil {
		t.Fatal(err)
	}

	for _, listing := range []sellListing{
		{pid: "246484", name: "Dragon (Token) // Gold (Token)", edition: "Commander 2017 Edition", notes: "6/6 Dragon; #007"},
		{pid: "221792", name: "Shapeshifter (Token) // Shapeshifter (Token)", edition: "Commander 2015 Edition"},
	} {
		_, cardID, err := retailMagic(csi, listing)
		if err == nil {
			co, _ := b.GetUUID(cardID)
			t.Errorf("retailMagic(%s) landed on %s, want a refusal", listing.pid, co)
		}
	}
}
