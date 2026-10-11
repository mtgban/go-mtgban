package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// promoLink is one product a promo link table names, as Cardmarket
// catalogs it, and the printing it must land on.
type promoLink struct {
	mkmID                   int
	name, number, expansion string
	setCode, cardNumber     string
	rarity                  string
}

// checkPromoLinks resolves every product of a link table and checks it
// lands on its card, set, number and, where given, rarity; a table entry
// with no row here fails too.
func checkPromoLinks(t *testing.T, b *mtgmatcher.Backend, table map[int]int, links []promoLink) {
	t.Helper()
	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}
	mkm.tcgBridge = map[int]int{}

	seen := map[int]bool{}
	for _, link := range links {
		seen[link.mkmID] = true
		_, found := table[link.mkmID]
		if !found {
			t.Errorf("product %d is not in the table", link.mkmID)
			continue
		}
		product := cm.Product{
			IDProduct:     link.mkmID,
			Name:          link.name,
			Number:        link.number,
			ExpansionName: link.expansion,
		}
		cardID, _, _, err := mkm.resolveProduct(&product)
		if err != nil {
			t.Errorf("resolveProduct(%d %q) = %v", link.mkmID, link.name, err)
			continue
		}
		co, err := b.GetUUID(cardID)
		if err != nil {
			t.Errorf("resolveProduct(%d %q) = %q, %v", link.mkmID, link.name, cardID, err)
			continue
		}
		name := versionTail.ReplaceAllString(link.name, "")
		if co.Name != name || co.SetCode != link.setCode || co.Number != link.cardNumber ||
			(link.rarity != "" && b.RarityLabel(co.Rarity) != link.rarity) {
			t.Errorf("product %d %q landed on %s %s %s %s, want %s %s %s",
				link.mkmID, link.name, co.Name, co.SetCode, co.Number, b.RarityLabel(co.Rarity),
				link.setCode, link.cardNumber, link.rarity)
		}
	}
	for mkmID := range table {
		if !seen[mkmID] {
			t.Errorf("table entry %d has no expected landing", mkmID)
		}
	}
}

// TestPokemonStampLinks pins every pokemonStampLinks entry to the stamped
// printing its Cardmarket product sells.
func TestPokemonStampLinks(t *testing.T) {
	b := loadPokemonBackend(t)
	checkPromoLinks(t, b, pokemonStampLinks, []promoLink{
		{280585, "Beldum", "29", "EX Hidden Legends", "MCAP", "029", ""},
		{295325, "Shiinotic", "17", "Sun & Moon", "PR-1840", "017", ""},
		{295411, "Kangaskhan", "99", "Sun & Moon", "PR-1840", "099", ""},
		{295426, "Oranguru", "113", "Sun & Moon", "PR-1840", "113", ""},
		{319255, "Alolan Vulpix", "21", "Guardians Rising", "MCAP", "021", ""},
		{371657, "Counter Catcher", "91", "Crimson Invasion", "PR-1539", "091", ""},
		{567076, "Ghetsis", "101", "Plasma Freeze", "PR-1840", "101", ""},
		{722693, "Ponyta", "14", "Generations", "MCAP", "014", ""},
		{722697, "Geodude", "43", "Generations", "GEN", "043", ""},
		{722699, "Meowth", "53", "Generations", "MCAP", "053", ""},
		{749041, "Forest Seal Stone", "156", "Silver Tempest", "PR-1840", "156", ""},
		{749909, "Iron Moth", "028", "Paradox Rift", "MCAP", "028", ""},
		{762531, "Koraidon", "119", "Temporal Forces", "MCAP", "119", ""},
		{800144, "Pikachu", "051", "Temporal Forces", "MCAP", "051", ""},
		{800837, "Skeledirge", "038", "Scarlet & Violet", "MCAP", "038", ""},
		{804338, "Miraidon", "121", "Temporal Forces", "MCAP", "121", ""},
		{819347, "Hop's Dubwool", "136", "Journey Together: Additionals", "MCAP", "136", ""},
		{833946, "Team Rocket's Mimikyu", "087", "Destined Rivals: Additionals", "MCAP", "087", ""},
		{841258, "Zekrom", "066", "Paradox Rift", "MCAP", "066", ""},
		{841273, "Latias ex", "076", "Surging Sparks", "MCAP", "076", ""},
		{858716, "Tyranitar", "135", "Paldea Evolved", "MCAP", "135", ""},
		{858724, "Sinistcha", "022", "Twilight Masquerade", "BLE", "022", ""},
		{858734, "Sylveon", "040", "Prismatic Evolutions: Additionals", "MCAP", "040", ""},
		{862171, "Team Rocket's Zapdos", "070", "Destined Rivals: Additionals", "MCAP", "070", ""},
		{882856, "Bayleef", "35", "EX Unseen Forces", "PR-1423", "35", ""},
		{901195, "Ho-Oh", "27", "EX Unseen Forces", "PR-1840", "027", ""},
	})
}

// TestYugiohPromoLinks pins every yugiohPromoLinks entry to the Reshef of
// Destruction promo its Cardmarket product sells. The id skips the number
// and other-card checks, so the rarity is held too.
func TestYugiohPromoLinks(t *testing.T) {
	b := loadYugiohBackend(t)
	checkPromoLinks(t, b, yugiohPromoLinks, []promoLink{
		{244606, "Dark Magician Knight (V.1 - Secret Rare)", "EN1", "Reshef of Destruction Promos", "VDP", "ROD-EN001", "Secret Rare"},
		{244607, "Knight's Title (V.1 - Secret Rare)", "EN1", "Reshef of Destruction Promos", "VDP", "ROD-EN002", "Secret Rare"},
		{363275, "Knight's Title (V.2 - Super Rare)", "001", "Reshef of Destruction Promos", "VDP", "ROD-EN002", "Super Rare"},
		{363276, "Dark Magician Knight (V.2 - Super Rare)", "001", "Reshef of Destruction Promos", "VDP", "ROD-EN001", "Super Rare"},
	})
}
