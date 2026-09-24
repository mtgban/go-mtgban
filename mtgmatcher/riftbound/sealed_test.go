package riftbound

import (
	"slices"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// sealedFixture is a minimal datastore carrying one card and two sealed
// products: one in the card's own set, one in a set the builder mints for a
// group sold only sealed. It pins the loader's side of the builder contract
// without needing a real datastore file.
const sealedFixture = `{"data": {
	"pageProps": {"page": {"blades": [{
		"type": "riftboundCardGallery",
		"sets": {"items": [
			{"id": "OGN", "name": "Origins", "baseSetSize": 298},
			{"id": "ACC", "name": "Accessories"}
		]},
		"cards": {"items": [
			{
				"id": "ogn-001",
				"name": "Fixture Card",
				"number": "001",
				"setCode": "OGN",
				"rarity": {"value": {"id": "common"}},
				"externalLinks": {"tcgPlayerId": 100001},
				"printings": [{"finish": "Normal", "id": "ogn-001_nonfoil"}]
			}
		]},
		"sealed": {"items": [
			{
				"id": "ogn-600001",
				"name": "Origins Booster Box",
				"setCode": "OGN",
				"image": "https://example.com/box.jpg",
				"externalLinks": {"tcgPlayerId": 600001}
			},
			{
				"id": "acc-600002",
				"name": "Playmat Bundle",
				"setCode": "ACC",
				"image": "https://example.com/mat.jpg",
				"externalLinks": {"tcgPlayerId": 600002}
			}
		]}
	}]}}
}}`

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
	co, err := b.GetUUID("ogn-600001")
	if err != nil {
		t.Fatal(err)
	}
	if !co.Sealed {
		t.Error("sealed product not marked Sealed")
	}
	if co.Rarity != "product" {
		t.Errorf("Rarity = %q, want %q", co.Rarity, "product")
	}
	if co.Edition != "Origins" {
		t.Errorf("Edition = %q, want %q", co.Edition, "Origins")
	}
	if got := co.Identifiers["tcgplayerProductId"]; got != "600001" {
		t.Errorf("tcgplayerProductId = %q, want %q", got, "600001")
	}
}

// TestCardSetBucket guards the regression this loader once had: SetUUIDs
// stayed nil because nothing built it, so GetUUIDsInSet silently answered
// empty for every set - which is what mtgban-website's edition-only
// searches (s:CODE) seed candidates from exclusively when there is no text
// to search, so those searches came back empty for Riftbound entirely.
func TestCardSetBucket(t *testing.T) {
	b := loadSealedFixture(t)

	if got := b.SetUUIDs["OGN"]; len(got) != 1 || got[0] != "ogn-001_nonfoil" {
		t.Errorf("SetUUIDs[OGN] = %v, want [ogn-001_nonfoil]", got)
	}
}

func TestSealedSetBuckets(t *testing.T) {
	b := loadSealedFixture(t)

	if got := b.SetSealedUUIDs["OGN"]; len(got) != 1 || got[0] != "ogn-600001" {
		t.Errorf("SetSealedUUIDs[OGN] = %v, want [ogn-600001]", got)
	}
	if got := len(b.Sets["OGN"].SealedProduct); got != 1 {
		t.Errorf("Sets[OGN].SealedProduct has %d entries, want 1", got)
	}

	// The accessories set holds no card, only its sealed product
	set, found := b.Sets["ACC"]
	if !found {
		t.Fatal("sealed-only set ACC not loaded")
	}
	if set.Name != "Accessories" {
		t.Errorf("ACC name = %q, want %q", set.Name, "Accessories")
	}
	if got := b.SetSealedUUIDs["ACC"]; len(got) != 1 || got[0] != "acc-600002" {
		t.Errorf("SetSealedUUIDs[ACC] = %v, want [acc-600002]", got)
	}
}

func TestSealedProductMap(t *testing.T) {
	b := loadSealedFixture(t)

	productMap := b.BuildSealedProductMap("tcgplayerProductId")
	if got := productMap[600001]; len(got) != 1 || got[0] != "ogn-600001" {
		t.Errorf("productMap[600001] = %v, want [ogn-600001]", got)
	}
	if got := productMap[600002]; len(got) != 1 || got[0] != "acc-600002" {
		t.Errorf("productMap[600002] = %v, want [acc-600002]", got)
	}
	// The single's product id belongs to the card namespace, not this one
	if got, found := productMap[100001]; found {
		t.Errorf("productMap[100001] = %v, want absent", got)
	}
}

func TestSealedSearch(t *testing.T) {
	b := loadSealedFixture(t)

	uuids, err := b.SearchSealedEquals("Origins Booster Box")
	if err != nil {
		t.Fatal(err)
	}
	if len(uuids) != 1 || uuids[0] != "ogn-600001" {
		t.Errorf("SearchSealedEquals = %v, want [ogn-600001]", uuids)
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
	// from TCGplayer, so nothing prevents the collision. The card must
	// keep matching and the sealed product must stay searchable.
	fixture := strings.Replace(sealedFixture,
		`"name": "Origins Booster Box"`, `"name": "Fixture Card"`, 1)
	b, err := Load(strings.NewReader(fixture))
	if err != nil {
		t.Fatal(err)
	}

	uuid, err := b.Match(&mtgmatcher.InputCard{Name: "Fixture Card", Edition: "Origins"})
	if err != nil {
		t.Fatalf("Match = %v, want the card", err)
	}
	// The uuid the fixture publishes for the card's one printing.
	if uuid != "ogn-001_nonfoil" {
		t.Errorf("Match = %q, want %q", uuid, "ogn-001_nonfoil")
	}

	uuids, err := b.SearchSealedEquals("Fixture Card")
	if err != nil {
		t.Fatalf("SearchSealedEquals = %v, want the sealed product", err)
	}
	if !slices.Contains(uuids, "ogn-600001") {
		t.Errorf("SearchSealedEquals = %v, want it to contain ogn-600001", uuids)
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

	co, err := b.GetUUID("acc-600002")
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
