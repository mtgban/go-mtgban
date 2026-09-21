package coolstuffinc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-retryablehttp"
)

const (
	csiBuylistURL  = "https://www.coolstuffinc.com/GeneratedFiles/SellList/Section-%s.json"
	csiBuylistLink = "https://www.coolstuffinc.com/main_selllist.php?s="
)

// csiSearchURL is a variable so a test can point the searches at a server
// of its own.
var csiSearchURL = "https://www.coolstuffinc.com/sq/"

// csiClient is shared, and retries. Every request in this file stands for
// a whole run of rows rather than a page of them - an edition's singles, a
// sealed query, the buylist, the edition map - so a connection dropped
// once discards all of it silently. A client built per call cannot pool
// anything either, and the one it was built from disables keep-alives, so
// each request paid for a fresh handshake against a storefront being asked
// for hundreds of editions at a time.
var csiClient = newCSIHTTPClient()

// csiUserAgent is what the storefront is asked as. Go's default agent is
// answered with the bare site chrome in place of the page asked for - no
// search results, no facets, no next link - under a 200 and with no error
// anywhere, so only the missing rows say that anything went wrong. The
// header therefore sits on the transport rather than on each request,
// where a call site that forgot it has now cost this scraper its sealed
// pages once and its whole singles inventory a second time.
const csiUserAgent = "curl/8.6.0"

// userAgentTransport stamps the agent on every request that does not name
// one of its own, retries included.
type userAgentTransport struct {
	base http.RoundTripper
}

func (t userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		req = req.Clone(req.Context())
		req.Header.Set("User-Agent", csiUserAgent)
	}
	return t.base.RoundTrip(req)
}

func newCSIHTTPClient() *http.Client {
	client := retryablehttp.NewClient()
	client.Logger = nil
	standard := client.StandardClient()
	standard.Transport = userAgentTransport{base: standard.Transport}
	return standard
}

// CSIPriceEntry is one card in the buylist feed.
type CSIPriceEntry struct {
	PID         string `json:"PID"`
	Name        string `json:"Name"`
	ItemSet     string `json:"ItemSet"`
	Notes       string `json:"Notes"`
	Price       string `json:"Price"`
	Number      string `json:"Number"`
	RarityName  string `json:"RarityName"`
	IsFoil      int    `json:"isFoil"`
	CreditPrice string `json:"CreditPrice"`
	// Code is the set's own 3-4 letter code, published alongside ItemSet's
	// storefront name but otherwise unused here - the anchor a two-sided
	// token pairing needs, since its own name alone is never enough. See
	// preprocessTokenPairBuylist.
	Code string `json:"Code"`
	// Image is the storefront's own sku for the product - "SFD224SIG",
	// the same name the sale listings carry inside an image url. It
	// names the set and the collector number where the product name
	// names neither. See riftboundSKUCard.
	Image string `json:"Image"`
}

// GetBuylist returns what Cool Stuff Inc is buying on one storefront shelf.
func GetBuylist(ctx context.Context, shelf string) ([]CSIPriceEntry, error) {
	link := fmt.Sprintf(csiBuylistURL, shelf)

	// The sell list is a large uncompressed download that occasionally
	// truncates mid-stream (unexpected EOF), so retry the whole fetch.
	const attempts = 3
	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		var entries []CSIPriceEntry
		entries, err = fetchBuylist(ctx, link)
		if err == nil {
			return entries, nil
		}
		if attempt == attempts {
			break
		}
		// Back off before the next attempt, bailing out if the caller
		// gives up in the meantime
		select {
		case <-ctx.Done():
			return nil, err
		case <-time.After(time.Duration(attempt) * time.Second):
		}
	}

	return nil, err
}

func fetchBuylist(ctx context.Context, link string) ([]CSIPriceEntry, error) {
	data, err := fetchWhole(ctx, link)
	if err != nil {
		return nil, err
	}

	var entries []CSIPriceEntry
	err = json.Unmarshal(data, &entries)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// fetchWhole downloads a link in full, asking again for the part that did
// not arrive.
//
// The sell list is an 18MB uncompressed download and the storefront cuts it
// mid-stream about one fetch in three, answering 200 with a short body and
// no transport error, so only the decode notices. Retrying the whole file
// leaves the run failing whenever three attempts land badly, which is what
// reddened the Pokemon workflow.
//
// The response states Content-Length and the server honours Range, so a
// short body is both detectable and resumable: each pass asks only for the
// bytes still missing. The loop ends when the body is whole or when a pass
// got no further than the one before, rather than after a set number of
// tries.
func fetchWhole(ctx context.Context, link string) ([]byte, error) {
	var body []byte
	longest := 0
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
		if err != nil {
			return nil, err
		}

		// Ask for the bytes as they are stored. The storefront serves this
		// path uncompressed whatever is offered - it gzips its PHP pages
		// but not /GeneratedFiles - and saying so keeps Content-Length
		// meaningful, which is what the resume below measures against.
		req.Header.Set("Accept-Encoding", "identity")
		if len(body) > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", len(body)))
		}

		resp, err := csiClient.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode/100 != 2 {
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected %d status code", resp.StatusCode)
		}
		// A server that will not honour the range answers with the whole
		// file again, so what arrives replaces what we held instead of
		// extending it.
		if resp.StatusCode != http.StatusPartialContent {
			body = nil
		}

		have := len(body)
		chunk, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		body = append(body, chunk...)

		// Only Content-Length says how much there was to read, and it
		// survives only while the body arrives exactly as sent: leave the
		// transport to negotiate compression, or ask for it, and Go
		// answers -1 for a length it had to decompress. With nothing to
		// measure against there is no resuming and no telling a whole body
		// from a cut one, so a read that ended early has to be reported
		// rather than passed off as the file.
		if resp.ContentLength < 0 {
			if readErr != nil {
				return nil, readErr
			}
			return body, nil
		}

		// A body that ends early is the very shape this works around: keep
		// what arrived and ask for the rest. Anything else is a real error.
		if readErr != nil && !errors.Is(readErr, io.ErrUnexpectedEOF) {
			return nil, readErr
		}
		total := have + int(resp.ContentLength)
		if len(body) >= total {
			return body, nil
		}
		// A pass that got no further than the one before is a server that
		// will not hand over the rest, whether it stalled or started the
		// file over, and asking again only asks again.
		if len(body) <= longest {
			return nil, fmt.Errorf("read stalled at %d of %d bytes", len(body), total)
		}
		longest = len(body)
	}
}

// LoadBuylistEditions returns the edition-to-id map the storefront links are
// built from.
func LoadBuylistEditions(ctx context.Context, shelf string) (map[string]string, error) {
	link := csiBuylistLink + shelf
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := csiClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	edition2id := map[string]string{}

	doc.Find(`option`).Each(func(_ int, s *goquery.Selection) {
		ed := s.Text()
		if ed == "" {
			return
		}
		id, found := s.Attr("value")
		if !found || id == "" {
			return
		}
		_, found = edition2id[ed]
		if found {
			return
		}

		edition2id[ed] = id
	})

	return edition2id, nil
}

// SearchResult is the first page of a search.
type SearchResult struct {
	// NextLink is the href the storefront's own next-page control
	// carries, empty on the page the results end on. It is followed as
	// it stands: the storefront joins the page number to the saved
	// query with an "&" where a "?" belongs, and rebuilding the link
	// from its parts is a second guess at a spelling the page already
	// gives.
	NextLink string
	Data     []byte
}

// searchRowSelector picks the product rows out of a results page.
const searchRowSelector = `div[class="row product-search-row main-container"]`

// searchPageParam matches the page number in one of those links.
var searchPageParam = regexp.MustCompile(`([?&])page=\d+`)

// widenSearchPage reads a search's first page again at the larger page
// the storefront serves, and answers the page and the link after it.
//
// The search POST answers 25 rows however many it is asked for, but the
// links it writes honour 50 and carry the size into the links after
// them, so a walk that asks once keeps it to the end: a 541-row shelf
// costs 12 requests rather than 22. The page number has to go back to 1
// with it, because page 2 of a 50-row page is rows 51-100 - following
// the storefront's own "page=2" at the larger size steps over rows
// 26-50 without a word - so the first page is read a second time, which
// is what the rest of the walk is halved for.
//
// A widened page that answers no rows is the storefront declining the
// size, not the shelf ending: the link being widened is the one it
// wrote to say there is more. It answers nothing then, and the caller
// walks the shelf as the storefront linked it.
func widenSearchPage(ctx context.Context, client *http.Client, link string) (*goquery.Document, string) {
	wide := searchPageParam.ReplaceAllString(link, "${1}resultsPerPage=50&page=1")
	// A link carrying no page number is one this cannot move back to the
	// first page, and following it unchanged would read the second page
	// as though it were the first and lose everything before it. The
	// walk keeps its own first page instead.
	if wide == link {
		return nil, ""
	}
	doc, err := fetchSearchPage(ctx, client, wide)
	if err != nil || doc.Find(searchRowSelector).Length() == 0 {
		return nil, ""
	}
	return doc, searchNextLink(doc)
}

// searchNextLink reads the href of a results page's next-page control.
func searchNextLink(doc *goquery.Document) string {
	next, _ := doc.Find(`span[id="nextLink"]`).Find("a").Attr("href")
	return next
}

// fetchSearchPage follows one of those links and parses the page it
// answers with.
func fetchSearchPage(ctx context.Context, client *http.Client, link string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Whatever the storefront answers a refusal with parses as a page
	// holding no rows and linking nowhere, which reads as the shelf
	// ending rather than as the error it is - the same shape that cost
	// this scraper its whole inventory once already. Say so instead: the
	// worker pool logs the shelf and carries on with the others.
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("unexpected %d status code for %s", resp.StatusCode, link)
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

// Search returns the first page of an item name's results, narrowed to
// the given rarity tiers, and the link to the page after it.
func Search(ctx context.Context, shelf, itemName string, skipOOS bool, rarities []string) (*SearchResult, error) {
	v := url.Values{}
	v.Set("name", "")
	v.Set("f[Artist][]", "")
	v.Add("f[Cost][]", "")
	v.Add("f[Cost][]", "")
	v.Set("f[Number][]", "")
	v.Set("f[Type][]", "")
	v.Set("f[Card+Text][]", "")
	v.Set("notes", "")
	v.Set("sign-Cost", "<")
	v.Set("sign-Power", "<")
	v.Set("f[Power][]", "")
	v.Set("sign-Toughness", "<")
	v.Set("f[Toughness][]", "")
	v.Set("sign-Loyalty", "<")
	v.Set("f[Loyalty][]", "")
	v.Set("signprice", "<")
	v.Set("price", "")
	if skipOOS {
		// This excludes all cards that lack a NM copy
		v.Set("options[instock]", "1")
	}
	// Naming every tier the game has but the sealed ones keeps sealed out
	// of a singles search, since a rarity constraint of any kind excludes
	// the rows that carry no rarity at all. An empty list asks for
	// everything, which costs a sealed row the condition parser then
	// refuses - never a card.
	for _, rarity := range rarities {
		v.Add("f[Rarity][]", rarity)
	}
	v.Set("f[ItemSet][]", itemName)
	v.Set("s", shelf)
	v.Set("page", "1")
	// 25 and 50 are the only values the search takes, and it serves 25 either way
	v.Set("resultsPerPage", "25")
	v.Set("submit", "Search")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, csiSearchURL, strings.NewReader(v.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	resp, err := csiClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	return &SearchResult{
		NextLink: searchNextLink(doc),
		Data:     data,
	}, nil
}
