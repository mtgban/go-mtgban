package tcgplayer

import (
	"compress/bzip2"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/mtgjson"
)

const (
	BaseProductURL    = "https://www.tcgplayer.com/product/"
	PartnerProductURL = "https://tcgplayer.pxf.io/c/%s/1830156/21018"
)

func TCGPlayerProductURL(productId int, printing, affiliate, condition, language string) string {
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
			language = "Portugese"
		case "Chinese Simplified":
			language = "Chinese (S)"
		case "Chinese Traditional":
			language = "Chinese (T)"
		}
		v.Set("Language", language)
	} else {
		v.Set("Language", "all")
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

func LoadTCGSKUs(reader io.Reader) (mtgjson.AllTCGSkus, error) {
	return mtgjson.LoadAllTCGSkus(bzip2.NewReader(reader))
}

const (
	SYPCSVURL = "https://store.tcgplayer.com/admin/direct/ExportSYPList?categoryid=1&setNameId=All&conditionId=All"
)

type TCGSYP struct {
	SkuId       int
	MarketPrice float64
	MaxQty      int
}

func LoadSyp(auth string) ([]TCGSYP, error) {
	req, err := http.NewRequest(http.MethodGet, SYPCSVURL, nil)
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

	var result []TCGSYP
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			continue
		}

		if len(record) < 8 {
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
