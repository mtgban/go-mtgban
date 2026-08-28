package hareruya

import (
	"log"
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// TestMain loads the datastore when one is configured; the rest of this
// package's tests read no cards, so a checkout without it still runs them.
func TestMain(m *testing.M) {
	if path := os.Getenv("ALLPRINTINGS5_PATH"); path != "" {
		if err := datastore.Load(path); err != nil {
			log.Fatalln(err)
		}
	}
	os.Exit(m.Run())
}

// TestTitleMarkers pins the three markers that say a listing is not the
// printing its set tag and number name. Each is announced in a different
// place, and each shares its number with the printing it reprints, so
// nothing else in the title tells them apart.
func TestTitleMarkers(t *testing.T) {
	if len(mtgmatcher.GetAllSets()) == 0 {
		t.Skip("ALLPRINTINGS5_PATH not set; skipping the marker suite")
	}

	for _, tt := range []struct {
		desc, title, wantSet, wantNumber string
	}{
		{
			desc:    "the alternate printing is announced where the finish goes",
			title:   "【アルターネイト版】《兜/Helm of Chatzuk》[4ED] 白R",
			wantSet: "4EDALT", wantNumber: "324alt",
		},
		{
			desc:    "and the printing it is an alternate of keeps its own set",
			title:   "《兜/Helm of Chatzuk》[4ED] 白R",
			wantSet: "4ED", wantNumber: "324",
		},
		{
			desc:    "the ampersand card is the promo set's own number",
			title:   "【Foil】(086)■アンパサンド■《ユアンティの魔術師/Yuan-Ti Malison》[AFR] 黒U",
			wantSet: "PAFR", wantNumber: "86a",
		},
		{
			desc:    "an ampersand listing that gives no number is left where it was",
			title:   "【Foil】■アンパサンド■《ユアンティの魔術師/Yuan-Ti Malison》[AFR] 黒U",
			wantSet: "AFR", wantNumber: "86",
		},
		{
			desc:    "the timeshifted reprints are a set of their own",
			title:   "【Foil】■旧枠■《エオスのレインジャー隊長/Ranger-Captain of Eos》(005)[MH1-RT] 白R",
			wantSet: "H1R", wantNumber: "5",
		},
		{
			desc:    "while the set it is appended to is untouched",
			title:   "【Foil】《エオスのレインジャー隊長/Ranger-Captain of Eos》[MH1] 白R",
			wantSet: "MH1", wantNumber: "21",
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
				t.Errorf("%q = %s|%s, want %s|%s", tt.title, co.SetCode, co.Number, tt.wantSet, tt.wantNumber)
			}
		})
	}
}
