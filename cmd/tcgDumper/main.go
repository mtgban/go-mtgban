package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"

	"github.com/mtgban/go-mtgban/tcgplayer"
)

const (
	defaultConcurrency = 8
)

func run() int {
	gameOpt := flag.Int("game", 0, "Game id to dump")
	tcgPublicKeyOpt := flag.String("pub", "", "TCGplayer public key")
	tcgPrivateKeyOpt := flag.String("pri", "", "TCGplayer private key")
	flag.Parse()

	pubEnv := os.Getenv("TCGPLAYER_PUBLIC_KEY")
	if pubEnv != "" {
		tcgPublicKeyOpt = &pubEnv
	}
	priEnv := os.Getenv("TCGPLAYER_PRIVATE_KEY")
	if priEnv != "" {
		tcgPrivateKeyOpt = &priEnv
	}

	if *tcgPublicKeyOpt == "" || *tcgPrivateKeyOpt == "" {
		log.Fatalln("Missing TCGplayer keys")
	}

	if *gameOpt == 0 {
		log.Fatalln("Missing Game id")
	}

	tcgClient := tcgplayer.NewTCGClient(*tcgPublicKeyOpt, *tcgPrivateKeyOpt)

	totalgroups, err := tcgClient.TotalGroups(*gameOpt)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	var groups []tcgplayer.TCGGroup
	for i := 0; i < totalgroups; i += tcgplayer.MaxLimit {
		out, err := tcgClient.ListAllGroups(*gameOpt, i, tcgplayer.MaxLimit)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		groups = append(groups, out...)
	}
	fmt.Fprintln(os.Stderr, "Found", len(groups), "groups")

	totalProducts, err := tcgClient.TotalProducts(*gameOpt, tcgplayer.AllProductTypes)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Fprintln(os.Stderr, "Found", totalProducts, "products")

	pages := make(chan int)
	channel := make(chan tcgplayer.TCGProduct)
	var wg sync.WaitGroup

	for i := 0; i < defaultConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				products, err := tcgClient.ListAllProducts(*gameOpt, tcgplayer.AllProductTypes, true, page, tcgplayer.MaxLimit)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					continue
				}
				for _, product := range products {
					channel <- product
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for i := 0; i < totalProducts; i += tcgplayer.MaxLimit {
			pages <- i
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	var products []tcgplayer.TCGProduct
	for result := range channel {
		products = append(products, result)
	}

	sort.Slice(products, func(i, j int) bool {
		return products[i].ProductId < products[j].ProductId
	})

	var output struct {
		Groups   []tcgplayer.TCGGroup   `json:"groups"`
		Products []tcgplayer.TCGProduct `json:"products"`
	}
	output.Products = products
	output.Groups = groups

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err = enc.Encode(output)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Fprintln(os.Stderr, "Dumped", len(products), "products and", len(groups), "groups")

	return 0
}

func main() {
	os.Exit(run())
}
