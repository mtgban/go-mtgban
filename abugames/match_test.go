package abugames

import (
	"errors"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func TestMatchCardIdentifiers(t *testing.T) {
	realDatastore(t)
	for _, foil := range []bool{false, true} {
		for _, tt := range []struct {
			name      string
			scryfall  []string
			tcgplayer []int64
			wantAlias bool
		}{
			{"Scryfall", []string{"0ecea0ba-da29-45f1-b72b-50b873309483"}, nil, false},
			{"TCGplayer", nil, []int64{496156}, false},
			{"both", []string{"0ecea0ba-da29-45f1-b72b-50b873309483"}, []int64{496156}, false},
			{"unknown Scryfall with valid TCGplayer", []string{"not-an-id"}, []int64{496156}, false},
			{"unknown TCGplayer with valid Scryfall", []string{"0ecea0ba-da29-45f1-b72b-50b873309483"}, []int64{-1}, false},
			{"missing", nil, nil, true},
			{"unknown", []string{"not-an-id"}, []int64{-1}, true},
			{"different card", []string{"f2a7042f-a6f0-4e77-86a2-5eb0d2587363"}, []int64{589737}, true},
			{"wrong namespace", []string{"496156"}, nil, true},
		} {
			t.Run(tt.name+map[bool]string{true: " foil", false: " nonfoil"}[foil], func(t *testing.T) {
				card := ABUCard{DisplayTitle: "Lobelia Sackville-Baggins (Borderless)", Edition: "The Lord of the Rings: Tales of Middle-earth", Number: "399", Language: []string{"English"}, ScryfallIDs: tt.scryfall, TCGplayerIDs: tt.tcgplayer}
				if foil {
					card.DisplayTitle += " - FOIL"
				}
				in, err := preprocess(&card)
				if err != nil {
					t.Fatal(err)
				}
				id, err := matchCard(&card, in)
				if tt.wantAlias {
					var alias *mtgmatcher.AliasingError
					if !errors.As(err, &alias) {
						t.Fatalf("got %s, %v; want ambiguity", id, err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				co, err := mtgmatcher.GetUUID(id)
				if err != nil {
					t.Fatal(err)
				}
				if co.SetCode != "LTR" || co.Number != "399" || co.Foil != foil {
					t.Fatalf("got %s, want LTR #399 (foil %t)", co, foil)
				}
			})
		}
	}
}

func TestMatchCardRejectsConflictingIDs(t *testing.T) {
	realDatastore(t)
	// Without a year or number the wording permits both MagicFest printings.
	card := ABUCard{DisplayTitle: "Counterspell (MagicFest) - FOIL", Edition: "Promo", Language: []string{"English"}, ScryfallIDs: []string{"8916e24f-9c74-4b6c-9894-d60669854f35", "9cb2478c-1672-44eb-a9a1-a103fcdf3701"}}
	in, err := preprocess(&card)
	if err != nil {
		t.Fatal(err)
	}
	id, err := matchCard(&card, in)
	var alias *mtgmatcher.AliasingError
	if !errors.As(err, &alias) {
		t.Fatalf("got %s, %v; conflicting IDs must leave ambiguity", id, err)
	}
}

func TestMatchCardKeepsWordingOverStaleIDs(t *testing.T) {
	realDatastore(t)
	for _, tt := range []struct {
		card        ABUCard
		set, number string
	}{
		{ABUCard{DisplayTitle: "Plains (38)", Edition: "Venser vs. Koth", Number: "40", ScryfallIDs: []string{"65beef5d-df12-4e33-a585-b32f552b58a3"}, TCGplayerIDs: []int64{77315}}, "DDI", "38"},
		{ABUCard{DisplayTitle: "Viridian Longbow (The List)", Edition: "Adventures in the Forgotten Realms Commander", Number: "221", ScryfallIDs: []string{"f4607634-6206-4c6c-b389-abcbfe969b65"}, TCGplayerIDs: []int64{243897}}, "PLST", "AFC-221"},
		{ABUCard{DisplayTitle: "Sudden Setback (b - Black Bottle) - FOIL", Edition: "Murders at Karlov Manor", Number: "72", ScryfallIDs: []string{"0b9e5fd6-a5ea-4ae5-83f5-89ed6a658dd3"}, TCGplayerIDs: []int64{535972}}, "MKM", "72†"},
	} {
		t.Run(tt.card.DisplayTitle, func(t *testing.T) {
			in, err := preprocess(&tt.card)
			if err != nil {
				t.Fatal(err)
			}
			id, err := matchCard(&tt.card, in)
			if err != nil {
				t.Fatal(err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != tt.set || co.Number != tt.number {
				t.Fatalf("got %s, want %s #%s", co, tt.set, tt.number)
			}
		})
	}
}

func TestMatchCardIDsDoNotOverrideUnsupported(t *testing.T) {
	realDatastore(t)
	card := ABUCard{ScryfallIDs: []string{"f2a7042f-a6f0-4e77-86a2-5eb0d2587363"}}
	in := mtgmatcher.InputCard{Name: "Counterspell", Edition: "URL/Convention Promos", Variation: "2", Foil: true, Language: "Italian"}
	_, err := matchCard(&card, &in)
	if !errors.Is(err, mtgmatcher.ErrUnsupported) {
		t.Fatalf("got %v, want unsupported language", err)
	}
}
