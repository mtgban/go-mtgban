package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/jmcvetta/randutil"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

var SetCodeOpt *string
var NumberOfBoosters *int
var BoosterTypeOpt *string
var AllPrintingsOpt *string
var ColorOpt *string

var CSVOutput *bool
var CSVWriter *csv.Writer

type Pick struct {
	CardId string
	Sheet  string
	Finish string
}

func run() int {
	allprintingsPath := *AllPrintingsOpt
	envAllprintings := os.Getenv("ALLPRINTINGS5_PATH")
	if envAllprintings != "" {
		allprintingsPath = envAllprintings
	}

	err := mtgmatcher.LoadDatastoreFile(allprintingsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	set, err := mtgmatcher.GetSet(*SetCodeOpt)
	if err != nil {
		fmt.Fprintln(os.Stderr, *SetCodeOpt, "not found")
		return 1
	}
	if set.Booster == nil {
		fmt.Fprintln(os.Stderr, *SetCodeOpt, "does not have booster information")
		return 1
	}
	_, found := set.Booster[*BoosterTypeOpt]
	if !found {
		fmt.Fprintln(os.Stderr, "Booster type", *BoosterTypeOpt, "not found for", *SetCodeOpt)
		return 1
	}

	if *CSVOutput {
		CSVWriter = csv.NewWriter(os.Stdout)
		CSVWriter.Write([]string{"setCode", "number", "name", "isFoil"})
	}

	for i := 0; i < *NumberOfBoosters; i++ {
		// Pick a rarity distribution as defined in Contents at random using their weight
		var choices []randutil.Choice
		for _, booster := range set.Booster[*BoosterTypeOpt].Boosters {
			choices = append(choices, randutil.Choice{
				Weight: booster.Weight,
				Item:   booster.Contents,
			})
		}
		choice, err := randutil.WeightedChoice(choices)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Fprintf(os.Stderr, "%v\n", choice.Item)

		contents := choice.Item.(map[string]int)

		var picks []Pick
		// For each sheet, pick a card at random using the weight
		for sheetName, frequency := range contents {
			// Grab the sheet
			sheet := set.Booster[*BoosterTypeOpt].Sheets[sheetName]

			// Determine foiling status
			finish := "nonfoil"
			if sheet.Foil {
				finish = "foil"
			}
			if strings.Contains(strings.ToLower(sheetName), "etched") {
				finish = "etched"
			}

			if sheet.Fixed {
				// Fixed means there is no randomness, just pick the cards as listed
				for cardId, frequency := range sheet.Cards {
					for j := 0; j < frequency; j++ {
						picks = append(picks, Pick{
							CardId: cardId,
							Sheet:  sheetName,
							Finish: finish,
						})
					}
				}
			} else {
				var duplicated map[string]bool
				var balanced map[string]bool

				// Prepare maps to keep track of duplicates and balaced colors if necessary
				if !sheet.AllowDuplicates {
					duplicated = map[string]bool{}
				}
				if sheet.BalanceColors {
					balanced = map[string]bool{}
				}

				// Move sheet data into randutil data type
				var cardChoices []randutil.Choice
				for cardId, weight := range sheet.Cards {
					cardChoices = append(cardChoices, randutil.Choice{
						Weight: weight,
						Item:   cardId,
					})
				}

				// Pick a card uuid as many times as defined by its frequency
				// Note that it's ok to pick the same card from the same sheet multiple times
				for j := 0; j < frequency; j++ {
					choice, err := randutil.WeightedChoice(cardChoices)
					if err != nil {
						fmt.Fprintln(os.Stderr, err)
						return 1
					}
					item := choice.Item.(string)
					// Validate card exists (ie in case of online-only printing)
					co, err := mtgmatcher.GetUUID(item)
					if err != nil {
						j--
						continue
					}

					// Check if we need to reroll due to BalanceColors
					if sheet.BalanceColors && frequency > 4 && j < 5 {
						// Reroll for the first five cards, the first 5 cards cannot be multicolor or colorless
						if len(co.Colors) != 1 {
							j--
							continue
						}
						// Reroll if one of the single colors was already found
						if balanced[co.Colors[0]] {
							j--
							continue
						}
						// Found!
						balanced[co.Colors[0]] = true
					}

					// Check if the sheet allows duplicates, and, if not, pick again
					// in case the uuid was already picked
					if !sheet.AllowDuplicates {
						if duplicated[item] {
							j--
							continue
						}
						duplicated[item] = true
					}

					picks = append(picks, Pick{
						CardId: item,
						Sheet:  sheetName,
						Finish: finish,
					})
				}
			}
		}

		sort.Slice(picks, func(i, j int) bool {
			if picks[i].Sheet == picks[j].Sheet {
				return picks[i].CardId < picks[j].CardId
			}
			return picks[i].Sheet < picks[j].Sheet
		})

		// Don't clobber CSV output if used
		out := os.Stdout
		if *CSVOutput {
			out = os.Stderr
		}
		w := tabwriter.NewWriter(out, 0, 0, 1, ' ', 0)
		for _, pick := range picks {
			id, _ := mtgmatcher.MatchId(pick.CardId, pick.Finish == "foil", pick.Finish == "etched")
			co, _ := mtgmatcher.GetUUID(id)
			fmt.Fprintf(w, "%s\t%s|%s\n", pick.Sheet, co, co.Rarity)
			if *CSVOutput {
				CSVWriter.Write([]string{co.SetCode, co.Number, co.Name, fmt.Sprint(co.Foil)})
			}
		}
		w.Flush()
		if *CSVOutput {
			CSVWriter.Flush()
		}
	}

	return 0
}

func main() {
	SetCodeOpt = flag.String("s", "", "Set code to choose")
	NumberOfBoosters = flag.Int("n", 1, "Number of boosters to generate")
	BoosterTypeOpt = flag.String("t", "default", "Type of booster to pick (default/set/collector/theme/jumpstart)")
	AllPrintingsOpt = flag.String("a", "allprintings5.json", "Load AllPrintings file path")
	ColorOpt = flag.String("c", "", "One letter color of the theme booster")
	CSVOutput = flag.Bool("csv", false, "Output a csv of the data")

	flag.Parse()

	if *SetCodeOpt == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *BoosterTypeOpt == "theme" {
		if *ColorOpt == "" {
			fmt.Fprintln(os.Stderr, "theme booster needs color information")
			os.Exit(1)
		}
		*BoosterTypeOpt += "-" + *ColorOpt
	}

	os.Exit(run())
}
