package magic

import (
	"strings"
	"testing"
)

// TestForcedLanguageImage pins that a foreign-only set's cards show the
// printing in the language they are sold as, and that a card never printed in
// it keeps its own.
func TestForcedLanguageImage(t *testing.T) {
	realDatastore(t)
	for _, tt := range []struct {
		uuid, language, scryfallID string
	}{
		{"396d83ed-f449-56f0-887a-08b54ad55b07", "Italian", "b2c2da8a-b50f-478b-9ead-f8a2e740ef3d"},
		{"3b9c696b-8457-595b-a991-b0545c49b293", "Japanese", "82eff756-e0b7-4d86-a156-f57be50e2a17"},
		{"d44cd8fc-9f0c-533b-a2d2-83b30b2130a0", "German", ""},
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
	}
}
