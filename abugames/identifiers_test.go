package abugames

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// The fixtures are identity fields from the full Solr audit. They pin cases
// where IDs correct the text parser and cases where ABU copied both IDs from
// the wrong printing. Image-checked examples include Forest B Night, Plains
// (38), Master of Winds, Katerina, Bronzebeak Foragers and Triceraton Commander.
func TestPrimaryIdentifiers(t *testing.T) {
	realDatastore(t)
	data, err := os.ReadFile("testdata/identifier_listings.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		Group   string  `json:"group"`
		Card    ABUCard `json:"card"`
		Set     string  `json:"set"`
		Number  string  `json:"number"`
		Primary bool    `json:"primary"`
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, tt := range fixtures {
		t.Run(tt.Group+" "+tt.Card.DisplayTitle, func(t *testing.T) {
			in, err := preprocess(&tt.Card)
			if err != nil {
				t.Fatal(err)
			}
			if (in.ID != "") != tt.Primary {
				t.Errorf("ID path = %t, want %t", in.ID != "", tt.Primary)
			}
			id, err := matchCard(&tt.Card, in)
			if err != nil {
				t.Fatal(err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != tt.Set || co.Number != tt.Number {
				t.Errorf("got %s, want %s #%s", co, tt.Set, tt.Number)
			}
		})
	}
}

func TestPrimaryIdentifierNameValidation(t *testing.T) {
	realDatastore(t)
	card := ABUCard{
		DisplayTitle: "Counterspell (MagicFest) - FOIL", Edition: "Promo", Language: []string{"English"},
		// A valid but unrelated Scryfall card must not beat the correctly named
		// TCGplayer printing, nor make the pair appear irreconcilable.
		ScryfallIDs: []string{"0ecea0ba-da29-45f1-b72b-50b873309483"}, TCGplayerIDs: []int64{540987},
	}
	in, err := preprocess(&card)
	if err != nil {
		t.Fatal(err)
	}
	if in.ID == "" {
		t.Fatal("valid TCGplayer ID was not used")
	}
	id, err := mtgmatcher.Match(in)
	if err != nil {
		t.Fatal(err)
	}
	co, err := mtgmatcher.GetUUID(id)
	if err != nil {
		t.Fatal(err)
	}
	if co.Name != "Counterspell" || co.SetCode != "PF24" || co.Number != "1" || !co.Foil {
		t.Fatalf("got %s", co)
	}
}

// Catalog replay cases for promo descriptors and separately filed foil twins.
// Brass's Bounty IDs correct swapped text results; foil twins must be unique,
// and adding descriptor support must not resurrect an unprinted finish.
func TestIdentifierCoverage(t *testing.T) {
	realDatastore(t)
	data, err := os.ReadFile("testdata/identifier_coverage.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		Group   string  `json:"group"`
		Card    ABUCard `json:"card"`
		UUID    string  `json:"uuid"`
		Primary bool    `json:"primary"`
		Error   string  `json:"error"`
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, tt := range fixtures {
		t.Run(tt.Group+" "+tt.Card.DisplayTitle, func(t *testing.T) {
			in, err := preprocess(&tt.Card)
			if tt.Error != "" {
				if err == nil || err.Error() != tt.Error {
					t.Fatalf("error = %v, want %s", err, tt.Error)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if (in.ID != "") != tt.Primary {
				t.Errorf("ID path = %t, want %t", in.ID != "", tt.Primary)
			}
			id, err := matchCard(&tt.Card, in)
			if err != nil {
				t.Fatal(err)
			}
			if id != tt.UUID {
				t.Errorf("got %s, want %s", id, tt.UUID)
			}
		})
	}
}

func TestDescribedArtwork(t *testing.T) {
	realDatastore(t)
	data, err := os.ReadFile("testdata/artwork_listings.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		Group  string
		Card   ABUCard
		Number string
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, tt := range fixtures {
		t.Run(tt.Group, func(t *testing.T) {
			// The artwork must resolve both with IDs and when they are missing.
			for _, withIDs := range []bool{true, false} {
				card := tt.Card
				if !withIDs {
					card.ScryfallIDs = nil
					card.TCGplayerIDs = nil
					card.MultiverseIDs = nil
				}
				in, err := preprocess(&card)
				if err != nil {
					t.Fatal(err)
				}
				id, err := matchCard(&card, in)
				if err != nil {
					t.Fatal(err)
				}
				co, err := mtgmatcher.GetUUID(id)
				if err != nil {
					t.Fatal(err)
				}
				if co.SetCode != "ATQ" || co.Number != tt.Number {
					t.Errorf("with IDs %t: got %s, want ATQ %s", withIDs, co, tt.Number)
				}
			}
		})
	}
}

// Conflicting vendor IDs must not erase independently named printings. These
// records were checked against ABU's images, including copyright misprints and
// List stamps. The corrected families must also work without vendor IDs.
func TestConflictingIdentifierPrintings(t *testing.T) {
	realDatastore(t)
	data, err := os.ReadFile("testdata/conflicting_identifiers.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		Group      string  `json:"group"`
		Card       ABUCard `json:"card"`
		UUID       string  `json:"uuid"`
		WithoutIDs bool    `json:"without_ids"`
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, tt := range fixtures {
		t.Run(tt.Group+" "+tt.Card.DisplayTitle, func(t *testing.T) {
			for _, withoutIDs := range []bool{false, true} {
				if withoutIDs && !tt.WithoutIDs {
					continue
				}
				card := tt.Card
				if withoutIDs {
					card.ScryfallIDs = nil
					card.TCGplayerIDs = nil
					card.MultiverseIDs = nil
				}
				in, err := preprocess(&card)
				if err != nil {
					t.Fatal(err)
				}
				id, err := matchCard(&card, in)
				if err != nil {
					t.Fatal(err)
				}
				if id != tt.UUID {
					t.Errorf("without IDs %t: got %s, want %s", withoutIDs, id, tt.UUID)
				}
			}
		})
	}
}

// Raw listings checked against printed set numbers and promo artwork. Re-run
// without IDs so these cases exercise the text matcher as well as ID validation.
func TestResidueListings(t *testing.T) {
	realDatastore(t)
	data, err := os.ReadFile("testdata/residue_listings.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		Group string
		Card  ABUCard
		UUID  string
		Error string
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, tt := range fixtures {
		for _, withoutIDs := range []bool{false, true} {
			label := tt.Group
			if withoutIDs {
				label += " without IDs"
			}
			t.Run(label, func(t *testing.T) {
				card := tt.Card
				if withoutIDs {
					card.ScryfallIDs = nil
					card.TCGplayerIDs = nil
					card.MultiverseIDs = nil
				}
				in, err := preprocess(&card)
				id := ""
				if err == nil {
					id, err = matchCard(&card, in)
				}
				if tt.Error != "" {
					if err == nil || err.Error() != tt.Error {
						t.Fatalf("got %s, %v; want %s", id, err, tt.Error)
					}
				} else if err != nil || id != tt.UUID {
					t.Fatalf("got %s, %v; want %s", id, err, tt.UUID)
				}
			})
		}
	}
}

func TestListingCorrectionRequiresOriginalFields(t *testing.T) {
	card := ABUCard{ProductID: "8124578", DisplayTitle: "Path of Ancestry (The List)", Edition: "Outlaws of Thunder Junction Commander", Number: "310"}
	corrected := correctListing(card)
	if corrected.Edition != "Modern Horizons 3 Commander" || corrected.Number != "363" {
		t.Fatalf("correction missing: %+v", corrected)
	}
	for _, change := range []func(*ABUCard){
		func(c *ABUCard) { c.ProductID = "unrelated" },
		func(c *ABUCard) { c.DisplayTitle = "Path of Ancestry" },
		func(c *ABUCard) { c.Edition = "Commander 2017" },
		func(c *ABUCard) { c.Number = "56" },
	} {
		revised := card
		change(&revised)
		got := correctListing(revised)
		if got.Edition != revised.Edition || got.Number != revised.Number {
			t.Fatalf("revised listing overwritten: %+v", got)
		}
	}
}
