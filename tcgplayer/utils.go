package tcgplayer

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	BaseProductURL    = "https://www.tcgplayer.com/product/"
	PartnerProductURL = "https://partner.tcgplayer.com/c/%s/1830156/21018"
)

func GenerateProductURL(productId int, printing, affiliate, condition, language string, isDirect bool) string {
	u, err := url.Parse(BaseProductURL + fmt.Sprint(productId))
	if err != nil {
		return ""
	}

	v := u.Query()
	if printing != "" {
		v.Set("Printing", printing)
	}
	if condition != "" {
		for full, short := range conditionMap {
			if short == condition {
				condition = full
				break
			}
		}
		v.Set("Condition", condition)
	}
	if language != "" {
		language = mtgmatcher.Title(language)
		switch language {
		case "Portuguese (Brazil)":
			language = "Portuguese"
		case "Chinese Simplified":
			language = "Chinese (S)"
		case "Chinese Traditional":
			language = "Chinese (T)"
		}
		v.Set("Language", language)
	} else {
		v.Set("Language", "all")
	}
	v.Set("direct", "false")
	if isDirect {
		v.Set("direct", "true")
	}

	// This chunk needs to be last, stash the built link in a query param
	// and use the impact URL instead
	if affiliate != "" {
		u.RawQuery = v.Encode()
		link := u.String()

		u, err = url.Parse(fmt.Sprintf(PartnerProductURL, affiliate))
		if err != nil {
			return ""
		}

		v = url.Values{}
		v.Set("u", link)
	}

	u.RawQuery = v.Encode()

	return u.String()
}

type TCGSku struct {
	Condition string `json:"condition"`
	Language  string `json:"language"`
	Printing  string `json:"printing"`
	Finish    string `json:"finish"`
	ProductId int    `json:"productId"`
	SkuId     int    `json:"skuId"`
}

type SKUMap map[string][]TCGSku

func LoadTCGSKUs(reader io.Reader) (SKUMap, error) {
	var payload struct {
		Data map[string][]TCGSku `json:"data"`
	}
	err := json.NewDecoder(reader).Decode(&payload)
	if err != nil {
		return nil, err
	}
	if len(payload.Data) == 0 {
		return nil, errors.New("empty SKU file")
	}
	return payload.Data, nil
}

const (
	SYPCSVURL = "https://store.tcgplayer.com/admin/direct/ExportSYPList?categoryid=1&setNameId=All&conditionId=All"
)

type TCGSYP struct {
	SkuId       int
	MarketPrice float64
	MaxQty      int
}

func LoadSyp(ctx context.Context, auth string) ([]TCGSYP, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, SYPCSVURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", "TCGAuthTicket_Production="+auth)

	resp, err := cleanhttp.DefaultClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	csvReader := csv.NewReader(resp.Body)
	csvReader.ReuseRecord = true

	var result []TCGSYP
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			continue
		}

		if len(record) < 9 {
			continue
		}

		id, err := strconv.Atoi(record[0])
		if err != nil {
			continue
		}
		price, err := strconv.ParseFloat(record[7], 64)
		if err != nil {
			continue
		}
		qty, err := strconv.Atoi(record[8])
		if err != nil {
			continue
		}

		result = append(result, TCGSYP{
			SkuId:       id,
			MarketPrice: price,
			MaxQty:      qty,
		})
	}

	if len(result) == 0 {
		return nil, errors.New("empty syp csv")
	}

	return result, nil
}

func DirectPriceAfterFees(price float64) float64 {
	directCost := 0.3 + price*(0.0895+0.025)

	var replacementFees float64
	if price < 3 {
		replacementFees = price / 2
		directCost = 0
	} else if price < 20 {
		replacementFees = 1.12
	} else if price < 250 {
		replacementFees = 3.97
	} else {
		replacementFees = 6.85
	}

	return price - directCost - replacementFees
}

const (
	defaultListingSize = 20
)

type ListingData struct {
	ProductId       int     `json:"product_id"`
	SkuId           int     `json:"sku_id"`
	Quantity        int     `json:"quantity"`
	SellerKey       string  `json:"seller_key"`
	Price           float64 `json:"price"`
	DirectInventory int     `json:"direct_inventory"`
	ConditionFull   string  `json:"condition_full"`
	Condition       string  `json:"condition"`
	Printing        string  `json:"printing"`
	Foil            bool    `json:"foil"`
}

func GetDirectQtysForProductId(ctx context.Context, productId int, onlyDirect bool) []ListingData {
	client := NewSellerClient()

	var result []ListingData
	for i := 0; ; i++ {
		listings, err := client.InventoryListing(ctx, productId, defaultListingSize, i, onlyDirect)
		if err != nil || len(listings) == 0 {
			break
		}

		for _, listing := range listings {
			if !listing.DirectProduct || !listing.DirectSeller {
				continue
			}

			result = append(result, ListingData{
				ProductId:       productId,
				SkuId:           int(listing.ProductConditionID),
				Quantity:        int(listing.Quantity),
				SellerKey:       listing.SellerKey,
				Price:           listing.Price,
				DirectInventory: int(listing.DirectInventory),
				ConditionFull:   listing.Condition,
				Condition:       conditionMap[listing.Condition],
				Printing:        listing.Printing,
				Foil:            listing.Printing != "Normal",
			})
		}
	}

	return result
}
