package magic

import (
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// IsToken reports the names Magic knows as tokens without their carrying a
// token type of their own: the rules tips and checklists, the oversized
// oddities, the storefront spellings that only ever name a token. The
// datastore's own token list is checked before this is ever asked.
func (Rules) IsToken(b *mtgmatcher.Backend, name string) bool {
	switch name {
	// Custom token names
	case "A Threat to Alara: Nicol Bolas",
		"Fun Format: Pack Wars",
		"On An Adventure",
		"Pyromantic Pixels",
		"Theme: The Gold Standard",
		"Theme: WUBRG Cards":
		return true
	// WCD extra cards
	case "Biography",
		"Blank",
		"Overview":
		return true
	}
	switch {
	// Avoid confusion with Monarch and Emblem below
	case mtgmatcher.HasPrefix(name, "Emblem of the Warmind"),
		mtgmatcher.HasPrefix(name, "Kavu Monarch"),
		mtgmatcher.HasPrefix(name, "Leering Emblem"),
		// and with the `card` wildcard
		mtgmatcher.HasPrefix(name, "Our Market Research"):
		return false
	// Anything token
	case strings.Contains(name, " Card"),
		strings.Contains(name, "Card "),
		strings.HasPrefix(name, "Bounty"),
		mtgmatcher.Contains(name, "Arena Code"),
		mtgmatcher.Contains(name, "Art Series"),
		mtgmatcher.Contains(name, "Charlie Brown"),
		mtgmatcher.Contains(name, "Checklist"),
		mtgmatcher.Contains(name, "Copy"),
		mtgmatcher.Contains(name, "Decklist"),
		mtgmatcher.Contains(name, "DFC Helper"),
		mtgmatcher.Contains(name, "Dungeon of the Mad Mage"),
		mtgmatcher.Contains(name, "Emblem"),
		mtgmatcher.Contains(name, "Experience C"),
		mtgmatcher.Contains(name, "Giant Teddy Bear"),
		mtgmatcher.Contains(name, "Guild Symbol"),
		mtgmatcher.Contains(name, "Magic Minigame"),
		mtgmatcher.Contains(name, "The Monarch"),
		strings.Contains(name, "The Initiative"),
		mtgmatcher.Contains(name, "Morph Overlay"),
		mtgmatcher.Contains(name, "On Your Turn"),
		mtgmatcher.Contains(name, "Online Code"),
		mtgmatcher.Contains(name, "Oversize"),
		mtgmatcher.Contains(name, "Punch Out"),
		mtgmatcher.Contains(name, "Token"),
		mtgmatcher.Contains(name, "Rules Tip"):
		return true
	// Alternative rules tip card names found on mkm
	case strings.HasPrefix(name, "Build a Deck: "),
		strings.HasPrefix(name, "Tip: "):
		return true
	}

	return false
}
