package coolstuffinc

import (
	"encoding/json"
	"io"

	"github.com/PuerkitoBio/goquery"
	cleanhttp "github.com/hashicorp/go-cleanhttp"
	http "github.com/hashicorp/go-retryablehttp"
)

type CSICard struct {
	Id             int     `json:"id,string"`
	URL            string  `json:"url"`
	Name           string  `json:"name"`
	ScryfallId     string  `json:"scryfallid"`
	Variation      string  `json:"variation"`
	Edition        string  `json:"edition"`
	Language       string  `json:"language"`
	IsFoil         bool    `json:"is_foil,string"`
	PriceRetail    float64 `json:"price_retail,string"`
	QuantityRetail int     `json:"qty_retail,string"`
	PriceBuy       float64 `json:"price_buy,string"`
	QuantityBuy    int     `json:"qty_buying,string"`
}

const (
	csiPricelistURL = "https://www.coolstuffinc.com/gateway_json.php?k="

	csiBuylistURL  = "https://www.coolstuffinc.com/GeneratedFiles/SellList/Section-mtg.json"
	csiBuylistLink = "https://www.coolstuffinc.com/main_selllist.php?s=mtg"
)

type CSIClient struct {
	client *http.Client
	key    string
}

func NewCSIClient(key string) *CSIClient {
	csi := CSIClient{}
	csi.client = http.NewClient()
	csi.client.Logger = nil
	csi.key = key
	return &csi
}

func (csi *CSIClient) GetPriceList() ([]CSICard, error) {
	resp, err := csi.client.Get(csiPricelistURL + csi.key)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var pricelist struct {
		Meta struct {
			CreatedAt string `json:"created_at"`
		} `json:"meta"`
		Data []CSICard `json:"data"`
	}
	err = json.Unmarshal(data, &pricelist)
	if err != nil {
		return nil, err
	}

	return pricelist.Data, nil
}

type CSIPriceEntry struct {
	// Pid           string `json:"PID"`
	// Ppqid         string `json:"PPQID"`
	Name string `json:"Name"`
	// Rarity        string `json:"Rarity"`
	ItemSet string `json:"ItemSet"`
	// Image   string `json:"Image"`
	Notes string `json:"Notes"`
	// SName         string `json:"sName"`
	// SAbbreviation string `json:"sAbbreviation"`
	Price string `json:"Price"`
	// TName         string `json:"tName"`
	// Color         string `json:"Color"`
	Number string `json:"Number"`
	// Code   string `json:"Code"`
	// BuyListNotes  string `json:"BuyListNotes"`
	// FullImage     struct {} `json:"FullImage"`
	RarityName  string `json:"RarityName"`
	IsFoil      int    `json:"isFoil"`
	CreditPrice string `json:"CreditPrice"`
}

func GetBuylist() ([]CSIPriceEntry, error) {
	resp, err := cleanhttp.DefaultClient().Get(csiBuylistURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
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

// Load the list of editions to id used to build links
func LoadBuylistEditions() (map[string]string, error) {
	resp, err := cleanhttp.DefaultClient().Get(csiBuylistLink)
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
