// Command boosterGen opens boosters, drawing from a set's sheets the way
// the real product does, and prints what came out.
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/mroth/weightedrand/v2"

	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// The command's flags, and the CSV writer the -csv flag turns on.
var (
	SetCodeOpt       *string
	NumberOfBoosters *int
	BoosterTypeOpt   *string
	AllPrintingsOpt  *string
	ColorOpt         *string

	CSVOutput *bool
	CSVWriter *csv.Writer
)

// Pick is one card drawn from a booster, with the sheet it came off.
type Pick struct {
	CardID string
	Sheet  string
	Finish string
}

func run() int {
	allprintingsPath := *AllPrintingsOpt
	envAllprintings := os.Getenv("ALLPRINTINGS5_PATH")
	if envAllprintings != "" {
		allprintingsPath = envAllprintings
	}

	allPrintingsReader, err := os.Open(allprintingsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer allPrintingsReader.Close()
	ds, err := magic.Load(allPrintingsReader)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	set, err := ds.GetSet(*SetCodeOpt)
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
		var choices []weightedrand.Choice[map[string]int, int]
		for _, booster := range set.Booster[*BoosterTypeOpt].Boosters {
			choices = append(choices, weightedrand.NewChoice(booster.Contents, booster.Weight))
		}
		sheetChooser, err := weightedrand.NewChooser(choices...)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		contents := sheetChooser.Pick()
		fmt.Fprintf(os.Stderr, "%v\n", contents)

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
				for cardID, frequency := range sheet.Cards {
					for range frequency {
						picks = append(picks, Pick{
							CardID: cardID,
							Sheet:  sheetName,
							Finish: finish,
						})
					}
				}
			} else {
				var duplicated map[string]bool
				var balanced map[string]bool

				// Prepare maps to keep track of duplicates and balanced colors if necessary
				if !sheet.AllowDuplicates {
					duplicated = map[string]bool{}
				}
				if sheet.BalanceColors {
					balanced = map[string]bool{}
				}

				// Move sheet data into weightedrand choices
				var cardChoices []weightedrand.Choice[string, int]
				for cardID, weight := range sheet.Cards {
					cardChoices = append(cardChoices, weightedrand.NewChoice(cardID, weight))
				}
				cardChooser, err := weightedrand.NewChooser(cardChoices...)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					return 1
				}

				// Pick a card uuid as many times as defined by its frequency
				// Note that it's ok to pick the same card from the same sheet multiple times
				for j := 0; j < frequency; j++ {
					item := cardChooser.Pick()
					// Validate card exists (ie in case of online-only printing)
					co, err := ds.GetUUID(item)
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
						CardID: item,
						Sheet:  sheetName,
						Finish: finish,
					})
				}
			}
		}

		sort.Slice(picks, func(i, j int) bool {
			if picks[i].Sheet == picks[j].Sheet {
				return picks[i].CardID < picks[j].CardID
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
			id, _ := ds.MatchID(pick.CardID, pick.Finish == "foil", pick.Finish == "etched")
			co, _ := ds.GetUUID(id)
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
