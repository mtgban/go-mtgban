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

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

func TestSlabCondition(t *testing.T) {
	type slab = mtgban.Slab
	tests := []struct {
		features  []string
		condition string
		old       bool
		foil      bool
		want      mtgban.Condition
		slab      slab
		unread    string
	}{
		{[]string{"Graded", "PSA", "PSA 10"}, "MINT", false, false, mtgban.NM, slab{Grader: "PSA", Score: 10}, ""},
		{[]string{"Graded", "PSA", "PSA 8"}, "NM", false, false, mtgban.NM, slab{Grader: "PSA", Score: 8}, ""},
		{[]string{"Graded", "BGS", "BGS 9.5"}, "MINT", false, false, mtgban.NM, slab{Grader: "BGS", Score: 9.5}, ""},
		{[]string{"Graded", "BGS", "BGS 9"}, "NM", false, false, mtgban.NM, slab{Grader: "BGS", Score: 9}, ""},
		{[]string{"Graded", "BGS", "BGS 8.5"}, "NM", false, false, mtgban.SP, slab{Grader: "BGS", Score: 8.5}, ""},
		{[]string{"Graded", "BGS", "BGS 8"}, "NM", false, false, mtgban.SP, slab{Grader: "BGS", Score: 8}, ""},
		{[]string{"Graded", "CGC", "CGC 10"}, "MINT", false, false, mtgban.NM, slab{Grader: "CGC", Score: 10}, ""},
		{[]string{"Graded", "CGC", "CGC 9.5"}, "MINT", false, false, mtgban.NM, slab{Grader: "CGC", Score: 9.5}, ""},
		{[]string{"Box Topper", "Graded", "CGC", "CGC 8"}, "NM", false, false, mtgban.NM, slab{Grader: "CGC", Score: 8}, ""},
		// The bucket wins over ABU's own condition.
		{[]string{"Graded", "BGS", "BGS 9"}, "SP", false, false, mtgban.NM, slab{Grader: "BGS", Score: 9}, ""},
		// Quad 10 is a Black Label, however the features are ordered.
		{[]string{"Graded", "BGS", "BGS Quad 10"}, "MINT", false, false, mtgban.NM, slab{Grader: "BGS", Score: 10, Label: "Black Label"}, ""},
		{[]string{"Graded", "BGS", "BGS Quad 10", "BGS 10"}, "NM", false, false, mtgban.NM, slab{Grader: "BGS", Score: 10, Label: "Black Label"}, ""},
		// Less than 8: ABU's condition, held under the grader's 7.
		{[]string{"Graded", "BGS", "BGS Less than 8"}, "NM", false, false, mtgban.SP, slab{Grader: "BGS", Label: "Less than 8"}, ""},
		{[]string{"Graded", "BGS", "BGS Less than 8"}, "HP", false, false, mtgban.MP, slab{Grader: "BGS", Label: "Less than 8"}, ""},
		{[]string{"Graded", "CGC", "CGC Less than 8"}, "PLD", false, false, mtgban.SP, slab{Grader: "CGC", Label: "Less than 8"}, ""},
		{[]string{"Graded", "PSA", "PSA Less than 8"}, "NM", false, false, mtgban.NM, slab{Grader: "PSA", Label: "Less than 8"}, ""},
		{[]string{"Graded", "PSA", "PSA Less than 8"}, "PLD", false, false, mtgban.SP, slab{Grader: "PSA", Label: "Less than 8"}, ""},
		// No bucket: ABU's condition alone, under the grader if it names one.
		{[]string{"Graded"}, "MINT", false, false, mtgban.NM, slab{}, ""},
		{[]string{"Graded"}, "PLD", false, false, mtgban.SP, slab{}, ""},
		{[]string{"Graded"}, "HP", false, false, mtgban.MP, slab{}, ""},
		{[]string{"Graded", "PSA"}, "NM", false, false, mtgban.NM, slab{Grader: "PSA"}, ""},
		// What does not grade is NM, with its wording.
		{[]string{"Graded"}, "", false, false, mtgban.NM, slab{Label: "Graded"}, "Graded"},
		{[]string{"Graded", "CGC"}, "SP", false, false, mtgban.NM, slab{Grader: "CGC", Label: "CGC SP"}, "CGC SP"},
		{[]string{"Graded", "CGC", "CGC Less than 8"}, "SP", false, false, mtgban.NM, slab{Grader: "CGC", Label: "CGC Less than 8 SP"}, "CGC Less than 8 SP"},
		{[]string{"Graded", "BGS", "BGS Quad 9"}, "MINT", false, false, mtgban.NM, slab{Grader: "BGS", Score: 9, Label: "Quad"}, ""},
		// A word beside the score is its label.
		{[]string{"Graded", "CGC", "CGC Pristine 10"}, "MINT", false, false, mtgban.NM, slab{Grader: "CGC", Score: 10, Label: "Pristine"}, ""},
		{[]string{"Graded", "CGC", "CGC Pristine"}, "MINT", false, false, mtgban.NM, slab{Grader: "CGC", Score: 10, Label: "Pristine"}, ""},
		{[]string{"Graded", "CGC", "CGC Authentic Altered"}, "MINT", false, false, mtgban.PO, slab{Grader: "CGC", Label: "Authentic Altered"}, ""},
		{[]string{"Graded", "BGS", "BGS Pristine"}, "MINT", false, false, mtgban.NM, slab{Grader: "BGS", Label: "BGS Pristine"}, "BGS Pristine"},
		{[]string{"Graded", "PSA", "PSA Authentic"}, "MINT", false, false, mtgban.NM, slab{Grader: "PSA", Label: "PSA Authentic"}, "PSA Authentic"},
		// TAG has no grade table; a score is kept, at NM.
		{[]string{"Graded", "TAG", "TAG 9.5"}, "MINT", false, false, mtgban.NM, slab{Grader: "TAG", Score: 9.5}, ""},
		{[]string{"Graded", "TAG", "TAG Pristine 10"}, "MINT", false, false, mtgban.NM, slab{Grader: "TAG", Score: 10, Label: "Pristine"}, ""},
		{[]string{"Graded", "TAG"}, "PLD", false, false, mtgban.NM, slab{Grader: "TAG", Label: "TAG"}, "TAG"},
		{[]string{"Graded", "TAG", "TAG Less than 8"}, "NM", false, false, mtgban.NM, slab{Grader: "TAG", Label: "TAG Less than 8 NM"}, "TAG Less than 8 NM"},
		// A grader outside the supported ones is unread, whatever ABU's condition.
		{[]string{"Graded", "SGC", "SGC 9"}, "PLD", false, false, mtgban.NM, slab{Label: "SGC 9"}, "SGC 9"},
		{[]string{"SGC 9", "Graded", "SGC"}, "NM", false, false, mtgban.NM, slab{Label: "SGC 9"}, "SGC 9"},
		{[]string{"Dual Land", "Graded", "SGC"}, "NM", false, false, mtgban.NM, slab{Label: "SGC"}, "SGC"},
		// A score off the half steps reads as the one below it.
		{[]string{"Graded", "BGS", "BGS 9.25"}, "MINT", false, false, mtgban.NM, slab{Grader: "BGS", Score: 9}, ""},
		{[]string{"Graded", "BGS", "BGS 8.75"}, "MINT", false, false, mtgban.SP, slab{Grader: "BGS", Score: 8.5}, ""},
		{[]string{"Graded", "CGC", "CGC 10.5"}, "MINT", false, false, mtgban.NM, slab{Grader: "CGC", Score: 10}, ""},
		{[]string{"Graded", "CGC", "CGC 95"}, "MINT", false, false, mtgban.NM, slab{Grader: "CGC", Label: "CGC 95"}, "CGC 95"},
		{[]string{"Graded", "CGC", "CGC 0.5"}, "MINT", false, false, mtgban.NM, slab{Grader: "CGC", Label: "CGC 0.5"}, "CGC 0.5"},
		// An old or foil card reads its condition a grade stricter, as a
		// plain copy does; a numeric bucket does not care.
		{[]string{"Graded"}, "PLD", true, false, mtgban.MP, slab{}, ""},
		{[]string{"Graded"}, "HP", true, false, mtgban.HP, slab{}, ""},
		{[]string{"Graded", "BGS", "BGS Less than 8"}, "PLD", true, false, mtgban.MP, slab{Grader: "BGS", Label: "Less than 8"}, ""},
		{[]string{"Graded", "PSA", "PSA Less than 8"}, "NM", true, false, mtgban.NM, slab{Grader: "PSA", Label: "Less than 8"}, ""},
		{[]string{"Graded", "BGS", "BGS 9"}, "NM", true, false, mtgban.NM, slab{Grader: "BGS", Score: 9}, ""},
		{[]string{"Graded", "BGS", "BGS Less than 8"}, "PLD", false, true, mtgban.MP, slab{Grader: "BGS", Label: "Less than 8"}, ""},
		{[]string{"Graded"}, "HP", false, true, mtgban.HP, slab{}, ""},
	}
	for _, test := range tests {
		doc := ABUCard{Features: test.features, Condition: test.condition}
		got, slab, unread := slabCondition(&doc, test.old, test.foil)
		if got != test.want || slab == nil || *slab != test.slab || unread != test.unread {
			t.Errorf("%v %q old=%t foil=%t: got %q %+v unread %q, want %q %+v unread %q", test.features, test.condition, test.old, test.foil, got, slab, unread, test.want, test.slab, test.unread)
		}
	}
}

func TestInfoForScraperGradedMode(t *testing.T) {
	abu := NewScraper(&matcher.Backend{})
	for _, name := range slices.Concat(availableMarketNames, availableTraderNames) {
		info := abu.InfoForScraper(name)
		if info.GradedMode != (name == "ABU Games Graded") {
			t.Errorf("%s: GradedMode %t", name, info.GradedMode)
		}
	}
}

// TestGradedSeller walks live slabs, and a group mixing slabs with plain
// copies, through Load: slabs land on the graded seller only, by bucket,
// each with its Slab, and one that does not grade is published as NM.
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
			line := fmt.Sprintf("%s %s %s foil=%t %s %.2f x%d", entry.SellerName, co.SetCode, co.Number, co.Foil, entry.Conditions, entry.Price, entry.Quantity)
			if entry.Slab != nil {
				line += fmt.Sprintf(" %+v", *entry.Slab)
			}
			got = append(got, line)
			if co.Foil && !strings.Contains(entry.URL, "Foil") {
				t.Errorf("%s links to %s", co, entry.URL)
			}
			graded := strings.Contains(entry.URL, "magic_features=%5B%5B%22Graded%22%5D%5D")
			if graded != (entry.SellerName == "ABU Games Graded") {
				t.Errorf("%s on %s links to %s", co, entry.SellerName, entry.URL)
			}
			if (entry.Slab != nil) != (entry.SellerName == "ABU Games Graded") {
				t.Errorf("%s on %s has slab %+v", co, entry.SellerName, entry.Slab)
			}
		}
	}
	slices.Sort(got)
	want := []string{
		"ABU Games 2ED 262 foil=false NM 9000.00 x1",
		"ABU Games Graded 2ED 262 foil=false NM 20000.00 x1 {Grader:BGS Score:10 Label:Black Label}",
		"ABU Games Graded 2ED 262 foil=false NM 5000.00 x1 {Grader:CGC Score:0 Label:CGC Less than 8 SP}",
		"ABU Games Graded 2ED 262 foil=false NM 7000.00 x1 {Grader:BGS Score:9 Label:}",
		"ABU Games Graded 2ED 262 foil=false SP 6324.99 x1 {Grader:BGS Score:8 Label:}",
		"ABU Games Graded 2ED 262 foil=false SP 6324.99 x1 {Grader:BGS Score:8.5 Label:}",
		"ABU Games Graded PLC 73 foil=true MP 25.45 x1 {Grader:BGS Score:0 Label:Less than 8}",
		"ABU Games Graded PUMA U5 foil=true NM 124.69 x1 {Grader:CGC Score:9.5 Label:}",
		"ABU Games Graded PUMA U5 foil=true NM 129.69 x1 {Grader:CGC Score:9.5 Label:}",
	}
	if !slices.Equal(got, want) {
		t.Errorf("inventory:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if !slices.Contains(logs, `[ABU] unsupported slab "CGC Less than 8 SP", published as NM on 900005`) {
		t.Errorf("unread slab not logged: %q", logs)
	}
}
