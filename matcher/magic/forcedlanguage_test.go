package magic

import (
	"strings"
	"testing"
)

// TestForcedLanguageImage pins that a foreign-only set's cards show and name
// the printing in the language they are sold as, that a card never printed in
// it keeps its own, and that a language set's cards are their own printing.
func TestForcedLanguageImage(t *testing.T) {
	realDatastore(t)
	for _, tt := range []struct {
		uuid, language, scryfallID, name string
	}{
		{"396d83ed-f449-56f0-887a-08b54ad55b07", "Italian", "b2c2da8a-b50f-478b-9ead-f8a2e740ef3d", "Animare i Muri"},
		{"3b9c696b-8457-595b-a991-b0545c49b293", "Japanese", "82eff756-e0b7-4d86-a156-f57be50e2a17", "純白の秘薬"},
		{"d44cd8fc-9f0c-533b-a2d2-83b30b2130a0", "German", "", "Wald"},
		// Badlands in FBB's French set, and the Italian one FBB sells
		{"77f91bd3-4a2b-55f1-a9e9-7359771d0582", "French", "2fb67912-3d3d-40d4-8aa0-141a8025b037", "Désolation"},
		{"e8cb1f71-56a7-5cbf-8f93-99cf78386545", "Italian", "9c263e5f-a213-4a69-8058-36d3f6be3f1d", "Malaterra"},
	} {
		co, err := testBackend.GetUUID(tt.uuid)
		if err != nil {
			t.Fatal(err)
		}
		if co.Language != tt.language {
			t.Errorf("%s %s#%s is %s, want %s", tt.uuid, co.SetCode, co.Number, co.Language, tt.language)
		}
		if tt.scryfallID != "" && !strings.Contains(co.Images["full"], tt.scryfallID) {
			t.Errorf("%s %s#%s shows %s, want the %s printing %s", tt.uuid, co.SetCode, co.Number, co.Images["full"], tt.language, tt.scryfallID)
		}
		if co.PrintedName != tt.name || co.FlavorName != tt.name {
			t.Errorf("%s %s#%s is named %q/%q, want %q", tt.uuid, co.SetCode, co.Number, co.PrintedName, co.FlavorName, tt.name)
		}
	}
}
