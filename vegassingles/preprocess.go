package vegassingles

import (
	"strconv"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func preprocess(product VSProduct) (*mtgmatcher.InputCard, error) {
	// Display name format: "Hallowed Fountain (RVR-280) - Ravnica Remastered"
	// Extract card name by finding the first " ("
	cardName := product.DisplayName
	if idx := strings.Index(cardName, " ("); idx != -1 {
		cardName = cardName[:idx]
	}

	edition := product.ProductData.Set
	if edition == "" {
		edition = product.ProductData.SetName
	}

	variant := ""
	if product.ProductData.CollectorNumberNormalized > 0 {
		variant = strconv.Itoa(product.ProductData.CollectorNumberNormalized)
	}

	foil := product.SelectedFinish == "foil"

	return &mtgmatcher.InputCard{
		Name:      cardName,
		Edition:   edition,
		Variation: variant,
		Foil:      foil,
	}, nil
}
