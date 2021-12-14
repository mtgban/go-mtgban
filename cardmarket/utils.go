package cardmarket

import (
	"compress/gzip"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var filteredExpansions = []string{
	"GnD Cards",
	"Rk post Products",
	"Starcity Games: Creature Collection",
}

type MKMExpansionIdPair struct {
	IdExpansion int
	Name        string
}

func (mkm *MKMClient) ListExpansionIds() ([]MKMExpansionIdPair, error) {
	expansions, err := mkm.MKMExpansions()
	if err != nil {
		return nil, err
	}

	var out []MKMExpansionIdPair
	for _, exp := range expansions {
		skipExpansion := false
		for _, expName := range filteredExpansions {
			if exp.Name == expName {
				skipExpansion = true
				break
			}
		}
		if strings.Contains(exp.Name, "Token") ||
			strings.Contains(exp.Name, "Oversized") ||
			strings.Contains(exp.Name, "Player Cards") {
			skipExpansion = true
		}
		if skipExpansion {
			continue
		}
		out = append(out, MKMExpansionIdPair{
			IdExpansion: exp.IdExpansion,
			Name:        exp.Name,
		})
	}

	return out, nil
}

type MKMPriceGuide map[int]MKMProductPriceGuide

type MKMProductPriceGuide struct {
	IdProduct      int
	AvgSellPrice   float64
	LowPrice       float64
	TrendPrice     float64
	GermanProLow   float64
	SuggestedPrice float64
	FoilSell       float64
	FoilLow        float64
	FoilTrend      float64
	LowPriceEx     float64
	AvgDay1        float64
	AvgDay7        float64
	AvgDay30       float64
	FoilAvgDay1    float64
	FoilAvgDay7    float64
	FoilAvgDay30   float64
}

func (mkm *MKMClient) MKMPriceGuide() (MKMPriceGuide, error) {
	raw, err := mkm.MKMRawPriceGuide()
	if err != nil {
		return nil, err
	}

	d := base64.NewDecoder(base64.StdEncoding, strings.NewReader(raw))
	gzipReader, err := gzip.NewReader(d)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	csvReader := csv.NewReader(gzipReader)
	// "idProduct","Avg. Sell Price","Low Price","Trend Price","German Pro Low","Suggested Price","Foil Sell","Foil Low","Foil Trend","Low Price Ex+","AVG1","AVG7","AVG30","Foil AVG1","Foil AVG7","Foil AVG30",
	first, err := csvReader.Read()
	if err == io.EOF {
		return nil, errors.New("empty csv")
	}
	if err != nil {
		return nil, fmt.Errorf("error reading csv header: %v", err)
	}

	// The CSV has a trailing comma at the end of the header
	csvReader.FieldsPerRecord = len(first) - 1

	out := MKMPriceGuide{}
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		idProduct, _ := strconv.Atoi(record[0])
		avgSellPrice, _ := strconv.ParseFloat(record[1], 64)
		lowPrice, _ := strconv.ParseFloat(record[2], 64)
		trendPrice, _ := strconv.ParseFloat(record[3], 64)
		germanProLow, _ := strconv.ParseFloat(record[4], 64)
		suggestedPrice, _ := strconv.ParseFloat(record[5], 64)
		foilSell, _ := strconv.ParseFloat(record[6], 64)
		foilLow, _ := strconv.ParseFloat(record[7], 64)
		foilTrend, _ := strconv.ParseFloat(record[8], 64)
		lowPriceEx, _ := strconv.ParseFloat(record[9], 64)
		avgDay1, _ := strconv.ParseFloat(record[10], 64)
		avgDay7, _ := strconv.ParseFloat(record[11], 64)
		avgDay30, _ := strconv.ParseFloat(record[12], 64)
		foilAvgDay1, _ := strconv.ParseFloat(record[13], 64)
		foilAvgDay7, _ := strconv.ParseFloat(record[14], 64)
		foilAvgDay30, _ := strconv.ParseFloat(record[15], 64)

		out[idProduct] = MKMProductPriceGuide{
			IdProduct:      idProduct,
			AvgSellPrice:   avgSellPrice,
			LowPrice:       lowPrice,
			TrendPrice:     trendPrice,
			GermanProLow:   germanProLow,
			SuggestedPrice: suggestedPrice,
			FoilSell:       foilSell,
			FoilLow:        foilLow,
			FoilTrend:      foilTrend,
			LowPriceEx:     lowPriceEx,
			AvgDay1:        avgDay1,
			AvgDay7:        avgDay7,
			AvgDay30:       avgDay30,
			FoilAvgDay1:    foilAvgDay1,
			FoilAvgDay7:    foilAvgDay7,
			FoilAvgDay30:   foilAvgDay30,
		}
	}

	return out, nil
}

type MKMProductList map[int]MKMProductListElement

type MKMProductListElement struct {
	IdProduct   int
	Name        string
	CategoryId  int
	Category    string
	ExpansionId int
	MetacardId  int
	DateAdded   string
}

func (mkm *MKMClient) MKMProductList() (MKMProductList, error) {
	raw, err := mkm.MKMRawProductList()
	if err != nil {
		return nil, err
	}

	d := base64.NewDecoder(base64.StdEncoding, strings.NewReader(raw))
	gzipReader, err := gzip.NewReader(d)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	csvReader := csv.NewReader(gzipReader)
	// idProduct,Name,"Category ID","Category","Expansion ID","Metacard ID","Date Added"
	_, err = csvReader.Read()
	if err == io.EOF {
		return nil, errors.New("empty csv")
	}
	if err != nil {
		return nil, fmt.Errorf("error reading csv header: %v", err)
	}

	out := MKMProductList{}
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		idProduct, _ := strconv.Atoi(record[0])
		name := record[1]
		categoryId, _ := strconv.Atoi(record[2])
		category := record[3]
		expansionId, _ := strconv.Atoi(record[4])
		metacardId, _ := strconv.Atoi(record[5])
		dateAdded := record[6]

		out[idProduct] = MKMProductListElement{
			IdProduct:   idProduct,
			Name:        name,
			CategoryId:  categoryId,
			Category:    category,
			ExpansionId: expansionId,
			MetacardId:  metacardId,
			DateAdded:   dateAdded,
		}
	}

	return out, nil
}
