package strikezone

import (
	"testing"
)

// TestUnnamedTextured pins which listings the textured guard refuses. The
// store shelves the Duskmourn mythics three times over - plain, "(Showcase)"
// and "(Showcase) (Textured)" - but the set files its showcase frame on the
// textured printing alone, so both showcase wordings reached it and the
// cheaper card was bought and sold at the textured one's price.
func TestUnnamedTextured(t *testing.T) {
	b := realDatastore(t)

	for _, tt := range []struct {
		desc, name, edition, notes string
		wantNumber                 string
	}{
		{
			desc: "the wording that names it keeps the textured printing",
			name: "Valgavoth, Terror Eater (Showcase) (Textured)", edition: "Duskmourn: House of Horror",
			notes: "Near Mint Foil English", wantNumber: "407",
		},
		{
			desc: "the one standing beside it names another card",
			name: "Valgavoth, Terror Eater (Showcase)", edition: "Duskmourn: House of Horror",
			notes: "Near Mint Foil English", wantNumber: "",
		},
		{
			desc: "and says so in either finish",
			name: "Valgavoth, Terror Eater (Showcase)", edition: "Duskmourn: House of Horror",
			notes: "Near Mint Normal English", wantNumber: "",
		},
		{
			desc: "Tyvar sells the same three",
			name: "Tyvar, the Pummeler (Showcase)", edition: "Duskmourn: House of Horror",
			notes: "Near Mint Foil English", wantNumber: "",
		},
		{
			desc: "so does Kaito",
			name: "Kaito, Bane of Nightmares (Showcase)", edition: "Duskmourn: House of Horror",
			notes: "Near Mint Foil English", wantNumber: "",
		},
		{
			desc: "and the Wandering Rescuer",
			name: "The Wandering Rescuer (Showcase)", edition: "Duskmourn: House of Horror",
			notes: "Near Mint Foil English", wantNumber: "",
		},
		{
			desc: "silence over a printing wearing nothing is not a claim",
			name: "Valgavoth, Terror Eater", edition: "Duskmourn: House of Horror",
			notes: "Near Mint Foil English", wantNumber: "120",
		},
		{
			desc: "nor is it one where the wording reaches the untextured sibling",
			name: "Solitude (Borderless)", edition: "Special Guests",
			notes: "Near Mint Foil English", wantNumber: "44",
		},
		{
			desc: "a set spelling the word out is left where it landed",
			name: "Solitude (Borderless) (Textured Foil)", edition: "Special Guests",
			notes: "Near Mint Foil English", wantNumber: "49",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			card, err := preprocess(b, tt.name, tt.edition, tt.notes)
			if err != nil {
				t.Fatalf("preprocess(%q) = %v", tt.name, err)
			}
			cardID, err := b.Match(card)
			if err != nil {
				t.Fatalf("Match(%q) = %v", tt.name, err)
			}
			co, err := b.GetUUID(cardID)
			if err != nil {
				t.Fatal(err)
			}

			refused := namesAbsentTreatment(card.Variation, co) ||
				wearsUnnamedTextured(b, card.Variation, co)
			if refused {
				if tt.wantNumber != "" {
					t.Fatalf("%q was refused, want %s #%s", tt.name, co.SetCode, tt.wantNumber)
				}
				return
			}
			if tt.wantNumber == "" {
				t.Fatalf("%q kept %s #%s %v, want refused",
					tt.name, co.SetCode, co.Number, co.PromoTypes)
			}
			if co.Number != tt.wantNumber {
				t.Errorf("%q got #%s, want #%s", tt.name, co.Number, tt.wantNumber)
			}
		})
	}
}
