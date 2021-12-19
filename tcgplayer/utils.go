package tcgplayer

import (
	"fmt"
	"net/url"
)

const baseProductURL = "https://www.tcgplayer.com/product/"

func TCGPlayerProductURL(productId int, printing string, affiliate string) string {
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
	u.RawQuery = v.Encode()

	return u.String()
}
