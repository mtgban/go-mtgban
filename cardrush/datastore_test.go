package cardrush

import (
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"

	_ "github.com/mtgban/go-mtgban/matcher/magic"
)

var (
	magicOnce sync.Once
	magicErr  error
	magicB    *matcher.Backend
)

// withMagic loads AllPrintings the first time a test asks for it, and skips
// where the run carries none.
func withMagic(t *testing.T) *matcher.Backend {
	t.Helper()
	magicOnce.Do(func() {
		path := os.Getenv("ALLPRINTINGS5_PATH")
		if path == "" {
			return
		}
		magicB, magicErr = datastore.Read("magic", path)
	})
	if magicErr != nil {
		t.Fatal(magicErr)
	}
	if magicB == nil {
		t.Skip("Need ALLPRINTINGS5_PATH set to run this test")
	}
	return magicB
}

// TestPreprocess runs buylist rows through preprocess and Match, as Load
// does, and checks the printing each lands on.
func TestPreprocess(t *testing.T) {
	b := withMagic(t)
	for _, tc := range []struct {
		pack, name, language string
		set, number          string
	}{
		{"FIN", "(FOIL)(シリアル)旅するチョコボ/Traveling Chocobo", "英語", "FIN", "551f"},
		{"MSH", "(コズミックFOIL)マインド・ストーン/The Mind Stone", "英語", "MSH", "385"},
		// Full art is extended art where the set has no borderless copy,
		// and a showcase frame where it has neither.
		{"BLC", "(フルアート)鼠声の剣/Sword of the Squeak", "英語", "BLC", "72"},
		{"BLC", "鼠声の剣/Sword of the Squeak", "英語", "BLC", "40"},
		{"AFR", "(フルアート)目玉の暴君の住処/Hive of the Eye Tyrant", "英語", "AFR", "355"},
		{"MPS", "(FOIL)信義の神オケチラ/Oketra the True", "英語", "MP2", "5"},
		{"MPS", "(全面光沢FOIL)電結の荒廃者/Arcbound Ravager\u200e", "英語", "MPS", "31"},
		{"MPS", "(部分光沢FOIL)電結の荒廃者/Arcbound Ravager\u200e", "英語", "MPS", "31"},
		{"LTC", "(ダブルレインボウFOIL)(Elven)太陽の指輪/Sol Ring", "英語", "LTC", "408z"},
		{"LTC", "(Dwarven)太陽の指輪/Sol Ring", "英語", "LTC", "409"},
		{"4ED", "(黒枠)魔力の櫃/Mana Vault", "日本語", "4BB", "334"},
		{"DGM", "摩耗+損耗/Wear+Tear", "英語", "DGM", "135"},
		{"DSK", "(フルアート)収納室+忘れられた地下室/Walk-In Closet+Forgotten Cellar", "英語", "DSK", "341"},
		{"MH2", "(スケッチアート)濁浪の執政/Murktide Regent", "英語", "MH2", "337"},
		{"BOT", "(シャッタードグラス)独創的な科学者、ジェットファイヤー/Jetfire, Ingenious Scientist", "英語", "BOT", "18"},
		{"PRM", "(Spotlight)鉱炉と前線の剣/Sword of Forge and Frontier", "英語", "PSPL", "3"},
		{"Battle Royale Box Set(TMP)", "生ける屍/Living Death", "英語", "BRB", "38"},
		{"P22", "(FOIL)全知/Omniscience", "英語", "P22", "2"},
		{"WMCQ Promo Cards", "(FOIL)聖トラフトの霊/Geist of Saint Traft", "英語", "WMC", "2014"},
		{"Reprint Cards(The List)", "メムナイト/Memnite(SOM)", "英語", "PLST", "SOM-174"},
		{"Reprint Cards(The List)", "稲妻/Lightning Bolt(MagicFest Cards2019)", "英語", "PLST", "PF19-1"},
		{"PRM", "(FOIL)(フルアート)セラの天使/Serra Angel", "英語", "PW24", "1"},
		{"FCA", "クリスタルタワーを目指して/To the Crystal Tower　//　謎めいた命令/Cryptic Command", "英語", "FCA", "29"},
	} {
		card, err := preprocess(b, tc.name, tc.pack, tc.language)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		id, err := b.Match(card)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		co, err := b.GetUUID(id)
		if err != nil {
			t.Fatal(err)
		}
		if co.SetCode != tc.set || co.Number != tc.number {
			t.Errorf("%s landed on %s %s, want %s %s", tc.name, co.SetCode, co.Number, tc.set, tc.number)
		}
	}
}

// TestFoilPack checks a product the store sells only in foil reads as foil
// though its rows carry no FOIL note.
func TestFoilPack(t *testing.T) {
	b := withMagic(t)
	card, err := preprocess(b, "らせんの円錐/Helix Pinnacle", "Reprint Cards(Mystery Booster FOIL)", "英語")
	if err != nil {
		t.Fatal(err)
	}
	if !card.Foil {
		t.Errorf("%s read as nonfoil", card)
	}
}

// TestPreprocessRefuses checks the rows no printing stands for.
func TestPreprocessRefuses(t *testing.T) {
	b := withMagic(t)
	for _, tc := range []struct {
		pack, name, language string
		unsupported          bool
	}{
		// A Japanese copy of an English printing.
		{"M19", "龍魂、サルカン/Sarkhan, Dragonsoul", "日本語", true},
		// One printing in the catalog, two products at the store.
		{"WAR", "(FOIL)(日限定イラスト)龍神、ニコル・ボーラス/Nicol Bolas, Dragon-God ※通常パック版前期※", "日本語", true},
		{"VOW", "(FOIL)(ショーケース枠)不笑のソリン/Sorin the Mirthless※コレクターブースター産", "英語", true},
		// A note nothing translates.
		{"Arena Promos", "(ラヴニカ)島/Island", "英語", false},
		// Full art on two promos the row does not tell apart.
		{"PRM", "(フルアート)ミッドガルの傭兵、クラウド/Cloud, Midgar Mercenary", "英語", false},
		// Full art in a set whose printings of the card have none.
		{"M10", "(フルアート)島/Island", "英語", false},
	} {
		_, err := preprocess(b, tc.name, tc.pack, tc.language)
		if err == nil || errors.Is(err, matcher.ErrUnsupported) != tc.unsupported {
			t.Errorf("%s: got %v", tc.name, err)
		}
	}
}

// TestAddProduct runs store listings through the path Load takes and checks
// the printing, the grade and the slab each is sold as.
func TestAddProduct(t *testing.T) {
	cr := NewScraper(withMagic(t))
	for _, tc := range []struct {
		title, set, number string
		grade              mtgban.Condition
	}{
		{"☆SALE☆[EX+]霊気拠点/Aether Hub《英語》【KLD】", "KLD", "242", mtgban.SP},
		{"[HPLD]思考囲い/Thoughtseize《英語》【THS】", "THS", "107", mtgban.HP},
		{"[PLD]稲妻/Lightning Bolt《英語》【LEB】#292", "LEB", "162", mtgban.MP},
		{"メムナイト/Memnite(SOM)《英語》【Reprint Cards(The List)】", "PLST", "SOM-174", mtgban.NM},
	} {
		id := tc.title
		cr.addProduct(Product{ID: id, Title: tc.title, Price: 100, Stock: 1}, 1)
		var found bool
		for cardID, entries := range cr.inventory {
			for _, entry := range entries {
				if entry.OriginalID != id {
					continue
				}
				found = true
				co, err := cr.backend.GetUUID(cardID)
				if err != nil {
					t.Fatal(err)
				}
				if co.SetCode != tc.set || co.Number != tc.number || entry.Conditions != tc.grade {
					t.Errorf("%s sold as %s %s %s, want %s %s %s", tc.title, co.SetCode, co.Number, entry.Conditions, tc.set, tc.number, tc.grade)
				}
			}
		}
		if !found {
			t.Errorf("%s not sold", tc.title)
		}
	}

	// Neither a sealed box, a slab nor a listing with no price is sold.
	before := len(cr.inventory)
	for _, product := range []Product{
		{Title: "(4個セット・日本語)FINAL FANTASY 統率者デッキ《-》【未開封BOX】", Price: 100},
		{Title: "〔BGS9.5鑑定済〕(ネオンピンクFOIL)旅するチョコボ/Traveling Chocobo《英語》【FIN】", Price: 100},
		{Title: "死滅都市の執政/Necropolis Regent《英語》【RTR】"},
	} {
		product.ID, product.Stock = product.Title, 1
		cr.addProduct(product, 1)
	}
	if len(cr.inventory) != before {
		t.Error("a sealed box, a slab or an unpriced listing was sold")
	}
}
