package merlion

import (
	"strings"
	"testing"
)

const fixture = `Product Line,Set Name,Product Name,Title,Number,Rarity,Condition,Buy Price,Buy Quantity,Merlion Buylist Link,TCGplayer Link,Photo URL
Riftbound,Spiritforged,"Ahri, Inquisitive (Signature)",,227*/221,Showcase,Near Mint Foil,4600.00,2,https://www.merlion.gg/buylist/riftbound/543,https://www.tcgplayer.com/product/664913/riftbound-spiritforged-ahri-inquisitive-signature,https://example.invalid/1.jpg
Riftbound,Origins,Gust,,058/298,Common,Near Mint,0.05,4,https://www.merlion.gg/buylist/riftbound/12,https://www.tcgplayer.com/product/652958/riftbound-origins-gust,https://example.invalid/2.jpg
Riftbound,Origins,No Link,,059/298,Common,Near Mint,1.00,1,https://www.merlion.gg/buylist/riftbound/13,,https://example.invalid/3.jpg
Riftbound,Origins,Unpriced,,060/298,Common,Near Mint,,1,https://www.merlion.gg/buylist/riftbound/14,https://www.tcgplayer.com/product/652959/riftbound-origins-unpriced,https://example.invalid/4.jpg
`

func TestParseBuylist(t *testing.T) {
	cards, err := parseBuylist(strings.NewReader(fixture))
	if err != nil {
		t.Fatal(err)
	}

	// The rows without a product id or without a price carry nothing that
	// could resolve or price them, so they never reach the caller.
	if len(cards) != 2 {
		t.Fatalf("got %d cards, want 2: %+v", len(cards), cards)
	}

	foil := cards[0]
	if foil.TCGplayerID != "664913" {
		t.Errorf("TCGplayerID = %q, want 664913", foil.TCGplayerID)
	}
	if !foil.Foil {
		t.Error("Near Mint Foil did not read as foil")
	}
	if foil.Condition != "Near Mint Foil" {
		t.Errorf("Condition = %q, want the column verbatim", foil.Condition)
	}
	if foil.BuyPrice != 4600 {
		t.Errorf("BuyPrice = %v, want 4600", foil.BuyPrice)
	}
	if foil.Quantity != 2 {
		t.Errorf("Quantity = %v, want 2", foil.Quantity)
	}
	if foil.URL != "https://www.merlion.gg/buylist/riftbound/543" {
		t.Errorf("URL = %q, want the Merlion link rather than the TCGplayer one", foil.URL)
	}

	plain := cards[1]
	if plain.Foil {
		t.Error("Near Mint read as foil")
	}
	if plain.TCGplayerID != "652958" {
		t.Errorf("TCGplayerID = %q, want 652958", plain.TCGplayerID)
	}
}

// The header is the contract with the feed; a column vanishing should stop
// the run rather than quietly produce an empty buylist.
func TestParseBuylistRejectsAMissingColumn(t *testing.T) {
	_, err := parseBuylist(strings.NewReader("Product Line,Set Name,Product Name\nRiftbound,Origins,Gust\n"))
	if err == nil {
		t.Fatal("expected an error for a header missing the priced columns")
	}
	if !strings.Contains(err.Error(), "Condition") {
		t.Errorf("error should name the missing column, got: %v", err)
	}
}
