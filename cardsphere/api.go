package cardsphere

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/cookiejar"
	"net/url"
	"strings"

	http "github.com/hashicorp/go-retryablehttp"
)

type CardSphereClient struct {
	client *http.Client
}

func NewCardSphereClient(email, password string) (*CardSphereClient, error) {
	cs := CardSphereClient{}
	cs.client = http.NewClient()
	cs.client.Logger = nil
	jar, _ := cookiejar.New(nil)
	cs.client.HTTPClient.Jar = jar

	resp, err := cs.client.PostForm("https://www.cardsphere.com/login", url.Values{
		"email":    {email},
		"password": {password},
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if strings.Contains(string(data), "Invalid credentials") {
		return nil, errors.New("invalid credentials")
	}

	return &cs, nil
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

func (cs *CardSphereClient) GetOfferListByMaxRelative(offset int) ([]CardSphereOfferList, error) {
	return cs.getOfferList(offset, "maxrel")
}

func (cs *CardSphereClient) GetOfferListByMaxAbsolute(offset int) ([]CardSphereOfferList, error) {
	return cs.getOfferList(offset, "maxabs")
}

func (cs *CardSphereClient) getOfferList(offset int, mode string) ([]CardSphereOfferList, error) {
	u, err := url.Parse("https://www.cardsphere.com/rest/v1/offers")
	if err != nil {
		return nil, err
	}
	v := url.Values{}
	v.Set("offset", fmt.Sprint(offset))
	v.Set("order", mode)
	v.Set("country", "CA,MX,US")
	v.Set("kind", "S")
	v.Set("language", "EN")
	u.RawQuery = v.Encode()

	resp, err := cs.client.Get(u.String())
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
