package cardkingdom

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// loadFixture is rows of Card Kingdom's pricelist of 2026-10-02 as
// published, but for the last two: Air Elemental is given no NM price, and
// Blood Lust no price at all.
const loadFixture = `{"meta": {"created_at": "2026-10-02 05:05:11", "base_url": "https://www.cardkingdom.com/"}, "data": [
{"id": 65155, "sku": "PTK-014", "scryfall_id": "f484d47a-fb1d-4746-8f1d-dd9d24e67c1a", "url": "mtg/portal-3k/pang-tong-young-phoenix", "name": "Pang Tong, \"Young Phoenix\"", "variation": "", "edition": "Portal 3K", "is_foil": "false", "price_retail": "69.99", "qty_retail": 4, "price_buy": "32.00", "qty_buying": 2, "condition_values": {"nm_price": "69.99", "nm_qty": 0, "ex_price": "59.49", "ex_qty": 1, "vg_price": "52.49", "vg_qty": 3, "g_price": "45.49", "g_qty": 0}},
{"id": 13048, "sku": "2ED-055", "scryfall_id": "7c666b4b-c4ff-40ca-9d16-c76aafebaa83", "url": "mtg/unlimited/counterspell", "name": "Counterspell", "variation": "", "edition": "Unlimited", "is_foil": "false", "price_retail": "84.99", "qty_retail": 0, "price_buy": "42.50", "qty_buying": 6, "condition_values": {"nm_price": "84.99", "nm_qty": 0, "ex_price": "67.99", "ex_qty": 0, "vg_price": "50.99", "vg_qty": 0, "g_price": "34.00", "g_qty": 0}},
{"id": 21218, "sku": "LEG-292", "scryfall_id": "c062cbae-ce5e-43be-9932-c81a0a3622e8", "url": "mtg/legends/relic-barrier", "name": "Relic Barrier", "variation": "", "edition": "Legends", "is_foil": "false", "price_retail": "12.99", "qty_retail": 20, "price_buy": "5.25", "qty_buying": 0, "condition_values": {"nm_price": "12.99", "nm_qty": 12, "ex_price": "10.39", "ex_qty": 2, "vg_price": "9.09", "vg_qty": 6, "g_price": "6.50", "g_qty": 0}},
{"id": 39000, "sku": "FNEM-001", "scryfall_id": "871ad2f3-1dd2-45ea-881d-529aad3b76ec", "url": "mtg/nemesis/angelic-favor-foil", "name": "Angelic Favor", "variation": "", "edition": "Nemesis", "is_foil": "true", "price_retail": "1.49", "qty_retail": 4, "price_buy": "0.45", "qty_buying": 10, "condition_values": {"nm_price": "1.49", "nm_qty": 0, "ex_price": "1.19", "ex_qty": 4, "vg_price": "0.89", "vg_qty": 0, "g_price": "0.60", "g_qty": 0}},
{"id": 10001, "sku": "4ED-059", "scryfall_id": "e5cfaefb-764c-4c56-bdb3-5f0375168597", "url": "mtg/4th-edition/air-elemental", "name": "Air Elemental", "variation": "", "edition": "4th Edition", "is_foil": "false", "price_retail": "0.35", "qty_retail": 23, "price_buy": "0.02", "qty_buying": 14, "condition_values": {"nm_price": "0.00", "nm_qty": 2, "ex_price": "0.28", "ex_qty": 3, "vg_price": "0.25", "vg_qty": 13, "g_price": "0.18", "g_qty": 5}},
{"id": 10033, "sku": "4ED-178", "scryfall_id": "426e477f-2873-4115-9527-a50a97769dd1", "url": "mtg/4th-edition/blood-lust", "name": "Blood Lust", "variation": "", "edition": "4th Edition", "is_foil": "false", "price_retail": "0.00", "qty_retail": 35, "price_buy": "0.01", "qty_buying": 10, "condition_values": {"nm_price": "0.00", "nm_qty": 3, "ex_price": "0.28", "ex_qty": 20, "vg_price": "0.25", "vg_qty": 11, "g_price": "0.18", "g_qty": 1}}
]}`

// TestLoadPricesEachGrade pins what a pricelist row becomes: a retail entry
// per grade in stock, and an offer per grade, the buy price scaled by that
// grade's retail price the way Card Kingdom scales its own.
func TestLoadPricesEachGrade(t *testing.T) {
	b := realDatastore(t)
	path := filepath.Join(t.TempDir(), "pricelist.json")
	err := os.WriteFile(path, []byte(loadFixture), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	ck := NewScraperLocal(b, path)
	ck.partner = "ban"
	ck.preserveOOS = true
	err = ck.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	got := map[string][]string{}
	keyOf := func(uuid string) string {
		co, _ := b.GetUUID(uuid)
		key := co.Name + " " + co.SetCode
		if co.Foil {
			key += " foil"
		}
		return key
	}
	for uuid, entries := range ck.inventory {
		for _, entry := range entries {
			got[keyOf(uuid)] = append(got[keyOf(uuid)], fmt.Sprintf("sells %s %.2f x%d", entry.Conditions, entry.Price, entry.Quantity))
		}
	}
	for uuid, entries := range ck.buylist {
		for _, entry := range entries {
			got[keyOf(uuid)] = append(got[keyOf(uuid)], fmt.Sprintf("%s buys %s %.2f", entry.VendorName, entry.Conditions, entry.BuyPrice))
		}
	}

	want := map[string][]string{
		`Pang Tong, "Young Phoenix" PTK`: {
			"sells SP 59.49 x1", "sells MP 52.49 x3",
			"Card Kingdom buys NM 32.00", "Card Kingdom buys SP 27.20",
			"Card Kingdom buys MP 24.00", "Card Kingdom buys HP 20.80",
		},
		// Out of stock: only the link is kept
		"Counterspell 2ED": {
			"sells NM 0.00 x1",
			"Card Kingdom buys NM 42.50", "Card Kingdom buys SP 34.00",
			"Card Kingdom buys MP 25.50", "Card Kingdom buys HP 17.00",
		},
		// Not buying: the price stands as the last one known
		"Relic Barrier LEG": {
			"sells NM 12.99 x12", "sells SP 10.39 x2", "sells MP 9.09 x6",
			"Card Kingdom (last known) buys NM 5.25", "Card Kingdom (last known) buys SP 4.20",
			"Card Kingdom (last known) buys MP 3.67", "Card Kingdom (last known) buys HP 2.63",
		},
		"Angelic Favor NEM foil": {
			"sells SP 1.19 x4",
			"Card Kingdom buys NM 0.45", "Card Kingdom buys SP 0.36",
			"Card Kingdom buys MP 0.27", "Card Kingdom buys HP 0.18",
		},
		// The NM price falls back to the retail price
		"Air Elemental 4ED": {
			"sells NM 0.35 x2", "sells SP 0.28 x3", "sells MP 0.25 x13", "sells HP 0.18 x5",
			"Card Kingdom buys NM 0.02", "Card Kingdom buys SP 0.02",
			"Card Kingdom buys MP 0.01", "Card Kingdom buys HP 0.01",
		},
	}
	for key, lines := range want {
		slices.Sort(lines)
		slices.Sort(got[key])
		if !slices.Equal(got[key], lines) {
			t.Errorf("%s:\ngot  %q\nwant %q", key, got[key], lines)
		}
	}
	for key := range got {
		_, found := want[key]
		if !found {
			t.Errorf("%s priced, want nothing: %q", key, got[key])
		}
	}

	// The fields a link and the buylist import read, on one card
	for uuid, entries := range ck.inventory {
		if keyOf(uuid) != `Pang Tong, "Young Phoenix" PTK` {
			continue
		}
		if entries[0].URL != "https://www.cardkingdom.com/mtg/portal-3k/pang-tong-young-phoenix?partner=ban&utm_campaign=ban&utm_medium=affiliate&utm_source=ban" {
			t.Errorf("retail link %s", entries[0].URL)
		}
		if entries[0].CustomFields["RetailPrice"] != "69.99" {
			t.Errorf("retail fields %v, want the NM price beside a lower grade", entries[0].CustomFields)
		}
		offer := ck.buylist[uuid][0]
		if offer.CustomFields["CKTitle"] != `Pang Tong, "Young Phoenix"` || offer.CustomFields["CKSKU"] != "PTK-014" {
			t.Errorf("NM offer fields %v", offer.CustomFields)
		}
		if fmt.Sprintf("%.2f", offer.PriceRatio) != "45.72" {
			t.Errorf("price ratio %.2f, want 45.72", offer.PriceRatio)
		}
		link, err := url.Parse(offer.URL)
		if err != nil {
			t.Fatal(err)
		}
		query := link.Query()
		if query.Get("filter[name]") != "Pang Tong, Young Phoenix" || query.Get("filter[edition]") != "portal-3k" || query.Get("partner") != "ban" {
			t.Errorf("buylist link %s", offer.URL)
		}
	}
}
