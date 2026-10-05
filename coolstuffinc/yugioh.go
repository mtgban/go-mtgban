package coolstuffinc

import "strings"

// yugiohCodes spells the set codes the buylist writes unpadded the way the
// catalog does: the Dinosaurs Rage deck is SD9 here and SD09 there, and the
// third Hobby League is HL3 where the catalog says HL03. Both the Number and
// the note carry them.
var yugiohCodes = strings.NewReplacer(
	"SD9-SS1", "SD09-ENSS1",
	"SD9-EN", "SD09-EN",
	"HL3EN", "HL03-EN",
	"HL3-EN", "HL03-EN",
)
