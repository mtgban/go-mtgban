package hareruya

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestTitleTreatments pins the treatments the storefront announces in the
// group the plain finish otherwise occupies. The treated printing shares its
// set tag and collector number with the plain one, so a marker read as a
// bare "Foil" prices the treatment as the plain card - and these are the
// printings the shop pays the most for. Each pair below is the same card
// bought twice, once with the marker and once without.
func TestTitleTreatments(t *testing.T) {
	if len(mtgmatcher.GetAllSets()) == 0 {
		t.Skip("ALLPRINTINGS5_PATH not set; skipping the treatment suite")
	}

	for _, tt := range []struct {
		desc, title, wantSet, wantNumber string
	}{
		{
			desc:    "the surge foil is its own printing of the set it reprints",
			title:   "【EN】【サージ・Foil】(166)《久遠なる栄光の笏/Sceptre of Eternal Glory》[40K-SF] 茶R",
			wantSet: "40K", wantNumber: "166★",
		},
		{
			desc:    "and the same card bought plain stays the plain printing",
			title:   "【EN】【Foil】(166)《久遠なる栄光の笏/Sceptre of Eternal Glory》[40K] 茶R",
			wantSet: "40K", wantNumber: "166",
		},
		{
			desc:    "the step-and-compleat foil carries the marked number",
			title:   "【EN】【S&C・Foil】(681)■ボーダーレス■《影生まれの使徒/Shadowborn Apostle》[SLD] 黒",
			wantSet: "SLD", wantNumber: "681Φ",
		},
		{
			desc:    "and the plain Secret Lair printing keeps the bare number",
			title:   "【EN】【Foil】(681)■ボーダーレス■《影生まれの使徒/Shadowborn Apostle》[SLD] 黒",
			wantSet: "SLD", wantNumber: "681",
		},
		{
			desc:    "the etched foil the table already named still resolves",
			title:   "【EN】【エッチング・Foil】(1072)■旧枠■《オパールのモックス/Mox Opal》[SLD] 茶R",
			wantSet: "SLD", wantNumber: "1072",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in, err := preprocess(tt.title)
			if err != nil {
				t.Fatalf("preprocess(%q) = %v", tt.title, err)
			}
			id, err := mtgmatcher.Match(in)
			if err != nil {
				t.Fatalf("Match(%q) = %v", in, err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNumber {
				t.Errorf("Match(%q) = %s|%s, want %s|%s", tt.title, co.SetCode, co.Number, tt.wantSet, tt.wantNumber)
			}
		})
	}
}
