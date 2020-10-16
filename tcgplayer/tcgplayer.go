package tcgplayer

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"net/http"

	"github.com/PuerkitoBio/goquery"

	"github.com/kodabb/go-mtgban/mtgban"
)

type requestChan struct {
	TCGProductId string
	UUID         string
	retry        int
}

type responseChan struct {
	cardId string
	entry  mtgban.InventoryEntry
	bl     mtgban.BuylistEntry
}

func getListingsNumber(client *http.Client, productId string) (int, error) {
	u, _ := url.Parse(tcgBaseURL)
	q := u.Query()
	q.Set("productId", productId)
	q.Set("pageSize", fmt.Sprintf("%d", 1))
	q.Set("page", fmt.Sprintf("%d", 1))
	u.RawQuery = q.Encode()

	resp, err := client.Get(u.String())
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0, err
	}

	viewingResults := doc.Find("span[class='sort-toolbar__total-item-count']").Text()
	results := strings.Fields(viewingResults)
	if len(results) < 3 {
		return 0, fmt.Errorf("unknown pagination for %s: %q", productId, viewingResults)
	}
	entriesNum, err := strconv.Atoi(results[3])
	if err != nil {
		return 0, err
	}

	return entriesNum, nil
}
