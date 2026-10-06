package abugames

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
)

func TestSlabCondition(t *testing.T) {
	tests := []struct {
		features  []string
		condition string
		old       bool
		foil      bool
		want      mtgban.Condition
		refused   string
	}{
		{[]string{"Graded", "PSA", "PSA 10"}, "MINT", false, false, mtgban.NM, ""},
		{[]string{"Graded", "PSA", "PSA 8"}, "NM", false, false, mtgban.NM, ""},
		{[]string{"Graded", "BGS", "BGS 9.5"}, "MINT", false, false, mtgban.NM, ""},
		{[]string{"Graded", "BGS", "BGS 9"}, "NM", false, false, mtgban.NM, ""},
		{[]string{"Graded", "BGS", "BGS 8.5"}, "NM", false, false, mtgban.SP, ""},
		{[]string{"Graded", "BGS", "BGS 8"}, "NM", false, false, mtgban.SP, ""},
		{[]string{"Graded", "CGC", "CGC 10"}, "MINT", false, false, mtgban.NM, ""},
		{[]string{"Graded", "CGC", "CGC 9.5"}, "MINT", false, false, mtgban.NM, ""},
		{[]string{"Box Topper", "Graded", "CGC", "CGC 8"}, "NM", false, false, mtgban.NM, ""},
		// The bucket wins over ABU's own condition.
		{[]string{"Graded", "BGS", "BGS 9"}, "SP", false, false, mtgban.NM, ""},
		// Less than 8: ABU's condition, held under the grader's 7.
		{[]string{"Graded", "BGS", "BGS Less than 8"}, "NM", false, false, mtgban.SP, ""},
		{[]string{"Graded", "BGS", "BGS Less than 8"}, "HP", false, false, mtgban.MP, ""},
		{[]string{"Graded", "CGC", "CGC Less than 8"}, "PLD", false, false, mtgban.SP, ""},
		{[]string{"Graded", "PSA", "PSA Less than 8"}, "NM", false, false, mtgban.NM, ""},
		{[]string{"Graded", "PSA", "PSA Less than 8"}, "PLD", false, false, mtgban.SP, ""},
		{[]string{"Graded", "CGC", "CGC Less than 8"}, "SP", false, false, "", "CGC Less than 8"},
		// No grader: ABU's condition alone.
		{[]string{"Graded"}, "MINT", false, false, mtgban.NM, ""},
		{[]string{"Graded"}, "PLD", false, false, mtgban.SP, ""},
		{[]string{"Graded"}, "HP", false, false, mtgban.MP, ""},
		{[]string{"Graded"}, "", false, false, "", ""},
		{[]string{"Graded", "BGS", "BGS Quad 10"}, "MINT", false, false, "", "BGS Quad 10"},
		// An old or foil card reads its condition a grade stricter, as a
		// plain copy does; a numeric bucket does not care.
		{[]string{"Graded"}, "PLD", true, false, mtgban.MP, ""},
		{[]string{"Graded"}, "HP", true, false, mtgban.HP, ""},
		{[]string{"Graded", "BGS", "BGS Less than 8"}, "PLD", true, false, mtgban.MP, ""},
		{[]string{"Graded", "PSA", "PSA Less than 8"}, "NM", true, false, mtgban.NM, ""},
		{[]string{"Graded", "BGS", "BGS 9"}, "NM", true, false, mtgban.NM, ""},
		{[]string{"Graded", "BGS", "BGS Less than 8"}, "PLD", false, true, mtgban.MP, ""},
		{[]string{"Graded"}, "HP", false, true, mtgban.HP, ""},
	}
	for _, test := range tests {
		doc := ABUCard{Features: test.features, Condition: test.condition}
		got, refused := slabCondition(&doc, test.old, test.foil)
		if got != test.want || refused != test.refused {
			t.Errorf("%v %q old=%t foil=%t: got %q refused %q, want %q refused %q", test.features, test.condition, test.old, test.foil, got, refused, test.want, test.refused)
		}
	}
}

// TestGradedSeller walks live slabs, and a group mixing slabs with plain
// copies, through Load: slabs land on the graded seller only, by bucket.
func TestGradedSeller(t *testing.T) {
	b := realDatastore(t)

	page, err := os.ReadFile("testdata/graded_listings.json")
	if err != nil {
		t.Fatal(err)
	}
	var logs []string
	abu := NewScraper(b)
	abu.logCallback = func(format string, a ...any) {
		logs = append(logs, fmt.Sprintf(format, a...))
	}
	abu.client.client = &http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) {
		body := string(page)
		if r.URL.Query().Get("start") != "0" {
			body = `{}`
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	err = abu.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var got []string
	for cardID, entries := range abu.Inventory() {
		co, err := b.GetUUID(cardID)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			got = append(got, fmt.Sprintf("%s %s %s foil=%t %s %.2f x%d", entry.SellerName, co.SetCode, co.Number, co.Foil, entry.Conditions, entry.Price, entry.Quantity))
			if co.Foil && !strings.Contains(entry.URL, "Foil") {
				t.Errorf("%s links to %s", co, entry.URL)
			}
			graded := strings.Contains(entry.URL, "magic_features=%5B%5B%22Graded%22%5D%5D")
			if graded != (entry.SellerName == "ABU Games Graded") {
				t.Errorf("%s on %s links to %s", co, entry.SellerName, entry.URL)
			}
		}
	}
	slices.Sort(got)
	want := []string{
		"ABU Games 2ED 262 foil=false NM 9000.00 x1",
		"ABU Games Graded 2ED 262 foil=false NM 7000.00 x1",
		"ABU Games Graded 2ED 262 foil=false SP 6324.99 x2",
		"ABU Games Graded PLC 73 foil=true MP 25.45 x1",
		"ABU Games Graded PUMA U5 foil=true NM 124.69 x1",
		"ABU Games Graded PUMA U5 foil=true NM 129.69 x1",
	}
	if !slices.Equal(got, want) {
		t.Errorf("inventory:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if !slices.Contains(logs, `[ABU] unsupported "BGS Quad 10" condition on 900004`) {
		t.Errorf("unknown bucket not refused: %q", logs)
	}
}
