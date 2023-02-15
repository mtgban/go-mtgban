package amazon

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

const (
	amzAPIURL = "http://greatermossdogapi.us-east-1.elasticbeanstalk.com/api/v1/pricing/"
)

type AMZClient struct {
	client *retryablehttp.Client
}

type authTransport struct {
	Parent http.RoundTripper
	Token  string
}

func NewAMZClient(token string) *AMZClient {
	amz := AMZClient{}
	amz.client = retryablehttp.NewClient()
	amz.client.Logger = nil
	amz.client.HTTPClient.Transport = &authTransport{
		Parent: amz.client.HTTPClient.Transport,
		Token:  token,
	}
	return &amz
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Access-Token", t.Token)
	return t.Parent.RoundTrip(req)
}

func (amz *AMZClient) GetPrices(list []string) (map[string]map[string]float64, error) {
	resp, err := amz.client.Get(amzAPIURL + strings.Join(list, ","))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var prices map[string]map[string]float64
	err = json.Unmarshal(data, &prices)
	if err != nil {
		return nil, err
	}

	return prices, nil
}
