package cardkingdom

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/mtgban/go-cardkingdom"
	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

var (
	datastoreOnce    sync.Once
	datastoreErr     error
	datastoreBackend *mtgmatcher.Backend
)

// realDatastore reads the Magic datastore the first time a test asks for it,
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

var PriceListTest = `
[
    {
      "id": 320768,
      "sku": "PTLE-0305",
      "scryfall_id": null,
      "url": "mtg/promotional/enlightened-tutor-commanders-bundle-promo",
      "name": "Enlightened Tutor",
      "variation": "Commander's Bundle Promo",
      "edition": "Promotional",
      "is_foil": "false"
    }
]
`

var priceListResults = []string{
	"3b00adaa-962a-5316-9b9d-e12e0284f87f",
	"b30a3061-ce20-54eb-b25c-7520aa76f8b7",
}

func TestPreprocess(t *testing.T) {
	b := realDatastore(t)
	var products []cardkingdom.Product
	err := json.NewDecoder(strings.NewReader(PriceListTest)).Decode(&products)
	if err != nil {
		t.Errorf("FAIL: cannot umarshal products: %s", err)
		return
	}

	for i, product := range products {
		test := product
		idx := i
		t.Run(fmt.Sprint(test.Name), func(t *testing.T) {
			t.Parallel()

			theCard, err := Preprocess(b, test)
			if err != nil {
				t.Errorf("FAIL: unxpected Preprocess error: %s", err)
				return
			}

			cardID, err := b.Match(theCard)
			if err != nil {
				t.Errorf("FAIL: unxpected Match error: %s", err)
				return
			}

			if cardID != priceListResults[idx] {
				co, _ := b.GetUUID(cardID)
				t.Errorf("FAIL %s: Expected '%s' got '%s' (%s)", test.Name, priceListResults[idx], cardID, co)
				return
			}
			t.Log("PASS:", product.Name)
		})
	}
}

// TestPreprocessTokens pins the two token paths a silent revert would take
// back to the old behavior: the double-faced split must not double the
// " Token" suffix the kept face already carries, and a sku code the
// datastore does not carry must reach the filing set once its treatment
// wrapping is stripped.
func TestPreprocessTokens(t *testing.T) {
	b := realDatastore(t)
	for _, tt := range []struct {
		desc    string
		product cardkingdom.Product
		name    string
		setCode string
	}{
		{
			desc: "a dfc split keeps one Token suffix",
			product: cardkingdom.Product{
				SKU:     "TMID-0001",
				Name:    "Human Token // Wolf Token",
				Edition: "Innistrad: Midnight Hunt Tokens",
			},
			name:    "Human Token",
			setCode: "TMID",
		},
		{
			// The surge-foil wrapping is one the older per-treatment
			// strips never knew, so only the wrapping loop reaches it.
			desc: "a surge-foil-wrapped token code reaches the filing set",
			product: cardkingdom.Product{
				SKU:     "SFTWHO-0034",
				Name:    "Alien Token",
				Edition: "Doctor Who",
				IsFoil:  true,
			},
			name:    "Alien Token",
			setCode: "TWHO",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			theCard, err := Preprocess(b, tt.product)
			if err != nil {
				t.Fatalf("Preprocess(%v) = %v", tt.product, err)
			}
			if theCard.Name != tt.name {
				t.Errorf("Preprocess name = %q, want %q", theCard.Name, tt.name)
			}
			cardID, err := b.Match(theCard)
			if err != nil {
				t.Fatalf("Match(%v) = %v", theCard, err)
			}
			co, err := b.GetUUID(cardID)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", cardID, err)
			}
			if co.SetCode != tt.setCode {
				t.Errorf("Match(%v) = %s (%v), want a %s printing", theCard, cardID, co, tt.setCode)
			}
		})
	}
}

// TestPreprocessTokenPairing pins CK's own two-sided token wording resolving
// to the combined entity mtgmatcher/magic/tokenpairs.go derives for the same
// physical pairing, anchored by CK's own scryfall_id rather than guessed from
// the name alone. Commander 2018 carries the Angel/Cat pairing under the same
// scryfall_id CK ships for it.
func TestPreprocessTokenPairing(t *testing.T) {
	b := realDatastore(t)
	theCard, err := Preprocess(b, cardkingdom.Product{
		SKU:        "TC18-003",
		ScryfallID: "6ac609aa-49d1-4330-b718-a90b0560da52",
		Name:       "Angel Token - Cat Token",
		Edition:    "Commander 2018",
	})
	if err != nil {
		t.Fatalf("Preprocess: %v", err)
	}
	cardID, err := b.Match(theCard)
	if err != nil {
		t.Fatalf("Match(%v) = %v", theCard, err)
	}
	const want = "3695fffc-7192-528c-83c8-b9893e6c4aa1_tp_dfd98280-5bf8-5d7d-a4ed-a90b91be8734"
	if cardID != want {
		co, _ := b.GetUUID(cardID)
		t.Errorf("Match(%v) = %s (%v), want the derived Angel // Cat pairing", theCard, cardID, co)
	}
}

// TestPreprocessTokenPairingParenthetical pins normalizeTokenFace stripping
// an artist parenthetical and the " Token" suffix regardless of which order
// CK's own wording puts them in - "X Token (Artist)" needs the parenthetical
// gone before the suffix trim can ever find it at the end of the string.
func TestPreprocessTokenPairingParenthetical(t *testing.T) {
	b := realDatastore(t)
	theCard, err := Preprocess(b, cardkingdom.Product{
		SKU:        "TPCA-002",
		ScryfallID: "b71177f3-a7cb-4d38-b8c7-daa5b8266a19",
		Name:       "Eldrazi Spawn Token (Briclot) - Eldrazi Token (Proce)",
		Edition:    "Planechase Anthology",
	})
	if err != nil {
		t.Fatalf("Preprocess: %v", err)
	}
	cardID, err := b.Match(theCard)
	if err != nil {
		t.Fatalf("Match(%v) = %v", theCard, err)
	}
	const want = "89c709c6-2fa7-5408-a5bf-5762b82d5477_tp_9d1ed5df-e69a-5bce-96fc-20dce8047d8a"
	if cardID != want {
		co, _ := b.GetUUID(cardID)
		t.Errorf("Match(%v) = %s (%v), want the derived Eldrazi // Eldrazi Spawn pairing", theCard, cardID, co)
	}
}

// TestPreprocessTokenPairingSecondHalfAnchored pins matchTokenPairing
// recognizing CK's scryfall_id anchoring the SECOND half of its own listing
// name rather than the first - "Cat Token - Cat Warrior Token" carries CK's
// id against the Cat Warrior face, so the Cat half is the partner to look
// up, not the other way the naming order would otherwise suggest.
func TestPreprocessTokenPairingSecondHalfAnchored(t *testing.T) {
	b := realDatastore(t)
	theCard, err := Preprocess(b, cardkingdom.Product{
		SKU:        "TC17-001",
		ScryfallID: "29c4e4f2-0040-4490-b357-660d729ad9cc",
		Name:       "Cat Token - Cat Warrior Token",
		Edition:    "Commander 2017",
	})
	if err != nil {
		t.Fatalf("Preprocess: %v", err)
	}
	cardID, err := b.Match(theCard)
	if err != nil {
		t.Fatalf("Match(%v) = %v", theCard, err)
	}
	const want = "7a13db1f-523c-5b19-80e5-d4d6f0121c6b_tp_7e5dc858-2163-5de0-95cb-f0e2933a7f7f"
	if cardID != want {
		co, _ := b.GetUUID(cardID)
		t.Errorf("Match(%v) = %s (%v), want the derived Cat // Cat Warrior pairing", theCard, cardID, co)
	}
}

// TestPreprocessTokenPairingBySetNumber pins matchTokenPairingBySetNumber:
// a "Mystery Booster/The List" listing that bundles two independently
// numbered token-sheet entries into one CK sku never carries a scryfall_id
// (no vendor sells that exact ad-hoc pairing as one product), so the first
// face is anchored by its own sku-derived set and number instead.
func TestPreprocessTokenPairingBySetNumber(t *testing.T) {
	b := realDatastore(t)
	theCard, err := Preprocess(b, cardkingdom.Product{
		SKU:     "MTMKC-0012",
		Name:    "Kobolds of Kher Keep Token // Soldier Token",
		Edition: "Mystery Booster/The List",
	})
	if err != nil {
		t.Fatalf("Preprocess: %v", err)
	}
	cardID, err := b.Match(theCard)
	if err != nil {
		t.Fatalf("Match(%v) = %v", theCard, err)
	}
	const want = "6aabcac2-f43d-5895-99fc-68c948e41e34_tp_b39718f3-8bdc-5726-9b59-c690d962b222"
	if cardID != want {
		co, _ := b.GetUUID(cardID)
		t.Errorf("Match(%v) = %s (%v), want the derived Soldier // Kobolds of Kher Keep Token pairing", theCard, cardID, co)
	}
}

// TestPreprocessSurgeFoilStarredDuplicate pins the skuFixupTable entries
// redirecting these two Warhammer 40,000 surge-foil skus to the ★-suffixed
// duplicate number the token set actually catalogues as the foil printing -
// the bare number these skus name is filed nonfoil-only.
func TestPreprocessSurgeFoilStarredDuplicate(t *testing.T) {
	b := realDatastore(t)
	for _, tt := range []struct {
		sku  string
		name string
		want string
	}{
		{"SFT40K-015", "Plaguebearer of Nurgle Token // Astartes Warrior Token", "cfce7c69-c5d6-565d-8661-efa842b3147d"},
		{"SFT40K-016", "Spawn Token // Plaguebearer of Nurgle Token", "014c729a-bbdc-5569-afe8-7155ec517de7"},
	} {
		t.Run(tt.sku, func(t *testing.T) {
			theCard, err := Preprocess(b, cardkingdom.Product{
				SKU:     tt.sku,
				Name:    tt.name,
				Edition: "Warhammer 40,000",
				IsFoil:  true,
			})
			if err != nil {
				t.Fatalf("Preprocess: %v", err)
			}
			cardID, err := b.Match(theCard)
			if err != nil {
				t.Fatalf("Match(%v) = %v", theCard, err)
			}
			if cardID != tt.want {
				co, _ := b.GetUUID(cardID)
				t.Errorf("Match(%v) = %s (%v), want the ★-suffixed foil printing", theCard, cardID, co)
			}
		})
	}
}

// TestPreprocessListAngelToken pins the sku fixup for the one Angel token
// The List carries twice. Its sku names the Forgotten Realms printing and
// its bare number reaches the Guilds of Ravnica one, which is the wrong
// card at the right number.
func TestPreprocessListAngelToken(t *testing.T) {
	b := realDatastore(t)
	theCard, err := Preprocess(b, cardkingdom.Product{
		SKU:     "MTAFR-001",
		Name:    "Angel Token // Spirit Token",
		Edition: "Mystery Booster/The List",
	})
	if err != nil {
		t.Fatalf("Preprocess: %v", err)
	}
	cardID, err := b.Match(theCard)
	if err != nil {
		t.Fatalf("Match(%v) = %v", theCard, err)
	}
	const want = "ba22fdaf-8d82-5f14-a5f0-3e5908f04d8c"
	if cardID != want {
		co, _ := b.GetUUID(cardID)
		t.Errorf("Match(%v) = %s (%v), want the Forgotten Realms Angel", theCard, cardID, co)
	}
}

// TestPreprocessEmblems pins the three shapes CK spells an emblem in, each
// against the uuid the datastore files at the address the sku names: the
// planeswalker left in the variation, the planeswalker abbreviated into a
// parenthetical on a card shared with another token, and the Mythic Edition
// numbering that diverges from mtgjson's so only the name can carry the row.
func TestPreprocessEmblems(t *testing.T) {
	b := realDatastore(t)
	for _, tt := range []struct {
		desc    string
		product cardkingdom.Product
		name    string
		uuid    string
	}{
		{
			desc: "the planeswalker rides in the variation",
			product: cardkingdom.Product{
				SKU:       "TDKA-003",
				Name:      "Emblem",
				Edition:   "Dark Ascension",
				Variation: "Sorin, Lord of Innistrad",
			},
			name: "Sorin, Lord of Innistrad Emblem",
			uuid: "1d0792f5-6ed6-5385-9a96-87b472909d1c",
		},
		{
			desc: "a parenthetical short name against the full one",
			product: cardkingdom.Product{
				SKU:     "TC14-035",
				Name:    "Emblem (Nixilis) - Zombie (Black) Token",
				Edition: "Commander 2014",
			},
			name: "Ob Nixilis of the Black Oath Emblem",
			uuid: "5f895769-4c30-5d4e-a7ad-51ebb7cbf63e",
		},
		{
			// CK numbers this G6 printing 006A, so the number cannot
			// reach it and the respelled name has to.
			desc: "a number the set does not carry",
			product: cardkingdom.Product{
				SKU:       "TMED-006A",
				Name:      "Emblem",
				Edition:   "Masterpiece Series: Mythic Edition",
				Variation: "Ral",
			},
			name: "Ral, Izzet Viceroy Emblem",
			uuid: "2bcf14e3-ff8f-58b7-acbd-9f96dd49ee6d",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			theCard, err := Preprocess(b, tt.product)
			if err != nil {
				t.Fatalf("Preprocess(%v) = %v", tt.product, err)
			}
			if theCard.Name != tt.name {
				t.Errorf("Preprocess name = %q, want %q", theCard.Name, tt.name)
			}
			cardID, err := b.Match(theCard)
			if err != nil {
				t.Fatalf("Match(%v) = %v", theCard, err)
			}
			if cardID != tt.uuid {
				co, _ := b.GetUUID(cardID)
				t.Errorf("Match(%v) = %s (%v), want %s", theCard, cardID, co, tt.uuid)
			}
		})
	}
}

// TestPreprocessSplitCard pins the split cards a T-prefixed set code used to
// sweep into the double-faced token split, which renamed them after their
// first face and lost the row.
func TestPreprocessSplitCard(t *testing.T) {
	b := realDatastore(t)
	theCard, err := Preprocess(b, cardkingdom.Product{
		SKU:     "TSR-186",
		Name:    "Rough // Tumble",
		Edition: "Time Spiral Remastered",
	})
	if err != nil {
		t.Fatalf("Preprocess: %v", err)
	}
	if theCard.Name != "Rough // Tumble" {
		t.Errorf("Preprocess name = %q, want the whole split card", theCard.Name)
	}
	cardID, err := b.Match(theCard)
	if err != nil {
		t.Fatalf("Match(%v) = %v", theCard, err)
	}
	const want = "609b3e64-4e46-595c-a99d-bcbb04691d4f"
	if cardID != want {
		co, _ := b.GetUUID(cardID)
		t.Errorf("Match(%v) = %s (%v), want the Time Spiral Remastered split card", theCard, cardID, co)
	}
}

// TestPreprocessTexturedFoilSplitCard pins the "TF" textured-foil prefix
// being unwrapped before token-sheet status is decided: the wrapped code
// "TFOTP" still starts with the token prefix "T", and without the unwrap a
// split card's textured foil reads as a two-faced token sheet and loses its
// second name entirely.
func TestPreprocessTexturedFoilSplitCard(t *testing.T) {
	b := realDatastore(t)
	theCard, err := Preprocess(b, cardkingdom.Product{
		SKU:        "TFOTP-0075",
		ScryfallID: "301f6df1-1b97-4a63-8043-8b97147b200b",
		Name:       "Crime // Punishment",
		Variation:  "0075 - Textured Foil",
		Edition:    "Outlaws of Thunder Junction Breaking News",
		IsFoil:     true,
	})
	if err != nil {
		t.Fatalf("Preprocess: %v", err)
	}
	if theCard.Name != "Crime // Punishment" {
		t.Errorf("Preprocess name = %q, want the whole split card", theCard.Name)
	}
	cardID, err := b.Match(theCard)
	if err != nil {
		t.Fatalf("Match(%v) = %v", theCard, err)
	}
	const want = "94f58721-0540-530a-a179-bb7240b5b2e3"
	if cardID != want {
		co, _ := b.GetUUID(cardID)
		t.Errorf("Match(%v) = %s (%v), want the OTP textured foil printing", theCard, cardID, co)
	}
}

// TestMatchPrereleaseSKU pins matchPrereleaseSKU rescuing a PREL sku Match
// denies as unsupported: CK titles Force of Nature "Prerelease Foil" but the
// row it sells is the Release promo, which the prerelease tag on the
// request refuses. A non-PREL sku, or a PREL sku whose id resolves to a
// printing that is not a Release promo, must still refuse.
func TestMatchPrereleaseSKU(t *testing.T) {
	b := realDatastore(t)
	t.Run("a PREL sku rescues to its Release promo", func(t *testing.T) {
		cardID, err := matchPrereleaseSKU(b, "PREL-005", "4638a30d-48e9-42cb-bf5f-001b4259391c", true)
		if err != nil {
			t.Fatalf("matchPrereleaseSKU: %v", err)
		}
		const want = "1e5da536-89b0-5682-8ac2-c48ccd4853e6"
		if cardID != want {
			co, _ := b.GetUUID(cardID)
			t.Errorf("matchPrereleaseSKU = %s (%v), want the P9ED release promo", cardID, co)
		}
	})
	t.Run("a non-PREL sku is refused", func(t *testing.T) {
		_, err := matchPrereleaseSKU(b, "PLST-005", "4638a30d-48e9-42cb-bf5f-001b4259391c", true)
		if !errors.Is(err, mtgmatcher.ErrUnsupported) {
			t.Errorf("matchPrereleaseSKU = %v, want ErrUnsupported", err)
		}
	})
	t.Run("a PREL sku whose id is not a Release promo is refused", func(t *testing.T) {
		_, err := matchPrereleaseSKU(b, "PREL-999", "72449552-aa2c-4ae3-846f-df523c5e6078", false)
		if !errors.Is(err, mtgmatcher.ErrUnsupported) {
			t.Errorf("matchPrereleaseSKU = %v, want ErrUnsupported", err)
		}
	})
}

func TestPreprocessTokenFoilRefused(t *testing.T) {
	b := realDatastore(t)
	for _, tt := range []struct {
		desc    string
		product cardkingdom.Product
	}{
		{
			// The sheet holds one nonfoil emblem, so the foil row would
			// be served as the price of the plain one
			desc: "a foil-wrapped code whose sheet was never sold foil",
			product: cardkingdom.Product{
				SKU:     "FTNEO-019",
				Name:    "Tezzeret, Betrayer of Flesh Emblem",
				Edition: "Kamigawa: Neon Dynasty",
				IsFoil:  true,
			},
		},
		{
			desc: "the same for a surge-foil wrapping",
			product: cardkingdom.Product{
				SKU:     "SFT40K-011",
				Name:    "Arco-Flagellant Token // Soldier Token",
				Edition: "Warhammer 40,000",
				IsFoil:  true,
			},
		},
		{
			// No real card is named Astartes Warrior, so the sheet files
			// the token without the suffix the row spells it with
			desc: "a token the datastore files without a Token suffix",
			product: cardkingdom.Product{
				SKU:     "SFT40K-012",
				Name:    "Astartes Warrior Token // Spawn Token",
				Edition: "Warhammer 40,000",
				IsFoil:  true,
			},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			theCard, err := Preprocess(b, tt.product)
			if !errors.Is(err, mtgmatcher.ErrUnsupported) {
				t.Errorf("Preprocess(%v) = %v, %v, want ErrUnsupported", tt.product, theCard, err)
			}
		})
	}
}

func TestUnindexedTokenSheet(t *testing.T) {
	b := realDatastore(t)
	for _, tt := range []struct {
		sku  string
		want bool
	}{
		// The Jumpstart sheets the datastore has no set for
		{"TJMP-001", true},
		{"TJ22-014", true},
		// A sheet it does carry, so a failure there is worth hearing
		{"TNCC-021", false},
		{"THOU-004", false},
		// Not a sheet at all
		{"JMP-001", false},
		{"PLST-TAFR-1", false},
		{"CMB1-042", false},
		{"nosku", false},
	} {
		t.Run(tt.sku, func(t *testing.T) {
			if got := unindexedTokenSheet(b, tt.sku); got != tt.want {
				t.Errorf("unindexedTokenSheet(%q) = %v, want %v", tt.sku, got, tt.want)
			}
		})
	}
}
