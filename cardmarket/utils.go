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

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

var filteredExpansions = []string{
	"73",   // Foreign White Bordered
	"96",   // Rinascimento
	"105",  // Salvat-Hachette
	"110",  // Oversized 6x9 Promos
	"111",  // Oversized Box Toppers
	"1259", // Salvat-Hachette 2011
	"1269", // 2006 Player Cards
	"1281", // Your Move Games Tokens
	"1332", // Fourth Edition: Black Bordered
	"1391", // 2005 Player Cards
	"1392", // 2007 Player Card
	"1401", // Simplified Chinese Alternate Art Cards
	"1408", // Filler Cards
	"1451", // Ultra-Pro Puzzle Cards
	"1500", // Starcity Games: Commemorative Tokens
	"1502", // Tokens for MTG
	"1599", // Chronicles: Japanese
	"1600", // Fourth Edition: Alternate
	"1638", // Fallen Empires: Wyvern Misprints
	"1639", // Oversized 9x12 Promos
	"1659", // Yummy Tokens
	"1691", // GnD Cards
	"1705", // Javi Alterations Tokens
	"1833", // Rk post Products
	"1834", // Mezzocielo & Friends Classic Tokens
	"1837", // The Dark Italian
	"1838", // Legends Italian
	"2114", // Johannes Voss Tokens
	"2484", // Modern Horizons: Art Series
	"2572", // Classic Art Tokens
	"2529", // Cats & Cantrips Tokens
	"2567", // Tokens of Spirit
}

var filteredCards = []string{
	"Build a Deck: The Basics // Popular Magic Formats",

	// TFTH tokens
	"Hydra Head",
	"Ravenous Brute Head",
	"Savage Vigor Head",
	"Shrieking Titan Head",
	"Snapping Fang Head",
	"Disorienting Glower",
	"Distract the Hydra",
	"Grown from the Stump",
	"Hydra's Impenetrable Hide",
	"Noxious Hydra Breath",
	"Neck Tangle",
	"Strike the Weak Spot",
	"Torn Between Heads",
	"Swallow the Hero Whole",
	"Unified Lunge",

	// TBTH tokens
	"Minotaur Goreseeker",
	"Minotaur Younghorn",
	"Mogis's Chosen",
	"Phoberos Reaver",
	"Reckless Minotaur",
	"Altar of Mogis",
	"Consuming Rage",
	"Intervention of Keranos",
	"Descend on the Prey",
	"Plundered Statue",
	"Refreshing Elixir",
	"Touch of the Horned God",
	"Massacre Totem",
	"Unquenchable Fury",
	"Vitality Salve",

	// TDAG tokens
	"Xenagos Ascended",
	"Rollicking Throng",
	"Ecstatic Piper",
	"Maddened Oread",
	"Pheres-Band Revelers",
	"Serpent Dancers",
	"Wild Maenads",
	"Dance of Flame",
	"Dance of Panic",
	"Impulsive Destruction",
	"Impulsive Charge",
	"Impulsive Return",
	"Rip to Pieces",
	"Xenagos's Strike",
	"Xenagos's Scorn",

	// Unique
	"1996 World Champion",
	"Shichifukujin Dragon",
	"Proposal",
	"Magic Guru",
	"Fraternal Exaltation",
	"Robot Chicken",
	"Phoenix Heart",
	"Splendid Genesis",
}

type MKMProductIdPair struct {
	ProductId   string
	ExpansionId string
}

func (mkm *MKMClient) ListProductIds() ([]MKMProductIdPair, error) {
	var output []MKMProductIdPair

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

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		if len(record) < 5 {
			continue
		}

		id := record[0]
		cardName := record[1]
		categoryId := record[2]
		expansionId := record[4]

		// Only "Magic Single" products
		if categoryId != "1" {
			continue
		}

		// Skip unsupported sets
		skipExpansion := false
		for _, expId := range filteredExpansions {
			if expansionId == expId {
				skipExpansion = true
				break
			}
		}
		if skipExpansion {
			continue
		}

		// Skip unsupported cards
		skipCard := false
		for _, name := range filteredCards {
			if cardName == name {
				skipCard = true
				break
			}
		}
		if skipCard ||
			mtgmatcher.IsToken(cardName) ||
			strings.Contains(cardName, "On Your Turn") {
			continue
		}

		output = append(output, MKMProductIdPair{
			ProductId:   id,
			ExpansionId: expansionId,
		})
	}

	return output, nil
}

type MKMExpansionIdPair struct {
	IdExpansion string
	Name        string
}

func (mkm *MKMClient) ListExpansionIds() ([]MKMExpansionIdPair, error) {
	expansions, err := mkm.MKMExpansions()
	if err != nil {
		return nil, err
	}

	var out []MKMExpansionIdPair
	for id, exp := range expansions {
		skipExpansion := false
		for _, expId := range filteredExpansions {
			if id == expId {
				skipExpansion = true
				break
			}
		}
		if skipExpansion || strings.Contains(exp.Name, "Token") {
			continue
		}
		out = append(out, MKMExpansionIdPair{
			IdExpansion: id,
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
