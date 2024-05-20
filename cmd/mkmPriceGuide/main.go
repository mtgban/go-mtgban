package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/mtgban/go-mtgban/cardmarket"
)

func run() int {
	appToken := flag.String("token", "", "App Token")
	appSecret := flag.String("secret", "", "App Secret")
	flag.Parse()

	if *appToken == "" {
		*appToken = os.Getenv("MKM_TOKEN")
	}
	if *appSecret == "" {
		*appSecret = os.Getenv("MKM_SECRET")
	}

	if *appToken == "" || *appSecret == "" {
		fmt.Fprintln(os.Stderr, "Missing token or secret parameter")
		return 1
	}

	client := cardmarket.NewMKMClient(*appToken, *appSecret)
	priceGuide, err := client.MKMPriceGuide()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	err = json.NewEncoder(os.Stdout).Encode(&priceGuide)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	return 0
}

func main() {
	os.Exit(run())
}
