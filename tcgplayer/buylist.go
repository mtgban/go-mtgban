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
	"github.com/kodabb/go-mtgban/mtgdb"
)

type skuType struct {
	SkuId int
	Foil  bool
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

	allSkus := []skuType{}
	for _, result := range skuResponse.Results {
		if result.ConditionId != 1 || result.LanguageId != 1 {
			continue
		}
		s := skuType{
			SkuId: result.SkuId,
			Foil:  result.PrintingId == 2,
		}
		allSkus = append(allSkus, s)
	}

	respBL, err := tcg.client.Get(tcgApiBuylistURL + fmt.Sprint(req.TCGProductId))
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
				Market float64 `json:"market"`
			} `json:"prices"`
			SKUs []struct {
				SkuId  int `json:"skuId"`
				Prices struct {
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

		price := sku.Prices.Market
		if price == 0 {
			continue
		}

		theCard := &mtgdb.Card{
			Id:   req.UUID,
			Foil: theSku.Foil,
		}
		cc, err := theCard.Match()
		if err != nil {
			return err
		}

		var sellPrice, priceRatio float64

		invCards := tcg.inventory[*cc]
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
			card: *cc,
			bl: mtgban.BuylistEntry{
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
		for _, code := range mtgdb.AllSets() {
			set, _ := mtgdb.Set(code)
			tcg.printf("Scraping %s", set.Name)

			setDate, _ := time.Parse("2006-01-02", set.ReleaseDate)
			if setDate.After(time.Now()) {
				continue
			}

			for _, card := range set.Cards {
				if card.TcgplayerProductId == 0 {
					continue
				}

				pages <- requestChan{
					TCGProductId: card.TcgplayerProductId,
					UUID:         card.UUID,
				}
			}
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	for result := range channel {
		err := tcg.buylist.Add(&result.card, &result.bl)
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
