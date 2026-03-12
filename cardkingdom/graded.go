package cardkingdom

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/RomainMichau/cloudscraper_go/cloudscraper"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	gradedURL = "https://www.cardkingdom.com/mtg/graded-magic"
)

type CardkingdomGraded struct {
	LogCallback mtgban.LogCallbackFunc
	Partner     string

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord

	client *cloudscraper.CloudScrapper
}

func NewScraperGraded() (*CardkingdomGraded, error) {
	client, err := cloudscraper.Init(false, false)
	if err != nil {
		return nil, err
	}

	ck := CardkingdomGraded{}
	ck.inventory = mtgban.InventoryRecord{}
	ck.client = client

	return &ck, nil
}

func (ck *CardkingdomGraded) printf(format string, a ...interface{}) {
	if ck.LogCallback != nil {
		ck.LogCallback("[CKGraded] "+format, a...)
	}
}

func (ck *CardkingdomGraded) totalPages() (string, int, error) {
	cookieMap := map[string]string{
		"Cookie": "limit=100; sortBy=price_desc; viewType=listShowCart listShowDetails;",
	}
	res, err := ck.client.Get(gradedURL, cookieMap, http.MethodGet)
	if err != nil {
		return "", 0, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(res.Body))
	if err != nil {
		return "", 0, err
	}

	lastPageText := strings.TrimSpace(doc.Find("a[aria-label='Display Final Results Page']").Text())
	pages, err := strconv.Atoi(lastPageText)
	if err != nil {
		return "", 0, fmt.Errorf("could not find final page link: %w", err)
	}

	var session string
	for _, c := range res.Cookies {
		if c.Name == "laravel_session" {
			session = c.Value
		}
	}
	if session == "" {
		return "", 0, fmt.Errorf("could not find session cookie")
	}

	return session, pages, nil
}

func (ck *CardkingdomGraded) scrapePage(session string, page int) error {
	cookieMap := map[string]string{
		"Cookie": "limit=100; sortBy=price_desc; viewType=listShowCart listShowDetails; laravel_session=" + session + ";",
	}
	res, err := ck.client.Get(gradedURL+"?page="+fmt.Sprint(page), cookieMap, http.MethodGet)
	if err != nil {
		return err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(res.Body))
	if err != nil {
		return err
	}

	u, _ := url.Parse("https://www.cardkingdom.com/")
	if ck.Partner != "" {
		q := u.Query()
		q.Set("partner", ck.Partner)
		q.Set("utm_source", ck.Partner)
		q.Set("utm_medium", "affiliate")
		q.Set("utm_campaign", ck.Partner)
		u.RawQuery = q.Encode()
	}

	doc.Find(".productListWrapper.gradedMagic").Each(func(i int, s *goquery.Selection) {
		if s.Find("form.addToCartForm").HasClass("noInventory") {
			return
		}

		// Title
		title := strings.TrimSpace(s.Find(".productTitle a").Text())

		theCard, err := preprocessGraded(title)
		if err != nil {
			if !errors.Is(err, mtgmatcher.ErrUnsupported) {
				ck.printf("%s: %v", title, err)
			}
			return
		}

		cardId, err := mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			return
		} else if err != nil {
			ck.printf("%v", err)
			ck.printf("%q", theCard)
			ck.printf("%q", title)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					ck.printf("- %s", card)
				}
			}
			return
		}

		// Price
		priceStr := strings.TrimSpace(s.Find(".itemPrice").First().Text())
		price, err := mtgmatcher.ParsePrice(priceStr)
		if err != nil {
			ck.printf("%s", err.Error())
			return
		}

		// URL (relative href)
		linkPath, _ := s.Find(".productTitle a").Attr("href")
		u.Path = linkPath
		link := u.String()

		// Product ID from hidden input
		id, _ := s.Find("input.product_id").Attr("value")

		out := &mtgban.InventoryEntry{
			Conditions: parseGradedCondition(title),
			Price:      price,
			URL:        link,
			OriginalId: id,
		}
		err = ck.inventory.Add(cardId, out)
		if err != nil {
			ck.printf("page %d: %s", page, err.Error())
			return
		}
	})

	return nil
}

func (ck *CardkingdomGraded) Load(ctx context.Context) error {
	session, pages, err := ck.totalPages()
	if err != nil {
		return err
	}

	ck.printf("Found %d pages", pages)

	start := time.Now()

	for i := 1; i <= pages; i++ {
		ck.printf("Scraping page %d", i)
		err := ck.scrapePage(session, i)
		if err != nil {
			ck.printf("%s", err.Error())
		}
	}

	ck.printf("Took %v", time.Since(start))
	ck.inventoryDate = time.Now()

	return nil
}

func (ck *CardkingdomGraded) Inventory() mtgban.InventoryRecord {
	return ck.inventory
}

func (ck *CardkingdomGraded) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Kingdom Graded"
	info.Shorthand = "CKGraded"
	info.InventoryTimestamp = &ck.inventoryDate
	return
}
