package matcher

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestValidateID(t *testing.T) {
	b := realDatastore(t)
	id := b.ConvertID(IDSpaceScryfall, "8916e24f-9c74-4b6c-9894-d60669854f35")
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
			got, err := b.ValidateID(in, IDValidationOptions{})
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
	orig := realDatastore(t)
	// A shallow copy, so nulling rules below does not disturb the shared
	// backend every other test in this package matches against.
	b := *orig
	b.rules = nil
	id := b.ConvertID(IDSpaceScryfall, "8916e24f-9c74-4b6c-9894-d60669854f35")
	if b.ValidatePrinting(InputCard{Name: "Counterspell"}, id) {
		t.Fatal("unsupported game accepted descriptors")
	}
}

func TestNormalizeListingLanguage(t *testing.T) {
	for _, tt := range []struct{ input, edition, variation, want string }{
		{"fr", "", "", "French"},
		{"Italian language", "", "", "Italian"},
		{"jp", "", "", "Japanese"},
		{"zhs", "", "", "Chinese Simplified"},
		{"en", "", "", ""},
		{"", "European Land Program", "Euro Blue: Germany", ""},
		{"", "Chronicles Japanese", "", "Japanese"},
		{"", "Fourth Edition", "S.Chinese", "Chinese"},
		{"", "Legends", "German", "German"},
	} {
		in := InputCard{Language: tt.input, Edition: tt.edition, Variation: tt.variation}
		in.normalizeLanguage()
		if in.Language != tt.want {
			t.Errorf("%+v: got %q, want %q", tt, in.Language, tt.want)
		}
	}
}

// TestMatchIDLanguageSpelling pins that a language spelled shorter than the
// datastore names the one printing it fits, and none where it fits two or
// names no language at all.
func TestMatchIDLanguageSpelling(t *testing.T) {
	b := &Backend{UUIDs: map[string]*CardObject{}}
	for uuid, data := range map[string]string{
		"base": `{"uuid": "base", "language": "Japanese", "foreignData": [
			{"language": "Portuguese (Brazil)", "uuid": "pt"},
			{"language": "Chinese Simplified", "uuid": "zhs"},
			{"language": "Chinese Traditional", "uuid": "zht"}]}`,
		"lone": `{"uuid": "lone", "language": "English", "foreignData": [
			{"language": "Italian", "uuid": "ita"}]}`,
	} {
		var card Card
		err := json.Unmarshal([]byte(data), &card)
		if err != nil {
			t.Fatal(err)
		}
		b.UUIDs[uuid] = &CardObject{Card: card}
	}
	for _, uuid := range []string{"pt", "zhs", "zht", "ita"} {
		b.UUIDs[uuid] = &CardObject{Card: Card{UUID: uuid}}
	}
	for _, tt := range []struct{ uuid, language, want string }{
		{"base", "Portuguese", "pt"},
		{"base", "Portuguese (Brazil)", "pt"},
		{"base", "Chinese Traditional", "zht"},
		{"base", "Japanese", "base"},
		{"base", "Chinese", ""},
		{"lone", "Italian", "ita"},
		{"lone", "", ""},
	} {
		got, err := b.MatchIDLanguage(tt.uuid, tt.language)
		if tt.want == "" {
			if !errors.Is(err, ErrCardUnknownID) {
				t.Errorf("MatchIDLanguage(%s, %q) = %q %v, want ErrCardUnknownID", tt.uuid, tt.language, got, err)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("MatchIDLanguage(%s, %q) = %q %v, want %q", tt.uuid, tt.language, got, err, tt.want)
		}
	}
}
