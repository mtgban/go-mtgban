package mtgdb

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kodabb/go-mtgban/mtgjson"
)

type Database struct {
	Sets  mtgjson.SetDatabase
	Cards mtgjson.CardDatabase
}

var internal *Database

func NewDatabase(setsReader, cardsReader io.Reader) (*Database, error) {
	sets, err := mtgjson.LoadAllPrintingsFromReader(setsReader)
	if err != nil {
		return nil, err
	}
	cards, err := mtgjson.LoadAllCardsFromReader(cardsReader)
	if err != nil {
		return nil, err
	}

	db := Database{
		Sets:  sets,
		Cards: cards,
	}

	return &db, nil
}

func NewDatabaseFromPaths(allprintingsPath, allcardsPath string) (*Database, error) {
	setsReader, err := os.Open(allprintingsPath)
	if err != nil {
		return nil, err
	}
	defer setsReader.Close()

	cardsReader, err := os.Open(allcardsPath)
	if err != nil {
		return nil, err
	}
	defer cardsReader.Close()

	return NewDatabase(setsReader, cardsReader)
}

func RegisterWithReaders(setsReader, cardsReader io.Reader) error {
	db, err := NewDatabase(setsReader, cardsReader)
	if err != nil {
		return err
	}
	internal = db
	return err
}

func RegisterWithPaths(allprintingsPath, allcardsPath string) error {
	db, err := NewDatabaseFromPaths(allprintingsPath, allcardsPath)
	if err != nil {
		return err
	}
	internal = db
	return err
}

func Set(codeName string) (*mtgjson.Set, error) {
	if internal == nil {
		return nil, fmt.Errorf("internal database is not initialized")
	}

	set, found := internal.Sets[codeName]
	if found {
		return set, nil
	}

	for i := range internal.Sets {
		if internal.Sets[i].Name == codeName {
			return internal.Sets[i], nil
		}
	}

	return nil, fmt.Errorf("set %s not found", codeName)
}

func EditionCode2Name(code string) (string, error) {
	if internal == nil {
		return "", fmt.Errorf("internal database is not initialized")
	}
	set, found := internal.Sets[strings.ToUpper(code)]
	if !found {
		return "", fmt.Errorf("edition code '%s' not found", code)
	}
	return set.Name, nil
}

func EditionName2Code(name string) (string, error) {
	if internal == nil {
		return "", fmt.Errorf("internal database is not initialized")
	}
	for key := range internal.Sets {
		if internal.Sets[key].Name == name {
			return key, nil
		}
	}
	return "", fmt.Errorf("edition name '%s' not found", name)
}
