package mtgmatcher

import (
	"slices"
	"strings"
)

// normalizeLanguage applies the language codes and listing hints used by Match.
func (c *InputCard) normalizeLanguage() {
	// Set up language
	if c.Language != "" {
		lang, found := LanguageCode2LanguageTag[strings.ToLower(c.Language)]
		if found {
			c.Language = lang
		} else {
			for field := range strings.FieldsSeq(c.Language) {
				field = Title(field)
				if slices.Contains(allLanguageTags, field) {
					c.Language = field
					break
				}
			}
		}
	}
	// Override if needed
	for _, tag := range allLanguageTags {
		if c.Contains(tag) {
			c.Language = tag
			break
		}
	}
}

func matchesLanguage(printed, requested string) bool {
	return strings.Contains(printed, requested)
}
