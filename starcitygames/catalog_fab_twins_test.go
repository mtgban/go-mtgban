package starcitygames

import (
	"errors"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestResolveFleshAndBloodMarvelTwins pins the two shelves where SCG's own
// catalog fields send two different printings to the same match: a plain
// rarity landing on its Marvel twin because nothing steered it away, and a
// renamed foreign printing landing on the English card whose name it
// borrows. Fixtures are copied from the export verbatim.
func TestResolveFleshAndBloodMarvelTwins(t *testing.T) {
	b := withGameDatastore(t, "fleshandblood", "FLESHANDBLOOD_PATH")

	for _, tt := range []struct {
		desc                  string
		p                     CatalogProduct
		wantNumber            string
		wantMarvel, wantMaori bool
	}{
		{
			desc: "a plain rarity is that plain printing",
			p: CatalogProduct{
				SKU: "SGL-FAB-UPR-042b_043b-ENC", Name: "Aether Ashwing // Ash",
				Game: "Flesh and Blood", Set: "Uprising", Rarity: "Token",
				Finish: "Cold Foil", FinishGroup: "Alt Foil",
				Language: "English", CollectorNumber: "042b",
				ProductType: ProductTypeSingles,
			},
			wantNumber: "UPR043", wantMarvel: false, wantMaori: false,
		},
		{
			desc: "not the Marvel twin at the same number",
			p: CatalogProduct{
				SKU: "SGL-FAB-UPR2-042_043-ENC", Name: "Aether Ashwing // Ash",
				Game: "Flesh and Blood", Set: "Uprising", Rarity: "Marvel",
				Finish: "Cold Foil", FinishGroup: "Alt Foil",
				Language: "English", CollectorNumber: "042",
				ProductType: ProductTypeSingles,
			},
			wantNumber: "UPR043", wantMarvel: true, wantMaori: false,
		},
		{
			desc: "the English printing is the name it sells under",
			p: CatalogProduct{
				SKU: "SGL-FAB-SUP2-009-ENC", Name: "Pleiades, Superstar",
				Game: "Flesh and Blood", Set: "Super Slam", Rarity: "Marvel",
				Finish: "Cold Foil", FinishGroup: "Alt Foil",
				Language: "English", CollectorNumber: "009",
				ProductType: ProductTypeSingles,
			},
			wantNumber: "SUP009", wantMarvel: true, wantMaori: false,
		},
		{
			desc: "the Maori one is not, though the sku sends the same name",
			p: CatalogProduct{
				SKU: "SGL-FAB-SUP2-009b-ENC", Name: "Pleiades, Superstar",
				Game: "Flesh and Blood", Set: "Super Slam", Rarity: "Marvel",
				Finish: "Cold Foil", FinishGroup: "Alt Foil",
				Language: "English", CollectorNumber: "009",
				ProductType: ProductTypeSingles,
			},
			wantNumber: "SUP009", wantMarvel: true, wantMaori: true,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			id, err := resolveProduct(b, GameFleshAndBlood, tt.p)
			if err != nil {
				t.Fatalf("resolveProduct(%s) = %v", tt.p.SKU, err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.Number != tt.wantNumber {
				t.Errorf("%s resolved to number %s, want %s", tt.p.SKU, co.Number, tt.wantNumber)
			}
			if got := co.HasPromoType("marvel"); got != tt.wantMarvel {
				t.Errorf("%s resolved to marvel=%v, want %v", tt.p.SKU, got, tt.wantMarvel)
			}
			if got := co.HasPromoType("pleiadessuperstar"); got != tt.wantMaori {
				t.Errorf("%s resolved to the Maori printing=%v, want %v (name=%q)", tt.p.SKU, got, tt.wantMaori, co.Name)
			}
		})
	}
}

// requireCredit skips a case whose premise the installed datastore does not
// hold. datastore-gen only started publishing Flesh and Blood's artist field
// once this fix needed it to steer by, and the datastore a checkout carries
// may still predate that: against such a copy fabCreditedTwin has nothing to
// match either uuid's Artist against and correctly leaves both alone, which
// is not this test's premise to assert against.
func requireCredit(t *testing.T, b *mtgmatcher.Backend, uuid string) {
	t.Helper()
	co, err := b.GetUUID(uuid)
	if err != nil || co.Artist == "" {
		t.Skipf("the installed Flesh and Blood datastore does not carry an artist for %s yet", uuid)
	}
}

// TestResolveFleshAndBloodCreditedTwins pins the shelf where the catalog's
// name is not enough either: SCG sells two artist-commissioned Auroras at
// ROS008 under names that fold to the same match once "(Marvel)" is read
// off, and only datastore-gen's own credit field - carried nowhere else in
// this product - says which sku is which. Fixtures are copied from the
// export verbatim.
func TestResolveFleshAndBloodCreditedTwins(t *testing.T) {
	b := withGameDatastore(t, "fleshandblood", "FLESHANDBLOOD_PATH")
	requireCredit(t, b, "ros008_565391_coldfoil")

	for _, tt := range []struct {
		desc       string
		p          CatalogProduct
		wantArtist string
	}{
		{
			desc: "008a is credited to Asur Misoa",
			p: CatalogProduct{
				SKU: "SGL-FAB-ROS2-008a-ENC", Name: "Aurora",
				Game: "Flesh and Blood", Set: "Rosetta", Rarity: "Marvel",
				Finish: "Cold Foil", FinishGroup: "Alt Foil",
				Language: "English", CollectorNumber: "008",
				ProductType: ProductTypeSingles,
			},
			wantArtist: "Asur Misoa",
		},
		{
			desc: "008b to Ramza Ardyputra",
			p: CatalogProduct{
				SKU: "SGL-FAB-ROS2-008b-ENC", Name: "Aurora",
				Game: "Flesh and Blood", Set: "Rosetta", Rarity: "Marvel",
				Finish: "Cold Foil", FinishGroup: "Alt Foil",
				Language: "English", CollectorNumber: "008",
				ProductType: ProductTypeSingles,
			},
			wantArtist: "Ramza Ardyputra",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			id, err := resolveProduct(b, GameFleshAndBlood, tt.p)
			if err != nil {
				t.Fatalf("resolveProduct(%s) = %v", tt.p.SKU, err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.Artist != tt.wantArtist {
				t.Errorf("%s resolved to artist %q, want %q", tt.p.SKU, co.Artist, tt.wantArtist)
			}
		})
	}

	t.Run("the pair lands on two different uuids", func(t *testing.T) {
		idA, errA := resolveProduct(b, GameFleshAndBlood, CatalogProduct{
			SKU: "SGL-FAB-ROS2-008a-ENC", Name: "Aurora",
			Game: "Flesh and Blood", Set: "Rosetta", Rarity: "Marvel",
			Finish: "Cold Foil", FinishGroup: "Alt Foil",
			Language: "English", CollectorNumber: "008",
			ProductType: ProductTypeSingles,
		})
		idB, errB := resolveProduct(b, GameFleshAndBlood, CatalogProduct{
			SKU: "SGL-FAB-ROS2-008b-ENC", Name: "Aurora",
			Game: "Flesh and Blood", Set: "Rosetta", Rarity: "Marvel",
			Finish: "Cold Foil", FinishGroup: "Alt Foil",
			Language: "English", CollectorNumber: "008",
			ProductType: ProductTypeSingles,
		})
		if errA != nil || errB != nil {
			t.Fatalf("resolveProduct errors: %v, %v", errA, errB)
		}
		if idA == idB {
			t.Errorf("008a and 008b resolved to the same uuid %s", idA)
		}
	})
}

// TestFleshAndBloodDuplicateStockRefused pins the shelf that is refused
// rather than steered: Sanctuary of Aria's "027a" and "027b" both sell the
// one ROS027 token the datastore carries, at their own price, and nothing
// says which is the card's price - so both are unsupported rather than
// letting whichever streams first win.
func TestFleshAndBloodDuplicateStockRefused(t *testing.T) {
	b := withGameDatastore(t, "fleshandblood", "FLESHANDBLOOD_PATH")

	for _, sku := range []string{"SGL-FAB-ROS-027a-ENN", "SGL-FAB-ROS-027b-ENN"} {
		t.Run(sku, func(t *testing.T) {
			p := CatalogProduct{
				SKU: sku, Name: "Sanctuary of Aria",
				Game: "Flesh and Blood", Set: "Rosetta", Rarity: "Token",
				Finish: "Non-foil", FinishGroup: "Non-foil",
				Language: "English", CollectorNumber: "027",
				ProductType: ProductTypeSingles,
			}
			_, err := resolveProduct(b, GameFleshAndBlood, p)
			if !errors.Is(err, mtgmatcher.ErrUnsupported) {
				t.Errorf("resolveProduct(%s) = %v, want ErrUnsupported", sku, err)
			}
		})
	}
}
