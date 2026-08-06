package starcitygames

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

const (
	// The set index is served by the same public MeiliSearch instance the
	// sell-your-cards storefront queries, with the search-only key embedded in
	// its frontend.
	scgSetsURL       = "https://search.starcitygames.com/indexes/sets_v2/search"
	scgSetsSearchKey = "f0b723e5fddfe6edbdad6fecc814511511a62b1722c6c30cff42be5bca4ed834"

	sellYourCardsHost = "https://sellyourcards.starcitygames.com"
)

// scgLanguageCodes maps the catalog language names to the codes the
// sell-your-cards bookmark links expect.
var scgLanguageCodes = map[string]string{
	"English":               "en",
	"German":                "de",
	"Spanish":               "es",
	"French":                "fr",
	"Italian":               "it",
	"Japanese":              "ja",
	"Korean":                "ko",
	"Portuguese":            "pt",
	"Russian":               "ru",
	"Chinese - Simplified":  "zs",
	"Chinese - Traditional": "zt",
}

type scgSet struct {
	Name        string `json:"name"`
	SetID       int    `json:"set_id"`
	GameID      int    `json:"game_id"`
	WizardsCode string `json:"wizards_code"`
}

// SetIDs returns the set name -> set_id map for a game, as used by the
// sell-your-cards bookmark links.
func (scg *SCGClient) SetIDs(ctx context.Context, game int) (map[string]int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, scgSetsURL,
		strings.NewReader(`{"q":"","limit":2000}`))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+scgSetsSearchKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := scg.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var page struct {
		Hits []scgSet `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, err
	}

	// Key by both the set name (matched fuzzily against a single's catalog set)
	// and the Wizards code (an exact fallback for the set code carried in a
	// SKU). A short code never fuzzy-matches a multi-word set name, so the two
	// keyspaces don't interfere.
	out := map[string]int{}
	for _, s := range page.Hits {
		if s.GameID != game {
			continue
		}
		if s.Name != "" {
			out[s.Name] = s.SetID
		}
		if s.WizardsCode != "" {
			out[s.WizardsCode] = s.SetID
		}
	}
	return out, nil
}

// setIDsForProduct resolves the sell-your-cards set ids for a catalog product:
// by the catalog set name (fuzzily), and when that is empty or unknown -- as it
// is for sealed and a few uncategorized singles -- by the set code carried in
// the SKU (SGL-<brand>-<set>-...), tried raw then normalized.
func setIDsForProduct(setIDs map[string]int, setName, sku string) []int {
	if ids := matchSetIDs(setIDs, setName); len(ids) > 0 {
		return ids
	}
	fields := strings.Split(sku, "-")
	if len(fields) > 2 {
		if id, ok := setIDs[fields[2]]; ok {
			return []int{id}
		}
		if id, ok := setIDs[fixupSetCode(fields[2])]; ok {
			return []int{id}
		}
	}
	return nil
}

func setWords(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// matchSetIDs resolves a catalog set name to sell-your-cards set ids. An exact
// name wins outright; otherwise every set sharing the longest leading-word
// prefix with the query is returned, so a name the index doesn't carry verbatim
// still lands on the closest set group -- "Marvel Super Heroes Commander" pulls
// in the Marvel Super Heroes sets, "Promo: General" falls back to "Promo".
func matchSetIDs(setIDs map[string]int, query string) []int {
	query = strings.TrimSuffix(query, " (Foil)")
	if id, ok := setIDs[query]; ok {
		return []int{id}
	}

	qwords := setWords(query)
	if len(qwords) == 0 {
		return nil
	}

	best := 0
	byLen := map[int][]int{}
	for name, id := range setIDs {
		n := leadingWordMatch(qwords, setWords(name))
		if n == 0 {
			continue
		}
		byLen[n] = append(byLen[n], id)
		if n > best {
			best = n
		}
	}
	return byLen[best]
}

func leadingWordMatch(a, b []string) int {
	n := 0
	for n < len(a) && n < len(b) && a[n] == b[n] {
		n++
	}
	return n
}

func gamePath(game int) string {
	switch game {
	case GameLorcana:
		return "lorcana"
	case GameRiftbound:
		return "riftbound"
	}
	return "mtg"
}

// SCGBuylistURL builds the sell-your-cards bookmark link that lists a product
// for sale to SCG, filtered to the given set ids (as opposed to the retail page
// SCGProductURL points to). The segment order mirrors the storefront's own URL
// builder. Returns "" if no set ids are given.
func SCGBuylistURL(game int, name, language string, setIDs []int) string {
	if len(setIDs) == 0 {
		return ""
	}

	ids := make([]string, len(setIDs))
	for i, id := range setIDs {
		ids[i] = strconv.Itoa(id)
	}

	nameSeg := "0"
	if name != "" {
		nameSeg = url.PathEscape(name)
	}
	langSeg := ","
	if code := scgLanguageCodes[language]; code != "" {
		langSeg = code
	}

	segments := []string{
		gamePath(game),
		"bookmark",
		nameSeg,                  // cardName
		"0",                      // cardNameExactMatch
		"0",                      // filterOutBulkProducts
		"0",                      // filterOnlyHotlist
		"0",                      // exportAsCSV
		strings.Join(ids, "%2C"), // set_ids
		langSeg,                  // languages
		",",                      // rarities
		"0",                      // cardPurchasePriceMin
		"999999.99",              // cardPurchasePriceMax
		",",                      // finishes
		"setReleaseDateAsc",      // sort
	}
	return sellYourCardsHost + "/" + strings.Join(segments, "/")
}
