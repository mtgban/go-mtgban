package mtgstocks

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/corpix/uarand"
	"github.com/hashicorp/go-retryablehttp"
)

type StocksInterest struct {
	InterestType string  `json:"interest_type"`
	Foil         bool    `json:"foil"`
	Percentage   float64 `json:"percentage"`
	PastPrice    float64 `json:"past_price"`
	PresentPrice float64 `json:"present_price"`
	Date         int64   `json:"date"`
	Print        struct {
		Id        int         `json:"id"`
		Slug      interface{} `json:"slug"` // string & int
		Name      string      `json:"name"`
		Rarity    string      `json:"rarity"`
		SetId     int         `json:"set_id"`
		SetName   string      `json:"set_name"`
		IconClass string      `json:"icon_class"`
		Reserved  bool        `json:"reserved"`
		SetType   string      `json:"set_type"`
		Legal     struct {
			Frontier  string `json:"frontier"`
			Pauper    string `json:"pauper"`
			Pioneer   string `json:"pioneer"`
			Modern    string `json:"modern"`
			Standard  string `json:"standard"`
			Commander string `json:"commander"`
			Vintage   string `json:"vintage"`
			Legacy    string `json:"legacy"`
		}
		IncludeDefault bool   `json:"include_default"`
		Image          string `json:"image"`
	} `json:"print"`
}

type MTGStocksInterests struct {
	Error     string           `json:"error"`
	Date      string           `json:"date"`
	Interests []StocksInterest `json:"interests"`
}

const (
	stksAverageURL = "https://api.mtgstocks.com/interests/average"
	stksMarketURL  = "https://api.mtgstocks.com/interests/market"
	stksSetsURL    = "https://api.mtgstocks.com/card_sets"
)

type STKSClient struct {
	client *retryablehttp.Client
	ua     string
}

func NewClient() *STKSClient {
	stks := STKSClient{}
	stks.client = retryablehttp.NewClient()
	stks.client.Backoff = retryablehttp.LinearJitterBackoff
	stks.client.RetryWaitMin = 2 * time.Second
	stks.client.RetryWaitMax = 10 * time.Second
	stks.client.RetryMax = 10
	stks.client.CheckRetry = customCheckRetry
	stks.client.PrepareRetry = customPrepareRetry
	stks.ua = uarand.GetRandom()
	return &stks
}

// Implement our own retry policy to leverage the internal retry mechanism
func customCheckRetry(ctx context.Context, resp *http.Response, err error) (bool, error) {
	defer resp.Body.Close()

	var reader io.Reader = resp.Body
	if strings.ToLower(resp.Header.Get("Content-Encoding")) == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return false, err
		}
		defer gzipReader.Close()
		reader = gzipReader
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return false, err
	}

	resp.Body = io.NopCloser(bytes.NewBuffer(data))
	if strings.Contains(string(data), "HTML") || strings.Contains(string(data), "ERROR") {
		return true, errors.New(string(data))
	}
	return false, nil
}

// Change user agent before another retry
func customPrepareRetry(req *http.Request) error {
	req.Header.Set("User-Agent", uarand.GetRandom())
	return nil
}

func (s *STKSClient) AverageInterests(ctx context.Context, foil bool) ([]StocksInterest, error) {
	out, err := s.query(ctx, stksAverageURL, foil)
	if err != nil {
		return nil, err
	}
	return out.Interests, nil
}

func (s *STKSClient) MarketInterests(ctx context.Context, foil bool) ([]StocksInterest, error) {
	out, err := s.query(ctx, stksMarketURL, foil)
	if err != nil {
		return nil, err
	}
	return out.Interests, nil
}

func (s *STKSClient) query(ctx context.Context, link string, foil bool) (*MTGStocksInterests, error) {
	extra := "/regular"
	if foil {
		extra = "/foil"
	}
	link += extra

	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", s.ua)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "https://www.mtgstocks.com/")
	req.Header.Set("Origin", "https://www.mtgstocks.com")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, identity")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("DNT", "1")

	resp, err := s.client.StandardClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var reader io.Reader = resp.Body
	if strings.ToLower(resp.Header.Get("Content-Encoding")) == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	var interests MTGStocksInterests
	err = json.NewDecoder(reader).Decode(&interests)
	if err != nil {
		return nil, err
	}

	if interests.Error != "" {
		return nil, errors.New(interests.Error)
	}

	return &interests, nil
}
