package tcgplayer

import (
	"fmt"
	"net/url"

	"github.com/kodabb/go-mtgban/mtgmatcher"
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
