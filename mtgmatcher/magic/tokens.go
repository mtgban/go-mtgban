package magic

import (
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// IsToken reports the token names the datastore cannot answer for itself.
// Two kinds are left: the names whose whole set is dropped when the
// datastore is built - art series, minigames, front cards, oversized
// oddities, arena and online codes, rules tips, the World Championship
// extras - and the storefront spellings that decorate a carried token with
// a word of their own. The datastore's own token list is consulted before
// this is ever asked, so every token filed under a surviving set is already
// answered there and needs no clause here.
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
	// Anything token
	case mtgmatcher.Contains(name, "Arena Code"),
		mtgmatcher.Contains(name, "Art Series"),
		mtgmatcher.Contains(name, "Charlie Brown"),
		mtgmatcher.Contains(name, "Checklist"),
		mtgmatcher.Contains(name, "DFC Helper"),
		mtgmatcher.Contains(name, "Experience C"),
		mtgmatcher.Contains(name, "Guild Symbol"),
		mtgmatcher.Contains(name, "Magic Minigame"),
		mtgmatcher.Contains(name, "Morph Overlay"),
		mtgmatcher.Contains(name, "On Your Turn"),
		mtgmatcher.Contains(name, "Online Code"),
		mtgmatcher.Contains(name, "Oversize"),
		mtgmatcher.Contains(name, "Punch Out"),
		mtgmatcher.Contains(name, "Token"),
		mtgmatcher.Contains(name, "Rules Tip"):
		return true
	// Verbatim, because normalizing drops every s and would then read
	// "Theme Deck Slither" as a decklist
	case strings.Contains(name, "Decklist"),
		strings.Contains(name, "The Initiative"):
		return true
	// The colon is what tells the bounty tokens from the dozen real cards
	// whose name merely begins with the word
	case strings.HasPrefix(name, "Bounty: "):
		return true
	// Alternative rules tip card names found on mkm
	case strings.HasPrefix(name, "Build a Deck: "),
		strings.HasPrefix(name, "Tip: "):
		return true
	}

	return false
}
