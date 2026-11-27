package cardrush

import (
	"bytes"
	"os"
	"testing"
)

func TestParseBuylistPage(t *testing.T) {
	body, err := os.ReadFile("testdata/buying_prices.html")
	if err != nil {
		t.Fatal(err)
	}
	rows, lastPage, err := parseBuylistPage(body)
	if err != nil {
		t.Fatal(err)
	}
	if lastPage != 51 || len(rows) != 2 {
		t.Fatalf("got %d rows over %d pages, want 2 over 51", len(rows), lastPage)
	}
	row := rows[0]
	if row.Name != "龍魂、サルカン/Sarkhan, Dragonsoul" || row.PackCode != "M19" ||
		row.Amount != 300 || row.Language != "日本語" || row.Condition != "" {
		t.Errorf("first row read as %+v", row)
	}
	if rows[1].Condition != "〔EX〕" {
		t.Errorf("second row's condition read as %q", rows[1].Condition)
	}

	_, _, err = parseBuylistPage([]byte("<html></html>"))
	if err == nil {
		t.Error("a page without __NEXT_DATA__ parsed")
	}
}

func TestParseStorePage(t *testing.T) {
	body, err := os.ReadFile("testdata/product_list.html")
	if err != nil {
		t.Fatal(err)
	}
	products, lastPage, err := parseStorePage(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if lastPage != 327 || len(products) != 3 {
		t.Fatalf("got %d listings over %d pages, want 3 over 327", len(products), lastPage)
	}
	want := Product{ID: "323639", Title: "☆SALE☆[EX+]霊気拠点/Aether Hub《英語》【KLD】", Price: 50, Stock: 10}
	if products[0] != want {
		t.Errorf("first listing read as %+v, want %+v", products[0], want)
	}
	if products[1].Price != 448000 || products[2].Stock != 2 {
		t.Errorf("read %+v and %+v", products[1], products[2])
	}

	moved := bytes.ReplaceAll(body, []byte(`class="figure"`), []byte(`class="amount"`))
	_, _, err = parseStorePage(bytes.NewReader(moved))
	if err == nil {
		t.Error("a page with its prices moved parsed")
	}
}

func TestEnglishName(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"稲妻/Lightning Bolt", "Lightning Bolt"},
		{"摩耗+損耗/Wear+Tear", "Wear // Tear"},
		{"+2メイス/+2 Mace", "+2 Mace"},
		{"クリスタルタワーを目指して/To the Crystal Tower　//　謎めいた命令/Cryptic Command", "Cryptic Command"},
		{"魔女の予見　//　否定の力/Force of Negation", "Force of Negation"},
		{"電結の荒廃者/Arcbound Ravager\u200e", "Arcbound Ravager"},
		{"トカシアの歓待/Tocasia’s Welcome", "Tocasia’s Welcome"},
		{"Word of Command", "Word of Command"},
		{"熊トークン", ""},
	} {
		got := englishName(tc.in)
		if got != tc.want {
			t.Errorf("englishName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
