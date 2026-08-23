// Command tcgid4scryfall walks Scryfall's catalog and reports where its
// TCGplayer ids disagree with the ones mtgmatcher resolves.
package main

import (
	"context"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
	"github.com/mtgban/go-mtgban/tcgplayer"
	api "github.com/mtgban/go-tcgplayer"
)

// Editions maps a TCGplayer group id to its name, read once and reused for
// every card checked.
var Editions map[int]string

// The command's flags.
var (
	VerboseOpt     *bool
	StepOpt        *int
	StepSizeOpt    *int
	StepStartOpt   *int
	ConcurrencyOpt *int

	AllPrintingsOpt *string
)

type responseChan struct {
	cardID string
	entry  mtgban.InventoryEntry
}

func processCards(ctx context.Context, client *api.Client, channel chan<- responseChan, page int) error {
	products, err := client.ListAllProducts(ctx, api.CategoryMagic, []string{"Cards"}, false, page)
	if err != nil {
		return err
	}

	for _, product := range products {
		theCard, err := tcgplayer.Preprocess(&product, Editions)
		if err != nil {
			continue
		}

		cardID, err := mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			continue
		}
		if err != nil {
			// Skip known broken cards
			switch theCard.Name {
			case "Sorcerous Spyglass", //page 44
				"Heroic Intervention": //page 47
				continue
			}
			if !*VerboseOpt &&
				(strings.HasPrefix(theCard.Edition, "Promo Pack") ||
					mtgmatcher.IsBasicLand(theCard.Name) ||
					strings.Contains(strings.ToLower(theCard.Variation), "serial") ||
					strings.Contains(theCard.Variation, "Thick Stock") ||
					theCard.Edition == "Secret Lair Drop" ||
					theCard.Edition == "Modern Horizons 3 Commander" ||
					theCard.Edition == "Prerelease Cards" ||
					theCard.Edition == "The List Reprints") {
				continue
			}

			fmt.Fprintln(os.Stderr, "error on page", page, "-", err)
			fmt.Fprintln(os.Stderr, theCard)
			fmt.Fprintln(os.Stderr, product)
			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					fmt.Fprintln(os.Stderr, "-", card)
				}
			}
			continue
		}

		_, variant := tcgplayer.GetProductNameAndVariant(&product)
		customFields := map[string]string{
			"number":  tcgplayer.GetProductNumber(&product),
			"variant": variant,
			"theCard": theCard.String(),
			"page":    fmt.Sprint(page),
		}

		out := responseChan{
			cardID: cardID,
			entry: mtgban.InventoryEntry{
				Conditions:   "NM",
				Price:        1,
				Quantity:     1,
				SellerName:   "tcg",
				OriginalID:   fmt.Sprint(product.ProductID),
				InstanceID:   fmt.Sprint(page),
				CustomFields: customFields,
			},
		}

		channel <- out
	}
	return nil
}

// Properties is what Scryfall publishes for a card, of which only the
// TCGplayer ids are compared here.
type Properties struct {
	Name       string
	Edition    string
	Number     string
	ScryfallID string

	OldTcgID       string
	NewTcgID       string
	NewEtchedTcgID string
}

func run() int {
	pubKey := os.Getenv("TCGPLAYER_PUBLIC_KEY")
	priKey := os.Getenv("TCGPLAYER_PRIVATE_KEY")

	client, err := api.NewClient(pubKey, priKey)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	allprintingsPath := *AllPrintingsOpt
	envAllprintings := os.Getenv("ALLPRINTINGS5_PATH")
	if envAllprintings != "" {
		allprintingsPath = envAllprintings
	}
	// Load static data once
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
	mtgmatcher.SetGlobalDatastore(ds)

	ctx := context.Background()
	editions, err := tcgplayer.EditionMap(ctx, client, api.CategoryMagic)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	Editions = map[int]string{}
	for groupID, group := range editions {
		Editions[groupID] = group.Name
	}

	start := *StepStartOpt + *StepSizeOpt*(*StepOpt-1)
	end := *StepStartOpt + *StepSizeOpt*(*StepOpt)
	if *StepOpt == 0 {
		totals, err := client.TotalProducts(ctx, api.CategoryMagic, []string{"Cards"})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Fprintln(os.Stderr, "Found", totals, "products")
		start = *StepStartOpt
		end = totals
	}

	pages := make(chan int)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < *ConcurrencyOpt; i++ {
		wg.Go(func() {
			for page := range pages {
				err := processCards(ctx, client, channel, page)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
				}
			}
		})
	}

	go func() {
		for i := start; i < end; i += api.MaxItemsInResponse {
			pages <- i
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	inventory := mtgban.InventoryRecord{}
	for result := range channel {
		err := inventory.AddStrict(result.cardID, &result.entry)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
	}

	fmt.Fprintln(os.Stderr, "Found", len(inventory), "mtgjson hashes")
	firstPage := 0

	// Reduce the map to the needed ids
	output := map[string]*Properties{}
	for uuid, cards := range inventory {
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil {
			fmt.Fprintln(os.Stderr, err, uuid)
			continue
		}
		switch co.Name {
		case "Bruna, Light of Alabaster":
			if co.Edition == "Open the Helvault" {
				continue
			}
		}

		newTcgID := cards[0].OriginalID
		newEtchedTcgID := ""
		oldTcgID := co.Identifiers["tcgplayerProductId"]

		// If etched, there is always going to be a separate id,
		// but the same is not guaranteed for every foil card
		if co.Etched {
			newEtchedTcgID = newTcgID
			newTcgID = ""
			oldTcgID = co.Identifiers["tcgplayerEtchedProductId"]
		}

		identifier := co.Identifiers["scryfallId"]
		if (newTcgID != "" && oldTcgID != newTcgID) ||
			(newEtchedTcgID != "" && oldTcgID != newEtchedTcgID) {
			_, found := output[identifier]
			if !found {
				output[identifier] = &Properties{}
			}
			output[identifier].Name = co.Name
			output[identifier].Edition = co.Edition
			output[identifier].Number = co.Number
			output[identifier].ScryfallID = identifier

			output[identifier].OldTcgID = oldTcgID
			if co.Etched {
				output[identifier].NewEtchedTcgID = newEtchedTcgID
			} else {
				output[identifier].NewTcgID = newTcgID
			}

			// Set the first page for validation
			page, _ := strconv.Atoi(cards[0].InstanceID)
			if firstPage == 0 || firstPage > page {
				firstPage = page
			}
		}
	}

	csvWriter := csv.NewWriter(os.Stdout)
	csvWriter.Write([]string{
		"name", "set", "cn", "scryfall_id", "old_tcgplayer_id", "new_tcgplayer_id", "new_tcgplayer_etched_id",
	})
	fixes := 0
	for _, props := range output {
		fixes++
		csvWriter.Write([]string{
			props.Name,
			props.Edition,
			props.Number,
			props.ScryfallID,
			props.OldTcgID,
			props.NewTcgID,
			props.NewEtchedTcgID,
		})
		csvWriter.Flush()
	}
	fmt.Fprintln(os.Stderr, "Fixed", fixes, "ids")
	if firstPage != 0 {
		fmt.Fprintln(os.Stderr, "The first page with problematic ids is:", firstPage)
	}
	return 0
}

func main() {
	VerboseOpt = flag.Bool("verbose", false, "Skip errors from sets that might be too new")
	StepOpt = flag.Int("step", 0, "How many ranges should be processed")
	StepSizeOpt = flag.Int("step-size", 1000, "Size of the range")
	StepStartOpt = flag.Int("step-start", 0, "Start offset of the range")
	ConcurrencyOpt = flag.Int("threads", 8, "How many threads to use")
	AllPrintingsOpt = flag.String("i", "allprintings5.json", "AllPrintings file path")
	flag.Parse()

	os.Exit(run())
}
