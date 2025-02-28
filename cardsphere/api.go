package cardsphere

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

type CardSphereClient struct {
	client *retryablehttp.Client
}

func NewCardSphereClient(token string) *CardSphereClient {
	cs := CardSphereClient{}
	cs.client = retryablehttp.NewClient()
	cs.client.Logger = nil
	// The api is very sensitive to multiple concurrent requests,
	// This backoff strategy lets the system chill out a bit before retrying
	cs.client.Backoff = retryablehttp.LinearJitterBackoff
	cs.client.RetryWaitMin = 2 * time.Second
	cs.client.RetryWaitMax = 10 * time.Second
	cs.client.RetryMax = 20

	jar, _ := cookiejar.New(nil)

	var cookies []*http.Cookie
	cookie := &http.Cookie{
		Name:   "cardsphere-session-5",
		Value:  token,
		Path:   "/",
		Domain: ".cardsphere.com",
	}
	cookies = append(cookies, cookie)

	u, _ := url.Parse(csURL)
	u.RawQuery = ""
	jar.SetCookies(u, cookies)

	cs.client.HTTPClient.Jar = jar
	return &cs
}

type CardSphereOfferList struct {
	WantId      int     `json:"wantId"`
	MinOffer    int     `json:"minOffer"`
	MaxOffer    int     `json:"maxOffer"`
	MinIndex    int     `json:"minIndex"`
	MaxIndex    int     `json:"maxIndex"`
	MinEff      int     `json:"minEff"`
	MaxEff      int     `json:"maxEff"`
	MinRelEff   float64 `json:"minRelEff"`
	MaxRelEff   float64 `json:"maxRelEff"`
	MasterId    int     `json:"masterId"`
	Image       string  `json:"image"`
	UserId      int     `json:"userId"`
	UserDisplay string  `json:"userDisplay"`
	Country     string  `json:"country"`
	CountryName string  `json:"countryName"`
	Balance     int     `json:"balance"`
	CardName    string  `json:"cardName"`
	Kind        string  `json:"kind"`
	Sets        []struct {
		Code   string `json:"code"`
		Name   string `json:"name"`
		Rarity string `json:"rarity"`
	} `json:"sets"`
	Languages  []string `json:"languages"`
	Conditions []int    `json:"conditions"`
	Finishes   []string `json:"finishes"`
	Quantity   int      `json:"quantity"`
}

// Convenience error message to simplify checking
type csError struct {
	Message string `json:"message"`
}

const csURL = "https://www.cardsphere.com/rest/v1/offers?offset=0&order=minrel&absge=50&country=USMIL,UM,US,CA&kind=S&language=EN"

func (cs *CardSphereClient) GetOfferList(offset int) ([]CardSphereOfferList, error) {
	u, err := url.Parse(csURL)
	if err != nil {
		return nil, err
	}
	v := u.Query()
	v.Set("offset", fmt.Sprint(offset))
	u.RawQuery = v.Encode()

	req, err := retryablehttp.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", "curl/8.6.0")

	resp, err := cs.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var pricelist []CardSphereOfferList
	err = json.Unmarshal(data, &pricelist)
	if err != nil {
		var msg csError
		errSub := json.Unmarshal(data, &msg)
		if errSub != nil {
			err = errors.New(err.Error() + "->" + errSub.Error())
		} else {
			err = errors.New(msg.Message)
		}
		return nil, err
	}

	return pricelist, nil
}
