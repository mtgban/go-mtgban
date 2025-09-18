package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
	"sort"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func getListForDeck(setCode, deckName string) ([]string, error) {
	var list []string

	set, err := mtgmatcher.GetSet(setCode)
	if err != nil {
		return nil, err
	}

	for _, deck := range set.Decks {
		if deck.Name != deckName {
			continue
		}

		for _, board := range [][]mtgmatcher.DeckCard{deck.Commander, deck.MainBoard, deck.SideBoard} {
			for _, card := range board {
				uuid, err := mtgmatcher.MatchId(card.UUID, card.IsFoil)
				if err != nil {
					continue
				}

				list = append(list, uuid)
			}
		}
	}

	return list, nil
}

func getListForSealed(setCode, sealedUUID string) ([]string, error) {
	var list []string

	set, err := mtgmatcher.GetSet(setCode)
	if err != nil {
		return nil, err
	}

	for _, product := range set.SealedProduct {
		if sealedUUID != product.UUID {
			continue
		}

		for key, contents := range product.Contents {
			for _, content := range contents {
				switch key {
				case "card":
					uuid, err := mtgmatcher.MatchId(content.UUID, content.Foil)
					if err != nil {
						return nil, err
					}
					list = append(list, uuid)

				case "pack":
					boosterList, err := mtgmatcher.BoosterGen(content.Set, content.Code)
					if err != nil {
						return nil, err
					}
					list = append(list, boosterList...)

				case "sealed":
					sealedList, err := getListForSealed(content.Set, content.UUID)
					if err != nil {
						return nil, err
					}
					list = append(list, sealedList...)

				case "deck":
					deckList, err := getListForDeck(content.Set, content.Name)
					if err != nil {
						return nil, err
					}
					list = append(list, deckList...)

				case "variable":
					for _, config := range content.Configs {
						for _, card := range config["card"] {
							uuid, err := mtgmatcher.MatchId(card.UUID, card.Foil)
							if err != nil {
								return nil, err
							}
							list = append(list, uuid)
						}
						for _, pack := range config["pack"] {
							boosterList, err := mtgmatcher.BoosterGen(pack.Set, pack.Code)
							if err != nil {
								return nil, err
							}
							list = append(list, boosterList...)
						}
						for _, sealed := range config["sealed"] {
							sealedList, err := getListForSealed(sealed.Set, sealed.UUID)
							if err != nil {
								return nil, err
							}
							list = append(list, sealedList...)
						}
						for _, deck := range config["deck"] {
							deckList, err := getListForDeck(deck.Set, deck.Name)
							if err != nil {
								return nil, err
							}
							list = append(list, deckList...)
						}
					}
				}
			}
		}
	}

	if len(list) == 0 {
		return nil, errors.New("nothing was picked")
	}

	return list, nil
}

func main() {
	SetCodeOpt = flag.String("s", "", "Set code to choose")
	SealedMode = flag.Bool("b", false, "List sealed without unpacking it")
	Reverse = flag.Bool("r", false, "Reverse results tree")
	allprintingsPath := flag.String("a", "allprintings5.json", "Load AllPrintings file path")

	flag.Parse()

	if *SetCodeOpt == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	envAllprintings := os.Getenv("ALLPRINTINGS5_PATH")
	if envAllprintings != "" {
		allprintingsPath = &envAllprintings
	}

	err := mtgmatcher.LoadDatastoreFile(*allprintingsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	os.Exit(run())
}

func run() int {
	set, err := mtgmatcher.GetSet(*SetCodeOpt)
	if err != nil {
		fmt.Fprintln(os.Stderr, *SetCodeOpt, "not found")
		return 1
	}
	if set.Booster == nil {
		fmt.Fprintln(os.Stderr, *SetCodeOpt, "does not have booster information")
		return 1
	}

	result := map[string][]string{}

	for _, product := range set.SealedProduct {
		var list []string
		var err error
		if *SealedMode {
			list = mtgmatcher.SealedWithinSealed(set.Code, product.UUID)
		} else {
			list, err = getListForSealed(set.Code, product.UUID)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, product.Name, err)
			continue
		}
		dedup := map[string]int{}
		for _, item := range list {
			dedup[item]++
		}

		for uuid := range dedup {
			co, err := mtgmatcher.GetUUID(uuid)
			if err != nil {
				continue
			}
			name := co.String()
			if co.Sealed {
				name = co.Name
			}
			result[product.Name] = append(result[product.Name], name)
		}
		sort.Strings(result[product.Name])
	}

	if *Reverse {
		tmp := map[string][]string{}

		for _, list := range result {
			for _, item := range list {
				if tmp[item] != nil {
					continue
				}
				for key, sublist := range result {
					if slices.Contains(sublist, item) {
						tmp[item] = append(tmp[item], key)
					}
				}
			}
		}

		result = tmp
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err = enc.Encode(result)
	if err != nil {
		return 1
	}
	return 0
}

var SetCodeOpt *string
var SealedMode *bool
var Reverse *bool
