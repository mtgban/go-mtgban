package mtgmatcher

import (
	"strings"
)

var replacer = strings.NewReplacer(
	// Remove a very common field, sometimes added with no reason
	// Needs the dashes to work with will-o'-the-wisp, whish is why
	// it needs to be before removing the dash step
	" the ", "",
	"-the-", "",

	// Quotes and commas and whatnot
	"''", "",
	"“", "",
	"”", "",
	"\"", "",
	"'", "",
	"-", "",
	"’", "",
	",", "",
	"®", "",
	":", "",
	"：", "",
	"~", "",
	"(", "",
	")", "",
	".", "",
	"!", "",
	"?", "",
	"+", "",

	// UNF blanks
	"_________", "_____",
	"________", "_____",
	"_______", "_____",
	"______", "_____",

	// Separators
	"goblin // soldier", "goblin // soldier",
	"/", "",
	"|", "",
	"trial and error", "trial and error",
	" and ", "",
	" to ", "",
	" & ", "",

	// Accented characters
	"â", "a",
	"á", "a",
	"à", "a",
	"ä", "a",
	"é", "e",
	"í", "i",
	"ö", "o",
	"ó", "o",
	"ú", "u",
	"û", "u",
	"ü", "u",
	"ñ", "n",

	// Ancient ligature
	"æ", "ae",

	// Also plurals, just preserve 'blossom' that aliases 'lotus bloom'
	// and 'asp' for 'tangle asp'/'tanglesap', and ogress...
	// 'vs' is a key for determining duel decks
	// Any accented s need to be removed as well to behave like a normal s
	"asp", "asp",
	"lossom", "lossom",
	"ogress", "ogress",
	"slash", "slash",
	"squash", "squash",
	"vs", "vs",
	"š", "",
	"s", "",

	// Spaces are overrated, except when not
	"waste land", "waste land",
	" ", "",
)

// Normalize uses the rules defined in Replacer to replace uncommon elements of
// card names, dropping all the spaces and producing a lowercase string.
func Normalize(str string) string {
	str = strings.TrimSpace(str)
	str = strings.ToLower(str)
	str = replacer.Replace(str)
	return str
}

// Compare strings after both are Normalize-d.
func Equals(str1, str2 string) bool {
	return Normalize(str1) == Normalize(str2)
}

// Check if str1 contains str2 after both are Normalize-d.
func Contains(str1, str2 string) bool {
	return strings.Contains(Normalize(str1), Normalize(str2))
}

// Check if str2 is the prefix of str1 after both are Normalize-d.
func HasPrefix(str1, str2 string) bool {
	return strings.HasPrefix(Normalize(str1), Normalize(str2))
}

// Check if str2 is the suffix of str1 after both are Normalize-d.
func HasSuffix(str1, str2 string) bool {
	return strings.HasSuffix(Normalize(str1), Normalize(str2))
}

var sealedReplacer = strings.NewReplacer(
	"booster pack display", "booster box",
	"booster display box", "booster box",
	"booster display", "booster box",
	"display box", "booster box",

	"magic modern", "modern",
	"magic arena", "arena",
	"magic game night", "game night",
	"magic 30th", "30th",

	"2010 core set", "magic 2010",
	"2011 core set", "magic 2011",
	"2012 core set", "magic 2012",
	"2013 core set", "magic 2013",
	"2014 core set", "magic 2014",
	"2015 core set", "magic 2015",
	"m10", "",
	"m11", "",
	"m12", "",
	"m13", "",
	"m14", "",
	"m15", "",

	"core 20", "core set 20",
	"revised booster", "revised edition booster",
	"universes beyond", "",
	"complete set", "",
	"multi player", "",

	"theme deck box", "theme deck display",
	"fat pack bundle", "bundle",
	"boxset", "set",
	"box set", "set",
	"prerelease box", "prerelease",
	"classic", "",
	"hedron", "",

	"4th edition", "fourth edition",
	"5th edition", "fifth edition",
	"6th edition", "sixth edition",
	"seventh edition", "7th edition",
	"eighth edition", "8th edition",
	"ninth edition", "9th edition",
	"tenth edition", "10th edition",

	"set of six", "complete",
	"set of five", "complete",
	"set of four", "complete",
	"set of two", "complete",
	"set of 6", "complete",
	"set of 5", "complete",
	"set of 4", "complete",
	"set of 2", "complete",

	"2-player game", "two player starter",
	"2-player", "two player",
	"2 player", "two player",
	"tournament pack", "starter",
	"starter set", "starter",
	"start deck", "starter",
	"starter deck box", "starter",

	"summer superdrop", "",
	"city of guilds", "",

	" 12", "",
	" 10", "",
	"pack", "",
	"kit", "",
	"deck", "",
	"draft", "",
	"guild", "",
	"the ", "",
	" the ", "",
)

func SealedNormalize(str string) string {
	str = strings.TrimSpace(str)
	str = strings.ToLower(str)

	str = strings.TrimSuffix(str, " w")
	str = strings.TrimSuffix(str, " u")
	str = strings.TrimSuffix(str, " b")
	str = strings.TrimSuffix(str, " r")
	str = strings.TrimSuffix(str, " g")

	str = sealedReplacer.Replace(str)
	str = replacer.Replace(str)
	return str
}

func SealedEquals(str1, str2 string) bool {
	return SealedNormalize(str1) == SealedNormalize(str2)
}

func SealedContains(str1, str2 string) bool {
	return strings.Contains(SealedNormalize(str1), SealedNormalize(str2))
}
