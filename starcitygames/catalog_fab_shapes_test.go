package starcitygames

import (
	"errors"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/fleshandblood"
)

func TestFabArtPosition(t *testing.T) {
	for _, tt := range []struct{ name, sku, want string }{
		{"Eye of Ophidia Art Card", "SGL-FAB-ANQ-P01-ENR", "top left"},
		{"Eye of Ophidia Art Card", "SGL-FAB-ANQ-P05-ENR", "middle center"},
		{"Heart of Fyendal Art Card", "SGL-FAB-ANQ-P10-ENR", "top left"},
		{"Heart of Fyendal Art Card", "SGL-FAB-ANQ-P14-ENR", "middle center"},
		{"Riches of Tropal-Dhani Art Card", "SGL-FAB-ANQ-P27-ENR", "bottom right"},
		{"Eye of Ophidia", "SGL-FAB-ANQ-002-ENN", ""},
		{"Eye of Ophidia Art Card", "SGL-FAB-ANQ-002-ENN", ""},
	} {
		if got := fabArtPosition(tt.name, tt.sku); got != tt.want {
			t.Errorf("fabArtPosition(%q, %q) = %q, want %q", tt.name, tt.sku, got, tt.want)
		}
	}
}

func TestFabPrintRunFinishes(t *testing.T) {
	for _, tt := range []struct{ set, finish, wantSet, wantFinish string }{
		{"Promotional Cards", "Gold Foil", "Promotional Cards", "Cold Foil"},
		{"Welcome to Rathe (1st Edition)", "Gold Foil", "Welcome to Rathe", "1st Edition Cold Foil"},
		{"Promotional Cards", "Cold Foil", "Promotional Cards", "Cold Foil"},
	} {
		set, finish := fabPrintRun(tt.set, tt.finish)
		if set != tt.wantSet || finish != tt.wantFinish {
			t.Errorf("fabPrintRun(%q, %q) = (%q, %q), want (%q, %q)", tt.set, tt.finish, set, finish, tt.wantSet, tt.wantFinish)
		}
	}
}

func TestFabTreatments(t *testing.T) {
	got := fabTreatments([]string{"red", "extended art", "blue"})
	if len(got) != 1 || got[0] != "extended art" {
		t.Errorf("fabTreatments = %v, want [extended art]", got)
	}
	if got := fabTreatments([]string{"red"}); len(got) != 0 {
		t.Errorf("fabTreatments(red) = %v, want none", got)
	}
}

// TestCatalogFabShapes pins the listings the catalog spells in ways the
// datastore does not, through the resolve path: a gold foil is the cold foil
// promo, a promo's finish cannot misname the one printing its number holds,
// an art card's number is its piece of the picture, the marker on a
// pitch-labelled base finds the extended art, a misspelt name and a
// misspelt number reach the printing, and an insert is no card.
func TestCatalogFabShapes(t *testing.T) {
	withGameDatastore(t, "FLESHANDBLOOD_PATH", fleshandblood.Load)

	product := func(sku, name, set, finish, group string) CatalogProduct {
		return CatalogProduct{
			SKU: sku, Name: name, Game: "Flesh and Blood", Set: set, ProductType: ProductTypeSingles,
			Finish: finish, FinishGroup: group, Language: "English",
		}
	}
	for _, tt := range []struct {
		desc string
		p    CatalogProduct
		want string
	}{
		{"gold foil", product("SGL-FAB-PRM-FAB_066-ENG", "Talismanic Lens", "Promotional Cards", "Gold Foil", "Alt Foil"), "fab066_275543_cold"},
		{"promo finish misnamed", product("SGL-FAB-PRM-FAB_189-ENC", "Boast", "Promotional Cards", "Cold Foil", "Alt Foil"), "fab189_580618_rainbow"},
		{"promo marvel misnamed", product("SGL-FAB-PRM-TNP_001-ENC", "Wild Ride (Red)", "Promotional Cards", "Cold Foil", "Alt Foil"), "tnp001_692575_rainbow"},
		{"art card piece", product("SGL-FAB-ANQ-P08-ENR", "Eye of Ophidia Art Card", "Antiquity Pack", "Rainbow Foil", "Foil"), "688788_rainbow"},
		{"art card of the next picture", product("SGL-FAB-ANQ-P14-ENR", "Heart of Fyendal Art Card", "Antiquity Pack", "Rainbow Foil", "Foil"), "688794_rainbow"},
		{"pitch-labelled base", product("SGL-FAB-HNT-155-ENR", "Drop of Dragon Blood", "The Hunted", "Rainbow Foil", "Foil"), "hnt155_611179_rainbow"},
		{"marker finds the extended art past the pitch label", product("SGL-FAB-HNT2-155-ENR", "Drop of Dragon Blood", "The Hunted", "Rainbow Foil", "Foil"), "hnt155_614550_rainbow"},
		{"misspelt name", product("SGL-FAB-PEN-190-ENN", "Deep Recess of Existence", "Compendium of Rathe", "Non-foil", "Non-foil"), "pen190_676395"},
		{"misspelt number", product("SGL-FAB-APS-056-ENR", "Attention Grabbers", "Armory Deck: Pleiades", "Rainbow Foil", "Foil"), "aps006_653806_rainbow"},
		{"lettered marvels", product("SGL-FAB-MPG2-112b-ENC", "Seismic Surge", "Mastery Pack Guardian", "Cold Foil", "Alt Foil"), "mpg112_647746_cold"},
	} {
		got, err := resolveProduct(GameFleshAndBlood, tt.p)
		if err != nil {
			t.Errorf("%s: resolveProduct(%s) = %v", tt.desc, tt.p.SKU, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%s: resolveProduct(%s) = %q, want %q", tt.desc, tt.p.SKU, got, tt.want)
		}
	}
	// A set card's finish keeps its say: the datastore sells no rainbow
	// Hyper Driver (Red), and the listing is refused rather than priced
	// as the cold foil.
	if got, err := resolveProduct(GameFleshAndBlood, product("SGL-FAB-DYN-110-ENR", "Hyper Driver (Red)", "Dynasty", "Rainbow Foil", "Foil")); err == nil {
		t.Errorf("resolveProduct(DYN-110 Rainbow Foil) = %q, want a refusal", got)
	}
	_, err := resolveProduct(GameFleshAndBlood, product("SGL-FAB-ROS-ART_000a-ENC", "Binder Label", "Rosetta", "Cold Foil", "Alt Foil"))
	if !errors.Is(err, mtgmatcher.ErrUnsupported) {
		t.Errorf("resolveProduct(Binder Label) = %v, want ErrUnsupported", err)
	}
}
