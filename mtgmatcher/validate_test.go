package mtgmatcher

import (
	"testing"
)

func TestValidateID(t *testing.T) {
	realDatastore(t)
	id := ConvertID(IDSpaceScryfall, "8916e24f-9c74-4b6c-9894-d60669854f35")
	original := InputCard{ID: id, Name: "Counterspell", Language: "English", Finish: FinishFoil}
	for _, tt := range []struct {
		name   string
		change func(*InputCard)
		valid  bool
	}{
		{"valid", func(*InputCard) {}, true},
		{"wrong name", func(in *InputCard) { in.Name = "Lightning Bolt" }, false},
		{"missing name", func(in *InputCard) { in.Name = "" }, false},
		{"language code", func(in *InputCard) { in.Language = "en" }, true},
		{"unknown language wording", func(in *InputCard) { in.Language = "English language" }, false},
		{"language hint conflicts", func(in *InputCard) { in.Variation = "Japanese" }, false},
		{"wrong language", func(in *InputCard) { in.Language = "Italian" }, false},
		{"unprinted finish", func(in *InputCard) { in.Finish = FinishEtched }, false},
		{"unknown identifier", func(in *InputCard) { in.ID = "not-a-uuid" }, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			in := original
			tt.change(&in)
			before := in
			got, err := ValidateID(in, IDValidationOptions{})
			if (err == nil) != tt.valid {
				t.Fatalf("got %s %v, valid=%t", got, err, tt.valid)
			}
			if in != before {
				t.Fatal("input mutated")
			}
		})
	}
}

func TestValidatePrintingRejectsUnknownRules(t *testing.T) {
	realDatastore(t)
	b := GlobalDatastore()
	b.rules = nil
	id := ConvertID(IDSpaceScryfall, "8916e24f-9c74-4b6c-9894-d60669854f35")
	if b.ValidatePrinting(InputCard{Name: "Counterspell"}, id) {
		t.Fatal("unsupported game accepted descriptors")
	}
}

func TestNormalizeListingLanguage(t *testing.T) {
	for _, tt := range []struct{ input, want string }{
		{"fr", "French"},
		{"Italian language", "Italian"},
		{"jp", "Japanese"},
		{"zhs", "Chinese Simplified"},
		{"en", ""},
	} {
		in := InputCard{Language: tt.input}
		in.normalizeLanguage()
		if in.Language != tt.want {
			t.Errorf("%q: got %q, want %q", tt.input, in.Language, tt.want)
		}
	}
}
