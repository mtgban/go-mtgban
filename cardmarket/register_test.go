package cardmarket

import (
	"reflect"
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgban"
)

// secretsAsked records every secret a constructor asks for.
type secretsAsked []string

func (s *secretsAsked) Secret(name string) (string, error) {
	*s = append(*s, name)
	return "set", nil
}

// TestIndexMakesNoAuthenticatedCall pins a golden: Index prices from the
// published id map and the public price guide alone. Offered every secret,
// it asks for none and holds no client to spend one with.
func TestIndexMakesNoAuthenticatedCall(t *testing.T) {
	b := datastoreBackend(t, "lorcana", foilOnlyDatastore)
	var asked secretsAsked
	_, err := mtgban.NewScraper(b, "cardmarket", mtgban.WithAuthenticator(&asked), WithCatalog(&cm.Catalog{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(asked) > 0 {
		t.Errorf("Index asked for %v", asked)
	}
	if path := clientPath(reflect.TypeOf(Index{}), "Index", map[reflect.Type]bool{}); path != "" {
		t.Errorf("Index holds a Cardmarket client at %s", path)
	}
}

// clientPath names the first field under t that holds a Cardmarket client.
func clientPath(t reflect.Type, path string, seen map[reflect.Type]bool) string {
	for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice || t.Kind() == reflect.Array || t.Kind() == reflect.Map {
		t = t.Elem()
	}
	if t == reflect.TypeOf(cm.Client{}) {
		return path
	}
	if t.Kind() != reflect.Struct || seen[t] {
		return ""
	}
	seen[t] = true
	for i := range t.NumField() {
		field := t.Field(i)
		if found := clientPath(field.Type, path+"."+field.Name, seen); found != "" {
			return found
		}
	}
	return ""
}
