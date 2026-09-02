package lorcana

import (
	"slices"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// sealedFixture is a minimal datastore carrying one card and two sealed
// products: one in the card's own set, one in a promotional group whose
// set entry the builder mints. It pins the loader's side of the
// lorcana-datastore contract without needing a real datastore file.
const sealedFixture = `{
	"metadata": {"formatVersion": "2.0.0", "language": "en"},
	"sets": {
		"1": {"name": "The First Chapter", "releaseDate": "2023-08-18", "type": "expansion", "number": 1},
		"D23": {"name": "D23 Promos", "releaseDate": "2022-09-11", "type": "promo"}
	},
	"cards": [
		{
			"id": 101,
			"fullName": "Fixture Mouse - Brave Tailor",
			"name": "Fixture Mouse",
			"setCode": "1",
			"number": 1,
			"rarity": "Common",
			"foilTypes": ["None", "Silver"],
			"externalLinks": {"tcgPlayerId": 100001}
		}
	],
	"sealed": [
		{
			"id": "1-600001",
			"name": "The First Chapter Booster Pack",
			"setCode": "1",
			"releaseDate": "2023-08-18",
			"image": "https://example.com/pack.jpg",
			"externalLinks": {"tcgPlayerId": 600001}
		},
		{
			"id": "d23-600002",
			"name": "D23 Collector Set",
			"setCode": "D23",
			"releaseDate": "2022-09-11",
			"image": "https://example.com/d23.jpg",
			"externalLinks": {"tcgPlayerId": 600002}
		}
	]
}`

func loadSealedFixture(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	b, err := Load(strings.NewReader(sealedFixture))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestSealedLoads(t *testing.T) {
	b := loadSealedFixture(t)

	if len(b.AllSealedUUIDs) != 2 {
		t.Fatalf("AllSealedUUIDs = %v, want 2 entries", b.AllSealedUUIDs)
	}
	co, err := b.GetUUID("1-600001")
	if err != nil {
		t.Fatal(err)
	}
	if !co.Sealed {
		t.Error("sealed product not marked Sealed")
	}
	if co.Rarity != "product" {
		t.Errorf("Rarity = %q, want %q", co.Rarity, "product")
	}
	if co.Edition != "The First Chapter" {
		t.Errorf("Edition = %q, want %q", co.Edition, "The First Chapter")
	}
	if got := co.Identifiers["tcgplayerProductId"]; got != "600001" {
		t.Errorf("tcgplayerProductId = %q, want %q", got, "600001")
	}
}

func TestSealedSetBuckets(t *testing.T) {
	b := loadSealedFixture(t)

	if got := b.SetSealedUUIDs["1"]; len(got) != 1 || got[0] != "1-600001" {
		t.Errorf("SetSealedUUIDs[1] = %v, want [1-600001]", got)
	}
	if got := len(b.Sets["1"].SealedProduct); got != 1 {
		t.Errorf("Sets[1].SealedProduct has %d entries, want 1", got)
	}

	// The promo group's set entry comes minted from the builder
	set, found := b.Sets["D23"]
	if !found {
		t.Fatal("minted set D23 not loaded")
	}
	if set.Name != "D23 Promos" {
		t.Errorf("D23 name = %q, want %q", set.Name, "D23 Promos")
	}
	if got := b.SetSealedUUIDs["D23"]; len(got) != 1 || got[0] != "d23-600002" {
		t.Errorf("SetSealedUUIDs[D23] = %v, want [d23-600002]", got)
	}
}

func TestSealedProductMap(t *testing.T) {
	b := loadSealedFixture(t)

	productMap := b.BuildSealedProductMap("tcgplayerProductId")
	if got := productMap[600001]; len(got) != 1 || got[0] != "1-600001" {
		t.Errorf("productMap[600001] = %v, want [1-600001]", got)
	}
	if got := productMap[600002]; len(got) != 1 || got[0] != "d23-600002" {
		t.Errorf("productMap[600002] = %v, want [d23-600002]", got)
	}
	// The single's product id belongs to the card namespace, not this one
	if got, found := productMap[100001]; found {
		t.Errorf("productMap[100001] = %v, want absent", got)
	}
}

func TestSealedSearch(t *testing.T) {
	b := loadSealedFixture(t)

	uuids, err := b.SearchSealedEquals("The First Chapter Booster Pack")
	if err != nil {
		t.Fatal(err)
	}
	if len(uuids) != 1 || uuids[0] != "1-600001" {
		t.Errorf("SearchSealedEquals = %v, want [1-600001]", uuids)
	}
}

func TestSealedOutsideExternalIdentifiers(t *testing.T) {
	b := loadSealedFixture(t)

	// Sealed ids resolve through BuildSealedProductMap alone; the external
	// index stays cards-only so MatchID cannot hand a sealed uuid to a
	// singles scraper
	if uuid, found := b.ExternalIdentifiers[mtgmatcher.IDSpaceTCGplayer]["600001"]; found {
		t.Errorf("ExternalIdentifiers[600001] = %q, want absent", uuid)
	}
	if _, found := b.ExternalIdentifiers[mtgmatcher.IDSpaceTCGplayer]["100001"]; !found {
		t.Error("the single's product id should stay in ExternalIdentifiers")
	}
}

func TestSealedNameCollidingWithCard(t *testing.T) {
	// A sealed product named exactly like a card: both vocabularies come
	// from commerce naming, so nothing prevents the collision. The card
	// must keep matching and the sealed product must stay searchable.
	fixture := strings.Replace(sealedFixture,
		`"name": "The First Chapter Booster Pack"`,
		`"name": "Fixture Mouse - Brave Tailor"`, 1)
	b, err := Load(strings.NewReader(fixture))
	if err != nil {
		t.Fatal(err)
	}

	uuid, err := b.Match(&mtgmatcher.InputCard{Name: "Fixture Mouse - Brave Tailor", Variation: "1"})
	if err != nil {
		t.Fatalf("Match = %v, want the card", err)
	}
	if uuid != "101" {
		t.Errorf("Match = %q, want %q", uuid, "101")
	}

	uuids, err := b.SearchSealedEquals("Fixture Mouse - Brave Tailor")
	if err != nil {
		t.Fatalf("SearchSealedEquals = %v, want the sealed product", err)
	}
	if !slices.Contains(uuids, "1-600001") {
		t.Errorf("SearchSealedEquals = %v, want it to contain 1-600001", uuids)
	}
}

func TestSealedAbsentSectionLoads(t *testing.T) {
	// A datastore built from the plain upstream file keeps loading, with
	// nothing in the sealed namespace
	fixture := strings.Replace(sealedFixture, `"sealed"`, `"ignored"`, 1)
	b, err := Load(strings.NewReader(fixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(b.AllSealedUUIDs) != 0 {
		t.Errorf("AllSealedUUIDs = %v, want empty", b.AllSealedUUIDs)
	}
}

func TestSealedZeroProductId(t *testing.T) {
	// A product the builder could not link to TCGplayer ships without an
	// id. Stamping the zero value would give BuildSealedProductMap a
	// shared key 0 for every unlinked storefront listing to funnel onto.
	fixture := strings.Replace(sealedFixture,
		`"externalLinks": {"tcgPlayerId": 600002}`, `"externalLinks": {"tcgPlayerId": 0}`, 1)
	b, err := Load(strings.NewReader(fixture))
	if err != nil {
		t.Fatal(err)
	}

	co, err := b.GetUUID("d23-600002")
	if err != nil {
		t.Fatal(err)
	}
	if got, found := co.Identifiers["tcgplayerProductId"]; found {
		t.Errorf("tcgplayerProductId = %q, want absent", got)
	}

	productMap := b.BuildSealedProductMap("tcgplayerProductId")
	if got, found := productMap[0]; found {
		t.Errorf("productMap[0] = %v, want absent", got)
	}
}
