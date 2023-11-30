package tcgplayer

import (
	"compress/bzip2"
	"fmt"
	"io"
	"net/url"

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
