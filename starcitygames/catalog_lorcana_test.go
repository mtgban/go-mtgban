package starcitygames

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestLorcanaNumber pins the collector number a Lorcana sku is read as. The
// sku is the more specific of the two numbers a product carries - it keeps the
// printing marker the number field drops and spells a promo's number under the
// series that issued it - but the product's own number vetoes it when the two
// name different cards, which is what a stray digit and a marker the datastore
// numbers separately both do.
func TestLorcanaNumber(t *testing.T) {
	for _, tt := range []struct {
		name, sku, number, want string
	}{
		{"a plain number is what both agree on", "SGL-LOR-001-036-ENC", "036", "036"},
		{"the printing marker the number field drops rides along",
			"SGL-LOR-002-117M-ENN", "117", "117M"},
		{"a marker both sides write stays", "SGL-LOR-002-073b-ENC", "073b", "073b"},
		{"the datastore's own variant letter stays", "SGL-LOR-003-004a-ENN", "004a", "004a"},
		{"a promo series is a heading, not part of the number",
			"SGL-LOR-PRM-P3_031-ENC", "P3_031", "031"},
		{"a promo series the number field already dropped",
			"SGL-LOR-PRM-P01_005-ENK", "005", "005"},
		{"a stray digit in the sku loses to the number field",
			"SGL-LOR-001-0142-ENN", "042", "042"},
		{"a marker the datastore numbers as a card of its own loses too",
			"SGL-LOR-PRM-P02_032B-ENK", "033", "033"},
		{"a segment naming no number at all leaves the number field to answer",
			"SGL-LOR-008-T03-ENN", "000", "000"},
		{"a sku short of a number segment has nothing to read",
			"SGL-LOR-008", "012", "012"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := CatalogProduct{SKU: tt.sku, CollectorNumber: tt.number}
			if got := lorcanaNumber(p); got != tt.want {
				t.Errorf("lorcanaNumber(%q, %q) = %q, want %q", tt.sku, tt.number, got, tt.want)
			}
		})
	}
}

// TestLorcanaFinish pins the one catalog treatment that has to be named for
// the datastore. Every other alt foil is the only foil its printing is sold
// in, so the foil flag reaches it and the marketing name says nothing the
// datastore would recognize.
func TestLorcanaFinish(t *testing.T) {
	for _, tt := range []struct{ finish, want string }{
		{"Rainbow Foil", "RainbowPillars"},
		{"Foil", ""},
		{"Non-foil", ""},
		{"Inkwash Foil", ""},
		{"Epic Foil", ""},
		{"Whisper Foil", ""},
		{"", ""},
	} {
		t.Run(tt.finish, func(t *testing.T) {
			if got := lorcanaFinish(tt.finish); got != tt.want {
				t.Errorf("lorcanaFinish(%q) = %q, want %q", tt.finish, got, tt.want)
			}
		})
	}
}

// TestSoleSibling pins the rule that decides which printing a sku marker
// names. One printing is spelled once per treatment it is sold in, so the
// candidates are counted by name; two names would leave the marker naming
// neither in particular, and the product is refused rather than guessed at.
func TestSoleSibling(t *testing.T) {
	card := func(name, uuid string) *mtgmatcher.CardObject {
		return &mtgmatcher.CardObject{Card: mtgmatcher.Card{Name: name, UUID: uuid}}
	}
	// The answer is pinned by name rather than by uuid: where one printing is
	// spelled twice, either spelling of it is the same right answer.
	for _, tt := range []struct {
		name       string
		candidates []*mtgmatcher.CardObject
		want       string
	}{
		{"no printing beside the base one", nil, ""},
		{"one printing answers", []*mtgmatcher.CardObject{card("Elsa - Gloves Off (Errata Version)", "a")}, "Elsa - Gloves Off (Errata Version)"},
		{
			"one printing sold in two treatments is still one printing",
			[]*mtgmatcher.CardObject{
				card("Elsa - Gloves Off (Errata Version)", "a"),
				card("Elsa - Gloves Off (Errata Version)", "b"),
			},
			"Elsa - Gloves Off (Errata Version)",
		},
		{
			"two printings name neither in particular",
			[]*mtgmatcher.CardObject{
				card("Hades - King of Olympus (Oversized)", "a"),
				card("Hades - King of Olympus (Errata Version)", "b"),
			},
			"",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := soleSibling(tt.candidates)
			if tt.want == "" {
				if got != nil {
					t.Fatalf("soleSibling = %v, want nothing", got)
				}
				return
			}
			if got == nil || got.Name != tt.want {
				t.Fatalf("soleSibling = %v, want %q", got, tt.want)
			}
		})
	}
}

// TestNamesAnotherFormat pins which longer names a sku marker may be routed
// onto. The datastore files a jumbo print of a card, and a lot of cards sold
// as one, at the base card's own set and number, so neither the set nor the
// number tells them from the errata printing a marker actually asks for - only
// what the name adds does. Star City Games sells a jumbo as a listing of its
// own and never as a marked variant, so adopting one prices the wrong object.
func TestNamesAnotherFormat(t *testing.T) {
	for _, tt := range []struct {
		extension string
		want      bool
	}{
		{" (Oversized)", true},
		{" (Set of 9)", true},
		{" (Set of 18)", true},
		{" (Errata Version)", false},
		{" (JP Exclusive)", false},
		{" (Serial Numbered)", false},
		{" (Version 2 of 2)", false},
		{"", false},
	} {
		t.Run(tt.extension, func(t *testing.T) {
			if got := namesAnotherFormat(tt.extension); got != tt.want {
				t.Errorf("namesAnotherFormat(%q) = %v, want %v", tt.extension, got, tt.want)
			}
		})
	}
}
