package cardkingdom

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"

	"github.com/mtgban/go-mtgban/mtgban"
)

// TestPreprocessGradedNested pins the titles whose edition carries a
// parenthetical of its own. Refusing them dropped 41 of the roughly 950
// listings this storefront grades, silently, and they are the expensive end
// of its inventory.
func TestPreprocessGradedNested(t *testing.T) {
	for _, tt := range []struct {
		desc    string
		title   string
		name    string
		edition string
		variant string
		foil    bool
	}{
		{
			desc:    "the treatment in a parenthetical of its own",
			title:   "Raph & Mikey, Troublemakers (TMNT Foil (Showcase) CGC Pristine 10) #7165",
			name:    "Raph & Mikey, Troublemakers",
			edition: "TMNT", variant: "Showcase", foil: true,
		},
		{
			desc:    "and where the edition carries a word of its own too",
			title:   "Michelangelo, the Heart (TMNT Eternal Foil (Borderless) CGC Pristine 10) #7051",
			name:    "Michelangelo, the Heart",
			edition: "TMNT Eternal", variant: "Borderless", foil: true,
		},
		{
			// The abbreviation reaches the set on its own, but not with
			// "Source Material Cards" behind it: the card falls back to
			// Shadowmoor, where it was first printed.
			desc:    "the edition this storefront abbreviates is spelled out",
			title:   "Plague of Vermin (TMNT Source Material Cards Foil CGC 10) #7067",
			name:    "Plague of Vermin",
			edition: "Teenage Mutant Ninja Turtles Source Material", foil: true,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := preprocessGraded(tt.title)
			if err != nil {
				t.Fatalf("preprocessGraded(%q) = %v", tt.title, err)
			}
			if got.Name != tt.name || got.Edition != tt.edition ||
				got.Variation != tt.variant || got.Foil != tt.foil {
				t.Errorf("preprocessGraded(%q) = %q/%q/%q foil=%v, want %q/%q/%q foil=%v",
					tt.title, got.Name, got.Edition, got.Variation, got.Foil,
					tt.name, tt.edition, tt.variant, tt.foil)
			}
		})
	}
}

// TestPreprocessGradedEditions pins three title shapes that can leave
// the edition either wrong or cluttered with grade-tier noise: this
// storefront's own "Breaking New" typo, the correctly-spelled showcase
// variant of the same shelf, and a rules-status word that belongs on the
// edition, not stripped from it.
func TestPreprocessGradedEditions(t *testing.T) {
	for _, tt := range []struct {
		desc    string
		title   string
		name    string
		edition string
		variant string
		foil    bool
	}{
		{
			desc:    "the storefront's own typo for Breaking News",
			title:   "Commandeer (Outlaws of Thunder Junction Breaking New Foil CGC Pristine 10) #8003",
			name:    "Commandeer",
			edition: "Breaking News", foil: true,
		},
		{
			desc:    "the correctly-spelled Breaking News Showcase",
			title:   "Path to Exile (Outlaws of Thunder Junction Breaking News Showcase Foil CGC Pristine 10) #8017",
			name:    "Path to Exile",
			edition: "Breaking News", variant: "Showcase", foil: true,
		},
		{
			desc:    "Eternal-Legal is CK's own name for the datastore's Eternal",
			title:   "Spider-Man, Miles Morales (Marvel's Spider-Man Eternal-Legal Foil CGC 9.5) #3066",
			name:    "Spider-Man, Miles Morales",
			edition: "Marvel's Spider-Man Eternal", foil: true,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := preprocessGraded(tt.title)
			if err != nil {
				t.Fatalf("preprocessGraded(%q) = %v", tt.title, err)
			}
			if got.Name != tt.name || got.Edition != tt.edition ||
				got.Variation != tt.variant || got.Foil != tt.foil {
				t.Errorf("preprocessGraded(%q) = %q/%q/%q foil=%v, want %q/%q/%q foil=%v",
					tt.title, got.Name, got.Edition, got.Variation, got.Foil,
					tt.name, tt.edition, tt.variant, tt.foil)
			}
		})
	}
}

// TestMatchGradedSecretLairCountdown pins matchGraded retrying an unknown
// "Secret Lair" edition against Secret Lair Countdown, and proves a card
// genuinely filed under the plain drop, ambiguous or not, still refuses
// exactly as it did before.
func TestMatchGradedSecretLairCountdown(t *testing.T) {
	b := realDatastore(t)
	for _, tt := range []struct {
		desc  string
		title string
		want  string // "" means still refused
	}{
		{
			desc:  "a halofoil printing only Secret Lair Countdown carries",
			title: "Hymn to Tourach (Secret Lair Halo Foil CGC Pristine 10) #9065",
			want:  "5435a718-eefc-504e-8316-b187336a83f1",
		},
		{
			desc:  "a nonfoil printing only Secret Lair Countdown carries",
			title: "Elite Spellbinder (Secret Lair CGC Pristine 10) #7125",
			want:  "a568851d-5da2-5089-8dab-a85db1796c3f",
		},
		{
			// Dark Ritual has two ordinary Secret Lair printings, so the
			// retry must not fire and mask the genuine ambiguity.
			desc:  "an ambiguous plain Secret Lair card stays refused",
			title: "Dark Ritual (Secret Lair Foil PSA 10) #5692",
			want:  "",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			theCard, err := preprocessGraded(tt.title)
			if err != nil {
				t.Fatalf("preprocessGraded(%q) = %v", tt.title, err)
			}
			cardID, err := matchGraded(b, theCard)
			if tt.want == "" {
				if err == nil {
					co, _ := b.GetUUID(cardID)
					t.Errorf("matchGraded(%v) = %s (%v), want it to stay refused", theCard, cardID, co)
				}
				return
			}
			if err != nil {
				t.Fatalf("matchGraded(%v) = %v", theCard, err)
			}
			if cardID != tt.want {
				co, _ := b.GetUUID(cardID)
				t.Errorf("matchGraded(%v) = %s (%v), want %s", theCard, cardID, co, tt.want)
			}
		})
	}
}

// TestMatchGradedClaimOnlyPrintings pins the tie between a plain printing and
// a japanshowcase or headliner twin: a title that never says Japan or
// Signature is the plain printing, and one that does never lands on it.
func TestMatchGradedClaimOnlyPrintings(t *testing.T) {
	b := realDatastore(t)
	for _, tt := range []struct {
		desc   string
		title  string
		number string // "" means it must stay refused
	}{
		{
			desc:   "a borderless title is the plain printing",
			title:  "Leonardo, Sewer Samurai (TMNT Foil (Borderless) CGC 10) #7145",
			number: "215",
		},
		{
			desc:   "a Japan Showcase title is the Japan printing",
			title:  "Leonardo, Cutting Edge (TMNT Foil (Japan Showcase) CGC Pristine 10) #7141",
			number: "281",
		},
		{
			desc:  "a Signature title is not taken for the plain printing",
			title: "Leonardo, Sewer Samurai (TMNT Foil (Borderless Signature) CGC 10) #7145",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			theCard, err := preprocessGraded(tt.title)
			if err != nil {
				t.Fatal(err)
			}
			cardID, err := matchGraded(b, theCard)
			if tt.number == "" {
				if err == nil {
					co, _ := b.GetUUID(cardID)
					t.Errorf("%q landed on %v, want it to stay refused", tt.title, co)
				}
				return
			}
			if err != nil {
				t.Fatalf("matchGraded(%v) = %v", theCard, err)
			}
			co, err := b.GetUUID(cardID)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != "TMT" || co.Number != tt.number {
				t.Errorf("%q landed on %s #%s, want TMT #%s", tt.title, co.SetCode, co.Number, tt.number)
			}
		})
	}
}

// TestParseGradedSlab pins the slab read off each title shape the storefront
// prints, the "Quad ++" tail the scraper has always read past, and a slab that
// does not fold being published as NM.
func TestParseGradedSlab(t *testing.T) {
	for _, tt := range []struct {
		title  string
		slab   mtgban.Slab
		grade  mtgban.Condition
		unread string
	}{
		{
			title: "Academy Ruins (Time Spiral Foil CGC 9) #5396",
			slab:  mtgban.Slab{Grader: "CGC", Score: 9}, grade: mtgban.NM,
		},
		{
			title: "Windswept Heath (Onslaught Foil CGC 7.5) #3072",
			slab:  mtgban.Slab{Grader: "CGC", Score: 7.5}, grade: mtgban.SP,
		},
		{
			title: "Arguel's Blood Fast (Ixalan Foil CGC Pristine 10) #7003",
			slab:  mtgban.Slab{Grader: "CGC", Score: 10, Label: "Pristine"}, grade: mtgban.NM,
		},
		{
			title: "Michelangelo, the Heart (TMNT Eternal Foil (Borderless) CGC Pristine 10) #7051",
			slab:  mtgban.Slab{Grader: "CGC", Score: 10, Label: "Pristine"}, grade: mtgban.NM,
		},
		{
			title: "Bad Moon - Inked (Alpha CGC Authentic Altered) #5002",
			slab:  mtgban.Slab{Grader: "CGC", Label: "Authentic Altered"}, grade: mtgban.PO,
		},
		{
			title: "Mox Diamond (Stronghold BGS 9.0) #3854",
			slab:  mtgban.Slab{Grader: "BGS", Score: 9}, grade: mtgban.NM,
		},
		{
			title: "Mox Diamond (Stronghold BGS 8.5) #3854",
			slab:  mtgban.Slab{Grader: "BGS", Score: 8.5}, grade: mtgban.SP,
		},
		{
			title: "Land Tax (4th Edition Alternate PSA 7) #3676",
			slab:  mtgban.Slab{Grader: "PSA", Score: 7}, grade: mtgban.NM,
		},
		{
			title: "Lion's Eye Diamond (Mirage BGS 9.5 Quad ++) #6775",
			slab:  mtgban.Slab{Grader: "BGS", Score: 9.5, Label: "Quad ++"}, grade: mtgban.NM,
		},
		{
			title: "Royal Assassin (Beta PSA Authentic) #6115",
			slab:  mtgban.Slab{Grader: "PSA", Label: "PSA Authentic"}, grade: mtgban.NM,
			unread: "PSA Authentic",
		},
		{
			title: "Royal Assassin (Beta PSA 11) #6115",
			slab:  mtgban.Slab{Grader: "PSA", Label: "PSA 11"}, grade: mtgban.NM,
			unread: "PSA 11",
		},
		{
			title: "Royal Assassin (Beta CGC 10.5) #6115",
			slab:  mtgban.Slab{Grader: "CGC", Score: 10}, grade: mtgban.NM,
		},
		{
			title: "Royal Assassin (Beta SGC 9) #6115",
			slab:  mtgban.Slab{Label: "SGC 9"}, grade: mtgban.NM,
			unread: "SGC 9",
		},
		{
			title: "Academy Ruins (Time Spiral Foil TAG 9.5) #5396",
			slab:  mtgban.Slab{Grader: "TAG", Score: 9.5}, grade: mtgban.NM,
		},
		{
			title: "Academy Ruins (Time Spiral Foil TAG) #5396",
			slab:  mtgban.Slab{Grader: "TAG", Label: "TAG"}, grade: mtgban.NM,
			unread: "TAG",
		},
		{
			title: "Academy Ruins (Time Spiral Foil CGC Pristine) #5396",
			slab:  mtgban.Slab{Grader: "CGC", Score: 10, Label: "Pristine"}, grade: mtgban.NM,
		},
		{
			title: "Academy Ruins (Time Spiral Foil CGC 8.25) #5396",
			slab:  mtgban.Slab{Grader: "CGC", Score: 8}, grade: mtgban.NM,
		},
		{
			title: "Academy Ruins (Time Spiral Foil BGS Quad 10) #5396",
			slab:  mtgban.Slab{Grader: "BGS", Score: 10, Label: "Black Label"}, grade: mtgban.NM,
		},
		{
			title: "Academy Ruins (Time Spiral Foil BGS Quad 9) #5396",
			slab:  mtgban.Slab{Grader: "BGS", Score: 9, Label: "Quad"}, grade: mtgban.NM,
		},
		{
			title: "Windswept Heath (Onslaught Foil CGC) #3024",
			slab:  mtgban.Slab{Grader: "CGC", Label: "CGC"}, grade: mtgban.NM,
			unread: "CGC",
		},
		{
			title: "Windswept Heath (Onslaught Foil) #3024",
			slab:  mtgban.Slab{Label: "Windswept Heath (Onslaught Foil) #3024"}, grade: mtgban.NM,
			unread: "Windswept Heath (Onslaught Foil) #3024",
		},
	} {
		t.Run(tt.title, func(t *testing.T) {
			slab, grade, unread := parseGradedSlab(tt.title)
			if slab == nil || *slab != tt.slab || grade != tt.grade || unread != tt.unread {
				t.Errorf("parseGradedSlab(%q) = %+v, %q, %q; want %+v, %q, %q",
					tt.title, slab, grade, unread, tt.slab, tt.grade, tt.unread)
			}
		})
	}
}

// TestGradedPageCarriesSlabs pins every entry of a page carrying its slab,
// two copies of one slab at one price staying apart by product id, and a slab
// that does not fold published as NM and logged.
func TestGradedPageCarriesSlabs(t *testing.T) {
	b := realDatastore(t)

	r, err := os.Open("testdata/graded_page.html")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		t.Fatal(err)
	}

	var logs []string
	ck := &Graded{inventory: mtgban.InventoryRecord{}, backend: b}
	ck.logCallback = func(format string, a ...any) {
		logs = append(logs, fmt.Sprintf(format, a...))
	}
	ck.parsePage(doc, 1)

	slabs := map[string]mtgban.Slab{}
	for _, entries := range ck.inventory {
		for _, entry := range entries {
			if entry.Slab == nil {
				t.Fatalf("entry %s has no slab", entry.OriginalID)
			}
			if entry.InstanceID != entry.OriginalID || entry.Quantity != 1 {
				t.Errorf("entry %s: instance %q, quantity %d", entry.OriginalID, entry.InstanceID, entry.Quantity)
			}
			slabs[entry.OriginalID] = *entry.Slab
		}
	}

	want := map[string]mtgban.Slab{
		"326940": {Grader: "CGC", Score: 10},
		"314595": {Grader: "CGC", Score: 10},
		"332494": {Grader: "CGC", Score: 8},
		"331192": {Grader: "CGC", Score: 8.5},
		"319260": {Grader: "CGC", Score: 10, Label: "Pristine"},
		"326480": {Grader: "CGC", Label: "Authentic Altered"},
		"270508": {Grader: "PSA", Label: "PSA Authentic"},
		"330176": {Label: "SGC 9"},
	}
	if !maps.Equal(slabs, want) {
		t.Errorf("slabs = %v, want %v", slabs, want)
	}
	wantLogs := []string{
		`[CKGraded] unsupported slab "PSA Authentic", published as NM`,
		`[CKGraded] unsupported slab "SGC 9", published as NM`,
	}
	if !slices.Equal(logs, wantLogs) {
		t.Errorf("logs = %q, want %q", logs, wantLogs)
	}
	if !ck.Info().GradedMode {
		t.Error("Info().GradedMode = false")
	}
}

// TestGradedAliasedSlabs pins a title that names several printings and no
// collector number being dropped, without a line only where its slab reads.
func TestGradedAliasedSlabs(t *testing.T) {
	b := realDatastore(t)
	for _, tt := range []struct {
		title  string
		logged bool
	}{
		{title: "Forest (Alpha CGC 9) #1"},
		{title: "Forest (Alpha SGC 9) #1", logged: true},
	} {
		t.Run(tt.title, func(t *testing.T) {
			page := `<div class="productListWrapper gradedMagic"><span class="productTitle"><a href="/f">` +
				tt.title + `</a></span><span class="itemPrice">$9.99</span></div>`
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(page))
			if err != nil {
				t.Fatal(err)
			}

			var logs []string
			ck := &Graded{inventory: mtgban.InventoryRecord{}, backend: b}
			ck.logCallback = func(format string, a ...any) {
				logs = append(logs, fmt.Sprintf(format, a...))
			}
			ck.parsePage(doc, 1)

			if len(ck.inventory) != 0 {
				t.Errorf("inventory = %v, want none", ck.inventory)
			}
			if !tt.logged && len(logs) != 0 {
				t.Errorf("logs = %q, want none", logs)
			}
			if tt.logged && !slices.Contains(logs, fmt.Sprintf("[CKGraded] %q", tt.title)) {
				t.Errorf("logs = %q, want the title", logs)
			}
		})
	}
}
