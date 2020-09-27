package tcgplayer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

type skuType struct {
	SkuId int
	Foil  bool
	Cond  int
}

func (tcg *TCGPlayerMarket) processBL(channel chan<- responseChan, req requestChan) error {
	resp, err := tcg.client.Get(fmt.Sprintf(tcgApiSKUURL, req.TCGProductId))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var skuResponse struct {
		Success bool     `json:"success"`
		Errors  []string `json:"errors"`
		Results []struct {
			SkuId int `json:"skuId"`
			//ProductId   int `json:"productId"`
			LanguageId  int `json:"languageId"`
			PrintingId  int `json:"printingId"`
			ConditionId int `json:"conditionId"`
		} `json:"results"`
	}
	err = json.Unmarshal(data, &skuResponse)
	if err != nil {
		return err
	}
	if !skuResponse.Success {
		return errors.New(strings.Join(skuResponse.Errors, "|"))
	}

	co, err := mtgmatcher.GetUUID(req.UUID)
	if err != nil {
		return err
	}

	allSkus := []skuType{}
	for _, result := range skuResponse.Results {
		if result.LanguageId != 1 {
			continue
		}

		// Untangle foiling status from single id (ie Unhinged, 10E etc)
		if result.PrintingId == 1 && !co.Card.HasNonFoil {
			continue
		} else if result.PrintingId == 2 && !co.Card.HasFoil {
			continue
		}

		s := skuType{
			SkuId: result.SkuId,
			Foil:  result.PrintingId == 2,
			Cond:  result.ConditionId,
		}
		allSkus = append(allSkus, s)
	}

	respBL, err := tcg.client.Get(tcgApiBuylistURL + req.TCGProductId)
	if err != nil {
		return err
	}
	defer respBL.Body.Close()

	data, err = ioutil.ReadAll(respBL.Body)
	if err != nil {
		return err
	}

	var responseBL struct {
		Success bool     `json:"success"`
		Errors  []string `json:"errors"`
		Results []struct {
			//ProductId   int `json:"productId"`
			Prices struct {
				High   float64 `json:"high"`
				Market float64 `json:"market"`
			} `json:"prices"`
			SKUs []struct {
				SkuId  int `json:"skuId"`
				Prices struct {
					High   float64 `json:"high"`
					Market float64 `json:"market"`
				} `json:"prices"`
			} `json:"skus"`
		} `json:"results"`
	}
	err = json.Unmarshal(data, &responseBL)
	if err != nil {
		return err
	}
	if !responseBL.Success {
		return errors.New(strings.Join(responseBL.Errors, "|"))
	}
	if len(responseBL.Results) < 1 {
		return errors.New("empty buylist response")
	}

	result := responseBL.Results[0]

	for _, sku := range result.SKUs {
		var theSku skuType
		for _, target := range allSkus {
			if sku.SkuId == target.SkuId {
				theSku = target
				break
			}
		}
		if theSku.SkuId == 0 {
			continue
		}

		price := sku.Prices.High
		if price == 0 {
			price = sku.Prices.Market
			if price == 0 {
				continue
			}
		}

		theCard := &mtgmatcher.Card{
			Id:   req.UUID,
			Foil: theSku.Foil,
		}
		cardId, err := mtgmatcher.Match(theCard)
		if err != nil {
			return err
		}

		cond := ""
		switch theSku.Cond {
		case 1:
			cond = "NM"
		case 2:
			cond = "SP"
		case 3:
			cond = "MP"
		case 4:
			cond = "HP"
		case 5:
			cond = "PO"
		default:
			tcg.printf("unknown condition %d for %d", theSku.Cond, sku.SkuId)
		}

		var sellPrice, priceRatio float64

		invCards := tcg.inventory[cardId]
		for _, invCard := range invCards {
			if invCard.SellerName != "TCG Market" {
				continue
			}
			sellPrice = invCard.Price
		}

		if sellPrice > 0 {
			priceRatio = price / sellPrice * 100
		}
		out := responseChan{
			cardId: cardId,
			bl: mtgban.BuylistEntry{
				Conditions: cond,
				BuyPrice:   price,
				TradePrice: price,
				Quantity:   0,
				PriceRatio: priceRatio,
				URL:        "https://store.tcgplayer.com/buylist",
			},
		}

		channel <- out
	}

	return nil
}

func (tcg *TCGPlayerMarket) scrpeBL() error {
	pages := make(chan requestChan)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < tcg.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				err := tcg.processBL(channel, page)
				if err != nil {
					tcg.printf("%s", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		sets := mtgmatcher.GetSets()
		i := 1
		for _, set := range sets {
			tcg.printf("Scraping %s (%d/%d)", set.Name, i, len(sets))
			i++

			setDate, _ := time.Parse("2006-01-02", set.ReleaseDate)
			if setDate.After(time.Now()) {
				continue
			}

			for _, card := range set.Cards {
				tcgId, found := card.Identifiers["tcgplayerProductId"]
				if !found {
					continue
				}

				pages <- requestChan{
					TCGProductId: tcgId,
					UUID:         card.UUID,
				}
			}
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	for result := range channel {
		err := tcg.buylist.Add(result.cardId, &result.bl)
		if err != nil {
			tcg.printf(err.Error())
			continue
		}
	}

	tcg.buylistDate = time.Now()

	return nil
}

func (tcg *TCGPlayerMarket) Buylist() (mtgban.BuylistRecord, error) {
	if len(tcg.buylist) > 0 {
		return tcg.buylist, nil
	}

	err := tcg.scrpeBL()
	if err != nil {
		return nil, err
	}

	return tcg.buylist, nil
}
