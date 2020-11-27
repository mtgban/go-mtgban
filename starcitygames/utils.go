package starcitygames

import (
	"bufio"
	"encoding/json"
	"errors"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

type productList struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"abbr"`
}

const (
	scgLoginURL    = "https://login.starcitygames.com/login"
	scgCategoryURL = "https://old.starcitygames.com/buylist"
)

type SCGBuylistClient struct {
	client *retryablehttp.Client
}

func NewSCGBuylistClient(username, password string) (*SCGBuylistClient, error) {
	scg := SCGBuylistClient{}
	scg.client = retryablehttp.NewClient()
	scg.client.Logger = nil

	jar, _ := cookiejar.New(nil)
	scg.client.HTTPClient.Jar = jar

	token, err := scg.getToken()
	if err != nil {
		return nil, err
	}

	err = scg.login(username, password, token)
	if err != nil {
		return nil, err
	}

	return &scg, nil
}

func (scg *SCGBuylistClient) getToken() (string, error) {
	resp, err := scg.client.Get(scgLoginURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	token, ok := doc.Find(`input[name="_token"]`).Attr("value")
	if !ok {
		return "", errors.New("token not found")
	}

	return token, nil
}

func (scg *SCGBuylistClient) login(username, password, token string) error {
	resp, err := scg.client.PostForm(scgLoginURL, url.Values{
		"username": {username},
		"password": {password},
		"action":   {"login"},
		"_token":   {token},
	})
	if err != nil {
		return err
	}
	// The response is just a redirect scheme, nothing intersting
	resp.Body.Close()
	return nil
}

func (scg *SCGBuylistClient) ParseCategories() ([]productList, error) {
	resp, err := scg.client.Get(scgCategoryURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var line string
	var found bool
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line = scanner.Text()
		if strings.Contains(line, "categories = ") {
			found = true
			break
		}
	}
	if !found {
		return nil, errors.New("categories array not found")
	}

	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "categories = ")
	line = strings.TrimSuffix(line, ",")

	var list []productList
	d := json.NewDecoder(strings.NewReader(line))
	err = d.Decode(&list)

	return list, err
}
