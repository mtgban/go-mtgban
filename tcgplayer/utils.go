package tcgplayer

import (
	"compress/bzip2"
	"fmt"
	"io"
	"net/url"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/mtgjson"
)

const baseProductURL = "https://www.tcgplayer.com/product/"

func TCGPlayerProductURL(productId int, printing, affiliate, language string) string {
	u, err := url.Parse(baseProductURL + fmt.Sprint(productId))
	if err != nil {
		return ""
	}

	v := u.Query()
	if printing != "" {
		v.Set("Printing", printing)
	}
	if affiliate != "" {
		v.Set("utm_campaign", "affiliate")
		v.Set("utm_medium", affiliate)
		v.Set("utm_source", affiliate)
		v.Set("partner", affiliate)
	}
	if language != "" {
		v.Set("Language", mtgmatcher.Title(language))
	}
	u.RawQuery = v.Encode()

	return u.String()
}

func LoadTCGSKUs(reader io.Reader) (mtgjson.AllTCGSkus, error) {
	return mtgjson.LoadAllTCGSkus(bzip2.NewReader(reader))
}
