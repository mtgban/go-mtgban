package cardrush

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/matcher/magic"
)

// reNotes takes the treatment notes off the front of a row's name.
var reNotes = regexp.MustCompile(`^[(（]([^)）]*)[)）]`)

// reTrailingEdition reads the set a handful of rows name after the card,
// "稲妻/Lightning Bolt(A25)", in place of their pack code.
var reTrailingEdition = regexp.MustCompile(`[(（]([^)）]+)[)）]\s*$`)

// reTitle splits a store title into its grade, or the slab a graded copy
// sits in, the notes and names, the language and the set:
// "[EX+](FOIL)稲妻/Lightning Bolt《英語》【M10】", "〔PSA9鑑定済〕(FOIL)...". A
// listing on sale opens with "☆SALE☆", and a one-off copy carries a lot
// number after the set, "#296".
var reTitle = regexp.MustCompile(`^(?:☆SALE☆)?(?:\[([A-Z+]+)\]|(〔[A-Za-z]+[0-9.]+鑑定済〕))?(.*)《(.*)》【(.*)】(?:#\d+)?$`)

// reNumber is a note that is the printing's collector number, "(0001)".
var reNumber = regexp.MustCompile(`^0*([0-9]+)$`)

// treatments names each note in the words the catalog uses. A note naming a
// foil also sets the foil flag, whether or not it has an entry here.
var treatments = map[string]string{
	"ショーケース枠":   "Showcase",
	"旧枠仕様":      "Retro Frame",
	"日限定イラスト":   "Japanese",
	"設計図":       "Schematic",
	"スケッチアート":   "Sketch",
	"PWマークなし":   "No Symbol",
	"30th":      "30th Anniversary",
	"シリアル":      "Serialized",
	"不可視インク":    "Invisible Ink Foil",
	"シャッタードグラス": "Shattered Glass",
	"プロモパック版":   "Promo Pack",

	"サージFOIL":           "Surge Foil",
	"ファーストプレイスFOIL":     "First Place Foil",
	"エッチングFOIL":         "Etched Foil",
	"フラクチャーFOIL":        "Fracture Foil",
	"ギャラクシーFOIL":        "Galaxy Foil",
	"ハローFOIL":           "Halo Foil",
	"テクスチャーFOIL":        "Textured Foil",
	"マナFOIL":            "Mana Foil",
	"レイズドFOIL":          "Raised Foil",
	"コンフェッティFOIL":       "Confetti Foil",
	"ステップアンドコンプリートFOIL": "Step-and-Compleat Foil",
	"銀幕FOIL":            "Silver Foil",
	"チョコボトラックFOIL":      "Chocobo Track Foil",
	"ドラゴンスケイルFOIL":      "Dragonscale Foil",
	"オイルスリックFOIL":       "Oil Slick",
	"コズミックFOIL":         "Cosmic Foil",
	"シンギュラリティFOIL":      "Singularity Foil",
	"シルバースクロールFOIL":     "Silver Scroll",
	"ファセットFOIL":         "Facet Foil",
	"リップルFOIL":          "Ripple Foil",
	"箔押しFOIL":           "Gilded Foil",
	"紋章FOIL":            "Embossed Foil",
	"ネオンイエローFOIL":       "Neon Ink Yellow",
	"ネオンブルーFOIL":        "Neon Ink Blue",
	"ネオングリーンFOIL":       "Neon Ink Green",
	"ネオンレッドFOIL":        "Neon Ink Red",
	"ネオンパープルFOIL":       "Neon Ink Purple",
	"ネオンピンクFOIL":        "Neon Ink Pink",
	"ネオン3色FOIL":         "Neon Ink Three-Color",
}

// plainNotes say nothing about which printing a row is beyond what its set
// does: an unopened promo, a Mystical Archive card, a promo without the
// store's name stamped on, and a Kaladesh Inventions card in full or partial
// gloss, which the catalog prints as one.
var plainNotes = []string{"未開封", "アーカイブ", "店舗名無", "全面光沢FOIL", "部分光沢FOIL"}

// blackBorder is the note on a foreign black-bordered printing, which is a
// set of its own.
const blackBorder = "黒枠"

var blackBorderEditions = map[string]string{
	"4ED": "4BB",
	"CHR": "BCHR",
}

// isTwinNote reports a note telling apart two products the catalog prints
// as one: a store-stamped promo, a foil from one kind of booster or another
// ("ドラフト・セットブースター産"), an early or late print run ("通常パック版前期").
func isTwinNote(note string) bool {
	return note == "店舗名有" || strings.HasSuffix(note, "産") ||
		strings.HasPrefix(note, "通常パック版")
}

// cardNames corrects the store's misspellings.
var cardNames = map[string]string{
	"Hurkyl's Recal":         "Hurkyl's Recall",
	"Cenurion of the Marked": "Centurion of the Marked",
	"Islands":                "Island",
}

// ltcSolRings numbers the LTC Sol Ring each art is named for, read off the
// art: the tree, the dwarf's vault, the crowned king.
var ltcSolRings = map[string]string{
	"Elven":   "408",
	"Dwarven": "409",
	"Human":   "410",
}

// editions names the sets the store calls something the matcher does not
// read.
var editions = map[string]string{
	"P21":                             "PL21",
	"MagicFest Cards2019":             "PF19",
	"MagicFest Cards2019(FOIL)":       "PF19",
	"Champs Promos":                   "PCMP",
	"Starter 2000":                    "S00",
	"Deckmasters Garfield vs. Finkel": "DKM",
}

// unprintedEditions file reprints the catalog has no printing of their own
// for: the Secret Lair Commander decks' cards, which otherwise land on
// whichever Secret Lair drop shares the name.
var unprintedEditions = []string{"Reprint Cards(Secret Lair Commander)"}

// masterpieceEditions are the names the store files both Kaladesh
// Inventions and Amonkhet Invocations under.
var masterpieceEditions = []string{"MPS", "Kaladesh Inventions"}

// boxedEditions are the products the store files by the set each card was
// first printed in, "Battle Royale Box Set(TMP)".
var boxedEditions = map[string]string{
	"Battle Royale Box Set": "BRB",
	"Beatdown Box Set":      "BTD",
	"Anthologies":           "ATH",
	"CHR":                   "CHR",
}

// languages reads the row's language; English needs no word.
var languages = map[string]string{
	"日本語":      "Japanese",
	"英語":       "",
	"イタリア語":    "Italian",
	"フランス語":    "French",
	"スペイン語":    "Spanish",
	"ポルトガル語":   "Portuguese (Brazil)",
	"ファイレクシア語": "Phyrexian",
}

// fullArt is the note the store files both borderless and extended art
// under.
const fullArt = "フルアート"

// japaneseOnly are the notes on a printing made only in Japanese. Every
// other Japanese row prices a Japanese copy of an English printing, which
// is not the product the printing's id stands for.
var japaneseOnly = []string{"日限定イラスト", blackBorder}

// preprocess turns the store's naming of a card into the card: a title
// carrying notes and names, a set's code or name, and a language.
func preprocess(b *matcher.Backend, title, pack, lang string) (*matcher.InputCard, error) {
	name := title
	var notes []string
	for {
		m := reNotes.FindStringSubmatch(name)
		if m == nil {
			break
		}
		notes = append(notes, m[1])
		name = name[len(m[0]):]
	}

	edition := pack
	if slices.Contains(unprintedEditions, edition) {
		return nil, matcher.ErrUnsupported
	}
	m := reTrailingEdition.FindStringSubmatch(name)
	if m != nil {
		name = strings.TrimSpace(name[:len(name)-len(m[0])])
		// A reprint product names the set the card was first printed in,
		// which tells its reprints apart.
		if strings.HasPrefix(edition, "Reprint Cards(") {
			notes = append(notes, cmp.Or(editions[m[1]], m[1]))
		} else {
			edition = m[1]
		}
	}
	// "Reprint Cards(Mystery Booster FOIL)", "MagicFest Cards2019(FOIL)".
	foilPack := strings.HasSuffix(edition, "FOIL)")

	code, found := editions[edition]
	if found {
		edition = code
	}
	box, _, found := strings.Cut(edition, "(")
	if found && boxedEditions[box] != "" {
		edition = boxedEditions[box]
	}

	// "※コレクターブースター産" says which booster a copy came from.
	name, origin, found := strings.Cut(name, "※")
	if found {
		notes = append(notes, strings.Trim(origin, "※"))
	}

	name = englishName(name)
	if name == "" {
		return nil, matcher.ErrUnsupported
	}
	fixed, found := cardNames[name]
	if found {
		name = fixed
	}

	// PRM holds every kind of promo, so the matcher looks among them all.
	promoWildcard := edition == "PRM"
	if promoWildcard {
		edition = ""
	}
	if slices.Contains(masterpieceEditions, edition) {
		edition = "MPS"
		if len(printingsIn(b, name, edition, matcher.FinishFoil)) == 0 {
			edition = "MP2"
		}
	}

	language, found := languages[lang]
	if !found {
		return nil, fmt.Errorf("unknown language %q", lang)
	}
	if language == "Japanese" && !slices.ContainsFunc(notes, func(note string) bool {
		return slices.Contains(japaneseOnly, note)
	}) {
		return nil, matcher.ErrUnsupported
	}

	foil := foilPack || slices.ContainsFunc(notes, func(note string) bool {
		return strings.HasSuffix(note, "FOIL")
	})
	special := slices.ContainsFunc(notes, func(note string) bool {
		return strings.HasSuffix(note, "FOIL") && note != "FOIL"
	})
	numbered := slices.ContainsFunc(notes, reNumber.MatchString)
	rainbow := slices.Contains(notes, doubleRainbow)
	// The store numbers its Secret Lair rows wherever the drops share a
	// name, so one without a number is one of several it cannot tell apart.
	if edition == "SLD" && !numbered && len(printingsIn(b, name, edition, finishOf(foil))) > 1 {
		return nil, matcher.ErrUnsupported
	}

	var variants []string
	for _, note := range notes {
		word, isTreatment := treatments[note]
		ring, isRing := ltcSolRings[note]
		number := reNumber.FindStringSubmatch(note)
		switch {
		case note == "FOIL", slices.Contains(plainNotes, note):
		case note == "ファイレクシア語":
			language = "Phyrexian"
		case isTwinNote(note):
			return nil, matcher.ErrUnsupported
		case note == blackBorder:
			code, found := blackBorderEditions[edition]
			if !found {
				return nil, fmt.Errorf("no black-bordered %s in %q", edition, title)
			}
			edition = code
		case note == fullArt && numbered:
		case note == fullArt:
			treatment, printings := fullArtPrintings(b, name, edition, foil, special)
			// A PRM row's search leaves the promo sets once it reads a
			// treatment, so the row names the one promo that has it.
			if promoWildcard && len(printings) > 1 {
				return nil, fmt.Errorf("several full-art promos of %s for %q", name, title)
			}
			if promoWildcard && len(printings) == 1 {
				edition = printings[0].SetCode
				promoWildcard = false
				variants = append(variants, printings[0].Number)
				continue
			}
			if treatment == "" && len(printingsIn(b, name, edition, "")) > 1 {
				return nil, fmt.Errorf("no full-art %s in %s for %q", name, edition, title)
			}
			variants = append(variants, treatment)
		case isTreatment:
			variants = append(variants, word)
		case note == doubleRainbow:
			variants = append(variants, rainbowTreatment(b, name, edition))
		case isRing && name == "Sol Ring" && edition == "LTC":
			variants = append(variants, serializedNumber(b, name, edition, ring, rainbow))
		case number != nil:
			variants = append(variants, serializedNumber(b, name, edition, number[1], rainbow))
		default:
			// A year, "(2020年)", or a word in English, "(Dwarven)".
			note = strings.TrimSuffix(note, "年")
			// An untranslated treatment would price the plain printing.
			if hasJapanese(note) {
				return nil, fmt.Errorf("unknown note %q in %q", note, title)
			}
			variants = append(variants, note)
		}
	}

	return &matcher.InputCard{
		Name:          name,
		Variation:     strings.Join(variants, " "),
		Edition:       edition,
		Foil:          foil,
		Language:      language,
		PromoWildcard: promoWildcard,
	}, nil
}

// doubleRainbow is the store's name for both foils the catalog calls double
// rainbow and rainbow, serialized or not.
const doubleRainbow = "ダブルレインボウFOIL"

// rainbowTreatment names the rainbow foil the card has in the set.
func rainbowTreatment(b *matcher.Backend, name, edition string) string {
	printings := printingsIn(b, name, edition, matcher.FinishFoil)
	hasPromo := func(promoType string) bool {
		return slices.ContainsFunc(printings, func(card matcher.Card) bool {
			return card.HasPromoType(promoType)
		})
	}
	if !hasPromo(magic.PromoTypeDoubleRainbow) && hasPromo(magic.PromoTypeRainbowFoil) {
		return "Rainbow Foil"
	}
	return "Double Rainbow Foil"
}

// serializedNumber is the collector number a numbered rainbow foil row
// names: the serialized copy filed beside the plain one, "750z", where the
// set has it.
func serializedNumber(b *matcher.Backend, name, edition, number string, rainbow bool) string {
	if !rainbow {
		return number
	}
	for _, card := range printingsIn(b, name, edition, matcher.FinishFoil) {
		if card.Number == number+"z" {
			return card.Number
		}
	}
	return number
}

// specialFoils are the foils the store names with a note of their own, so a
// row reads as one of them only when it says so.
var specialFoils = []string{
	magic.PromoTypeChocoboTrackFoil, magic.PromoTypeConfettiFoil,
	magic.PromoTypeCosmicFoil, magic.PromoTypeDoubleRainbow,
	magic.PromoTypeDragonScaleFoil, magic.PromoTypeEmbossed,
	magic.PromoTypeFacetFoil, magic.PromoTypeFirstPlaceFoil,
	magic.PromoTypeFractureFoil, magic.PromoTypeGalaxyFoil,
	magic.PromoTypeGilded, magic.PromoTypeHaloFoil, magic.PromoTypeInvisibleInk,
	magic.PromoTypeManaFoil, magic.PromoTypeNeonInk, magic.PromoTypeOilSlick,
	magic.PromoTypeRainbowFoil, magic.PromoTypeRaisedFoil,
	magic.PromoTypeRippleFoil, magic.PromoTypeSerialized,
	magic.PromoTypeSilverFoil, magic.PromoTypeSilverScroll,
	magic.PromoTypeSingularityFoil, magic.PromoTypeStepAndCompleat,
	magic.PromoTypeSurgeFoil, magic.PromoTypeTextured,
}

// fullArtTreatments are what the store files under full art, in the order a
// card's printings are read for one: extended art where the card has it,
// else borderless, a full-art basic land, or a showcase frame.
var fullArtTreatments = []struct {
	name string
	has  func(matcher.Card) bool
}{
	{"Extended Art", func(c matcher.Card) bool { return c.HasFrameEffect(magic.FrameEffectExtendedArt) }},
	{"Borderless", func(c matcher.Card) bool { return c.BorderColor == magic.BorderColorBorderless }},
	{"Full Art", func(c matcher.Card) bool { return c.IsFullArt }},
	{"Showcase", func(c matcher.Card) bool { return c.HasFrameEffect(magic.FrameEffectShowcase) }},
}

// fullArtPrintings names what the store calls full art for one card in one
// set, and returns the printings that have it. The printings sold in the
// row's finish are read first, and special foils only when the row names
// one. The store numbers the rows where two of these would compete.
func fullArtPrintings(b *matcher.Backend, name, edition string, foil, special bool) (string, []matcher.Card) {
	for _, finish := range []string{finishOf(foil), ""} {
		var printings []matcher.Card
		for _, card := range printingsIn(b, name, edition, finish) {
			if slices.ContainsFunc(specialFoils, card.HasPromoType) == special {
				printings = append(printings, card)
			}
		}
		for _, treatment := range fullArtTreatments {
			having := slices.DeleteFunc(slices.Clone(printings), func(card matcher.Card) bool {
				return !treatment.has(card)
			})
			if len(having) > 0 {
				return treatment.name, having
			}
		}
	}
	return "", nil
}

func finishOf(foil bool) string {
	if foil {
		return matcher.FinishFoil
	}
	return matcher.FinishNonfoil
}

// printingsIn returns the card's printings in the set named by code or by
// name, or in every promo set where the name is empty, as Match searches a
// PRM row; names compare the way the matcher does. Only printings sold in
// the finish given count, or in any where it is empty.
func printingsIn(b *matcher.Backend, name, edition, finish string) []matcher.Card {
	var sets []*matcher.Set
	if edition == "" {
		ids, _ := b.SearchEquals(name)
		for _, id := range ids {
			co, err := b.GetUUID(id)
			if err != nil {
				continue
			}
			set := b.Sets[co.SetCode]
			if set.Type == "promo" && !slices.Contains(sets, set) {
				sets = append(sets, set)
			}
		}
	} else {
		set, err := b.GetSet(edition)
		if err != nil {
			set, err = b.GetSetByName(edition)
		}
		if err != nil {
			return nil
		}
		sets = append(sets, set)
	}

	norm := matcher.Normalize(name)
	var out []matcher.Card
	for _, set := range sets {
		for _, card := range set.Cards {
			front, _, _ := strings.Cut(card.Name, " // ")
			named := matcher.Normalize(card.Name) == norm || matcher.Normalize(front) == norm
			// Each face of a double-faced card is listed on its own.
			twin := slices.ContainsFunc(out, func(c matcher.Card) bool {
				return c.SetCode == card.SetCode && c.Number == card.Number
			})
			if named && !twin && (finish == "" || slices.Contains(card.Finishes, finish)) {
				out = append(out, card)
			}
		}
	}
	return out
}

// englishName reads the English half of "稲妻/Lightning Bolt". A printing
// with a flavor name lists it first, "JP/EN　//　JP/EN", and is read by the
// card's own name after it; a split card joins its halves with "+".
func englishName(name string) string {
	parts := strings.Split(name, "//")
	name = strings.TrimSpace(strings.Trim(parts[len(parts)-1], "\u3000 "))
	i := strings.LastIndex(name, "/")
	if i >= 0 {
		name = strings.TrimSpace(name[i+1:])
	}
	// Some names carry a left-to-right mark.
	name = strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, name)
	if name == "" || hasJapanese(name) {
		return ""
	}
	if len(name) > 1 {
		name = name[:1] + strings.ReplaceAll(name[1:], "+", " // ")
	}
	return name
}

func hasJapanese(s string) bool {
	return strings.ContainsFunc(s, func(r rune) bool {
		return unicode.In(r, unicode.Hiragana, unicode.Katakana, unicode.Han) ||
			r == '※' || r == 'ー' || r == '・'
	})
}
