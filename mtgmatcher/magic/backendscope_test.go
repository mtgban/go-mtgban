package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// The identification hooks used to answer their auxiliary lookups from
// whichever datastore was installed globally, so a Backend opened on the
// side was matched against with another one's data. They take the backend
// now, and these pin that they read it: the same input answered against an
// empty backend must find nothing, where before it would have found
// whatever the global happened to hold.
func TestHelpersReadTheGivenBackend(t *testing.T) {
	// A name printed exactly once in Secret Lair Ultimate, which is what
	// the tag check asks the datastore about.
	var name string
	for _, card := range testBackend.Sets["SLU"].Cards {
		if len(testBackend.MatchInSet(card.Name, "SLU")) == 1 {
			name = card.Name
			break
		}
	}
	if name == "" {
		t.Skip("no uniquely printed SLU card in this datastore")
	}

	// The wording says Secret Lair; which drop it belongs to is what the
	// datastore is asked.
	inCard := &mtgmatcher.InputCard{Name: name, Edition: "Secret Lair Drop"}
	if !hasSecretLairTag(testBackend, inCard, "SLU") {
		t.Errorf("hasSecretLairTag(%q) = false against the datastore that holds it", name)
	}
	if hasSecretLairTag(&mtgmatcher.Backend{}, inCard, "SLU") {
		t.Errorf("hasSecretLairTag(%q) = true against an empty backend, so it read another", name)
	}
}

// The same for the promo tag closures, which reach the datastore to forgive
// a listing that names a treatment its printing carries untagged.
func TestPromoTagFuncsReadTheGivenBackend(t *testing.T) {
	var tagFunc func(*mtgmatcher.Backend, *mtgmatcher.InputCard) bool
	for _, element := range promoTypeElements {
		if element.PromoType == PromoTypeGalaxyFoil {
			tagFunc = element.TagFunc
			break
		}
	}
	if tagFunc == nil {
		t.Fatal("no galaxy foil promo element carrying a tag function")
	}

	// A Secret Lair foil says nothing about the galaxy treatment, so what
	// the drop was printed in is what the datastore is asked. Command Tower
	// is the one name the closure answers by number instead.
	var name string
	for _, uuid := range testBackend.AllUUIDs {
		co, err := testBackend.GetUUID(uuid)
		if err != nil || co.Sealed || co.Name == "Command Tower" {
			continue
		}
		if co.SetCode == "SLD" && co.HasPromoType(PromoTypeGalaxyFoil) {
			name = co.Name
			break
		}
	}
	if name == "" {
		t.Skip("no galaxy foil printing in this datastore")
	}

	inCard := &mtgmatcher.InputCard{Name: name, Edition: "Secret Lair Drop", Foil: true}
	if !tagFunc(testBackend, inCard) {
		t.Errorf("galaxy foil tag for %q = false against the datastore that holds it", name)
	}
	if tagFunc(&mtgmatcher.Backend{}, inCard) {
		t.Errorf("galaxy foil tag for %q = true against an empty backend, so it read another", name)
	}
}
