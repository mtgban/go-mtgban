package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/mtgban/go-mtgban/cardmarket"
)

func run() int {
	mode := flag.String("mode", "prices", "Which file to download [prices]/singles/sealed")

	flag.Parse()

	var output interface{}
	var err error

	switch *mode {
	default:
		output, err = cardmarket.GetPriceGuide()
	case "singles":
		output, err = cardmarket.GetProductListSingles()
	case "sealed":
		output, err = cardmarket.GetProductListSealed()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	err = json.NewEncoder(os.Stdout).Encode(output)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	return 0
}

func main() {
	os.Exit(run())
}
