package bazaarofmagic

import (
	"cmp"
	"errors"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

// errSkip marks a listing this scraper does not price: a signed copy.
var errSkip = errors.New("not priced")

// qualifierRe matches the last parenthesised qualifier of a title:
// "A Good Day to Die (#438) (Klingon) (foil) (showcase)".
var qualifierRe = regexp.MustCompile(`\s*\(([^()]*)\)$`)

// numberRe matches a collector number qualifier, "#428" or "#551f".
var numberRe = regexp.MustCompile(`^#\s*(\S+)$`)

// artRe matches the art a basic land is, counted in collector number order
// within its set: "Forest (3)".
var artRe = regexp.MustCompile(`^[1-9]$`)

// bomCondition maps the store's grades onto ours. It grades on its own
// scale, M, NM, EX, VG, GD and Poor, so it keeps its own table.
var bomCondition = map[string]mtgban.Condition{
	"M":    mtgban.NM,
	"NM":   mtgban.NM,
	"EX":   mtgban.SP,
	"VG":   mtgban.MP,
	"GD":   mtgban.HP,
	"Poor": mtgban.PO,
}

var languages = map[string]bool{
	"Chinese":    true,
	"French":     true,
	"German":     true,
	"Italian":    true,
	"Japanese":   true,
	"Korean":     true,
	"Portuguese": true,
	"Russian":    true,
	"Spanish":    true,
}

// setSuffixes are the endings of the store's set names that file a part of
// a set under its own entry, with what the matcher reads for each.
var setSuffixes = []struct {
	suffix  string
	edition string
	variant string
}{
	{" Varianten", "", ""},
	{" Variants", "", ""},
	{" Planeswalker Stamped", " Promos", "Planeswalker Stamp"},
	{" Prerelease Promo's", " Promos", "Prerelease"},
	{" Promo's", " Promos", ""},
}

// listing is a title read apart: the card the matcher reads and the copy's
// grade.
type listing struct {
	card *matcher.InputCard
	cond mtgban.Condition
}

// storeCodes are the store's set codes the datastore spells another way.
var storeCodes = map[string]string{
	"avb":     "DDH",
	"clc":     "CLB",
	"cmc":     "CMM",
	"dd3_dvd": "DVD",
	"dd3_evg": "EVG",
	"dd3_gvl": "GVL",
	"dd3_jvc": "JVC",
	"ecmr":    "CMR",
	"evt":     "DDF",
	"hvm":     "DDL",
	"ivg":     "DDJ",
	"jvv":     "DDM",
	"kvd":     "DDG",
	"pvc":     "DDE",
	"svt":     "DDK",
	"vvk":     "DDI",
}

// promoSets are the store's sets gathering promos from many of the
// datastore's, where a code the datastore also uses names only one of them.
// The value says whether the store's set name narrows the search: it does
// for its prerelease promos, and its Friday Night Magic set holds the promo
// packs handed out there, which no FNM edition does.
var promoSets = map[string]bool{
	"fnm": false,
	"mpr": true,
	"pov": true,
	"pre": true,
	"wpn": true,
}

// notCards end the names of the inserts a store sells beside its cards,
// which the datastore does not hold.
var notCards = []string{" front card", " Front Card", " helper", " helper card", " reminder card", " substitute card", " sticker sheet"}

// listCodeRe matches the set a card on The List was first printed in, which
// with its number makes the List's own number: "(#393) (LCI)" is LCI-393.
var listCodeRe = regexp.MustCompile(`^[A-Z0-9]{3,5}$`)

// ptRe matches a token's power and toughness, which the matcher does not
// read: "Spirit token (1/1)".
var ptRe = regexp.MustCompile(`^[0-9*X+]+/[0-9*X+]+$`)

// preprocess reads a listing title in one of the store's sets. The edition
// comes from the set code where the datastore knows it, and from the
// store's set name otherwise.
func preprocess(b *matcher.Backend, set Set, title string, finish string) (*listing, error) {
	name := title
	var qualifiers []string
	for {
		m := qualifierRe.FindStringSubmatchIndex(name)
		if m == nil || m[0] == 0 {
			break
		}
		qualifiers = append([]string{name[m[2]:m[3]]}, qualifiers...)
		name = name[:m[0]]
	}
	name = strings.TrimSpace(name)
	if strings.HasPrefix(name, "Art Card ") || slices.ContainsFunc(notCards, func(s string) bool { return strings.HasSuffix(name, s) }) {
		return nil, errSkip
	}
	if base, found := strings.CutSuffix(name, " Token"); found {
		name = base + " token"
	}

	out := &listing{
		card: &matcher.InputCard{
			Name: name,
			Foil: finish != "" && finish != "0",
		},
		cond: mtgban.NM,
	}
	var number, art, listCode string
	var variants []string
	var blackBorder bool
	for _, q := range qualifiers {
		if artRe.MatchString(q) {
			art = q
			continue
		}
		if ptRe.MatchString(q) {
			continue
		}
		if cond, found := bomCondition[q]; found {
			out.cond = cond
			continue
		}
		if languages[q] {
			out.card.Language = q
			continue
		}
		m := numberRe.FindStringSubmatch(q)
		if m != nil {
			number = m[1]
			continue
		}
		if set.Code == "plist" && listCodeRe.MatchString(q) {
			listCode = q
			continue
		}
		switch strings.ToLower(q) {
		case "signed":
			return nil, errSkip
		case "foil":
			out.card.Foil = true
			continue
		case "bb":
			blackBorder = true
			continue
		}
		if strings.Contains(strings.ToLower(q), "foil") {
			out.card.Foil = true
		}
		variants = append(variants, q)
	}

	if listCode != "" && number != "" {
		number = listCode + "-" + number
	} else if listCode != "" {
		variants = append(variants, listCode)
	}

	edition := set.Name
	if named, found := promoSets[set.Code]; found {
		if named {
			out.card.Edition = edition
		}
		out.card.PromoWildcard = true
		if number != "" {
			variants = append([]string{number}, variants...)
		}
		out.card.Variation = strings.Join(variants, " ")
		return out, nil
	}
	code := cmp.Or(storeCodes[set.Code], strings.ToUpper(set.Code))
	known, err := b.GetSet(code)
	if err == nil {
		edition = known.Name
	} else {
		for _, s := range setSuffixes {
			base, found := strings.CutSuffix(set.Name, s.suffix)
			if !found {
				continue
			}
			edition = base + s.edition
			if s.variant != "" {
				variants = append(variants, s.variant)
			}
			break
		}
		known, _ = b.GetSetByName(edition)
	}
	if known != nil {
		if art != "" && number == "" {
			number = nthPrinting(b, known, name, art)
		} else if number != "" {
			number = setNumber(b, known, name, number)
		}
	}
	if blackBorder {
		edition += " Foreign Black Border"
	}
	out.card.Edition = edition

	if number != "" {
		variants = append([]string{number}, variants...)
	}
	out.card.Variation = strings.Join(variants, " ")
	return out, nil
}

// numbersOf returns the collector numbers of name in set, or in its token
// set for a token, in collector number order.
func numbersOf(b *matcher.Backend, set *matcher.Set, name string) []string {
	cards := set.Cards
	token, isToken := strings.CutSuffix(name, " token")
	if isToken {
		tokens, err := b.GetSet(cmp.Or(set.TokenSetCode, "T"+set.Code))
		if err != nil {
			return nil
		}
		cards, name = tokens.Cards, token
	}
	norm := matcher.Normalize(name)
	var numbers []string
	for _, card := range cards {
		if matcher.Normalize(card.Name) == norm {
			numbers = append(numbers, card.Number)
		}
	}
	slices.SortFunc(numbers, func(a, b string) int {
		x, _ := strconv.Atoi(strings.TrimRightFunc(a, notDigit))
		y, _ := strconv.Atoi(strings.TrimRightFunc(b, notDigit))
		return cmp.Or(cmp.Compare(x, y), strings.Compare(a, b))
	})
	return numbers
}

func notDigit(r rune) bool {
	return r < '0' || r > '9'
}

// nthPrinting returns the collector number of the nth art of name in set,
// which is how the store tells apart a set's alternate arts, or "" where
// the set holds fewer.
func nthPrinting(b *matcher.Backend, set *matcher.Set, name, nth string) string {
	n, _ := strconv.Atoi(nth)
	// A starred number is the foil copy of an art already counted.
	numbers := slices.DeleteFunc(numbersOf(b, set, name), func(number string) bool {
		return strings.Contains(number, "★")
	})
	if n > len(numbers) {
		return ""
	}
	return numbers[n-1]
}

// setNumber returns number as the set spells it. The store suffixes a "b"
// to the plain number of a printing whose sibling carries an "a", as
// Battle for Zendikar's full-art lands.
func setNumber(b *matcher.Backend, set *matcher.Set, name, number string) string {
	numbers := numbersOf(b, set, name)
	if len(numbers) == 0 || slices.Contains(numbers, number) {
		return number
	}
	bare := strings.TrimRightFunc(number, func(r rune) bool { return r >= 'a' && r <= 'z' })
	if bare != number && slices.Contains(numbers, bare) {
		return bare
	}
	return number
}
