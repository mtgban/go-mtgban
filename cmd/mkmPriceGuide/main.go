package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/mtgban/go-mtgban/cardmarket"
)

func run() int {
	mode := flag.String("mode", "prices", "Which file to download [prices]/singles/sealed")
	game := flag.Int("game", 1, "Select which game (default=magic)")

	flag.Parse()

	var output interface{}
	var err error

	ctx := context.Background()
	switch *mode {
	default:
		output, err = cardmarket.GetPriceGuide(ctx, *game)
	case "singles":
		output, err = cardmarket.GetProductListSingles(ctx, *game)
	case "sealed":
		output, err = cardmarket.GetProductListSealed(ctx, *game)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err = enc.Encode(output)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	return 0
}

func main() {
	os.Exit(run())
}
