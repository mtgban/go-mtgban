package strikezone

import "testing"

// TestFlavorHead pins which "Head - Card" listings keep their head. A flavor
// name the catalog sells the card under reaches Match whole, even where the
// search does not index it, as does a Godzilla name on the Ikoria shelf; a
// head nothing knows leaves the card's own.
func TestFlavorHead(t *testing.T) {
	b := realDatastore(t)

	for _, tt := range []struct {
		desc, name, edition, notes string
		wantSet, wantNumber        string
	}{
		{
			desc: "a reversible card's flavor name",
			name: "Optimus Prime - Darksteel Colossus", edition: "Secret Lair", notes: "Near Mint Normal English",
			wantSet: "SLD", wantNumber: "1081",
		},
		{
			desc: "and one printed on a single face",
			name: "Mimir's Ancient Wisdom - Teferi's Ageless Insight", edition: "Secret Lair", notes: "Near Mint Foil English",
			wantSet: "SLD", wantNumber: "2214",
		},
		{
			desc: "Ikoria's Godzilla names reach the Ikoria branch",
			name: "Battra Terror of the City - Dirge Bat", edition: "Ikoria: Lair of Behemoths", notes: "Normal",
			wantSet: "IKO", wantNumber: "386",
		},
		{
			desc: "a head nothing knows leaves the card's own",
			name: "Astral Tiran - Primeval Titan (Showcase)", edition: "Universes Beyond: FINAL FANTASY: Through the Ages", notes: "Foil",
			wantSet: "FCA", wantNumber: "48",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			card, err := preprocess(b, tt.name, tt.edition, tt.notes)
			if err != nil {
				t.Fatalf("preprocess(%q) = %v", tt.name, err)
			}
			id, err := b.Match(card)
			if err != nil {
				t.Fatalf("Match(%q) = %v", tt.name, err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNumber {
				t.Errorf("%q got %s #%s, want %s #%s", tt.name, co.SetCode, co.Number, tt.wantSet, tt.wantNumber)
			}
		})
	}
}
