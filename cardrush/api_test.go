package cardrush

import (
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
