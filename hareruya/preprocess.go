package hareruya

import (
	"errors"
	"regexp"
	"strings"
	"unicode"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

var reParens = regexp.MustCompile(`\(([^)]+)\)`)
var reBrackets = regexp.MustCompile(`\[([^\]]+)\]`)
var reSquares = regexp.MustCompile(`■([^■]+)■`)
var reJapanese = regexp.MustCompile(`[\p{Hiragana}\p{Katakana}\p{Han}]`)

var reCardName = regexp.MustCompile(`《([^》]+)》`)
var reThick = regexp.MustCompile(`【([^】]+)】`)

// dashSuffix reads the tag the storefront appends to a set code. "-RT" names
// the timeshifted reprints, which are a set of their own and numbered as one.
// Every other tag describes the frame or the booster a card came out of,
// which the collector number in the title already pins down, so they are
// dropped as they always were.
//
// The tempting generalisation is "-P" onto the set's promo set. It was tried
// and measured: those listings carry the base set's collector number, and the
// promo sets number differently, so it lost 51 rows and moved 151 more while
// gaining none.
func dashSuffix(base, suffix string) string {
	if suffix == "RT" && base == "MH1" {
		return "H1R"
	}
	return base
}

// storeStamped is the wording for a store championship prize printed with
// the winning shop's name.
const storeStamped = "店舗名印字入り"

// prerelease is the one promo a modern set numbers among its own cards, so
// the wording naming it does not send the listing to the set's promo line.
// The title is read for it further down either way.
const prerelease = "プレリリース"

// judgeRewards is the wording every judge tag translates into, spelled once
// so the tables and the rule cannot drift apart.
const judgeRewards = "Judge Reward"

// powerToughness matches the pair a title states in place of a collector
// number for the cards a set prints several times at several sizes.
var powerToughness = regexp.MustCompile(`^\d+/\d+$`)

// splitParens separates the parenthesised groups of a title into the collector
// number, which comes before the card name, and the series, which comes after
// it. A trailing group only counts as a series when it spells a set name
// outright: the matcher's lookup also honors set codes and its own aliases,
// and the storefront's promo qualifiers land on those by accident.
func splitParens(title string) (number, series, treatment string) {
	end := strings.Index(title, "》")
	for _, loc := range reParens.FindAllStringSubmatchIndex(title, -1) {
		group := title[loc[2]:loc[3]]
		if end >= 0 && loc[0] > end {
			set, err := mtgmatcher.GetSetByName(group)
			if err == nil && mtgmatcher.Normalize(set.Name) == mtgmatcher.Normalize(group) {
				series = group
				continue
			}
		}
		if number == "" {
			// A slash separates two spellings of one promo code, and the
			// first is the one the catalog files - except where both sides
			// are numbers, which is a power and toughness rather than a
			// code, and the only thing telling one variant of an Unstable
			// card from another.
			number = group
			if !powerToughness.MatchString(group) {
				// The padding is a collector number's, and a power is not
				// one: stripping a leading zero off "0/4" leaves "/4",
				// which names nothing and drops the listing.
				number = strings.TrimLeft(strings.Split(group, "/")[0], "0")
			}
			continue
		}

		// Past the number the title states the treatment, and only the
		// wordings the table answers are read as one: the rest of what a
		// group holds there is the card's colour.
		if treatment == "" && end >= 0 && loc[0] > end && group != prerelease {
			if _, found := editionTable[group]; found {
				treatment = group
			}
		}
	}
	return number, series, treatment
}

// Preprocess turns a storefront product into the card description the matcher
// takes, reporting an error for what is not a card.
func Preprocess(product Product) (*mtgmatcher.InputCard, error) {
	if strings.Contains(product.ProductNameEN, "Wyvern back") ||
		strings.Contains(product.ProductNameEN, "Orversized") ||
		strings.Contains(product.ProductNameEN, "Oversized") ||
		strings.Contains(product.ProductNameEN, "Error Card") ||
		strings.Contains(product.ProductNameEN, "Error card") ||
		strings.Contains(product.ProductNameEN, "H19") ||
		strings.Contains(product.ProductNameEN, "Test Print") ||
		strings.Contains(product.ProductName, "Ultra Pro Puzzle") ||
		strings.Contains(strings.ToLower(product.CardName), "test print") {
		return nil, mtgmatcher.ErrUnsupported
	}

	cardName := product.CardName
	fixup, found := cardTable[cardName]
	if found {
		cardName = fixup
	}

	foil := product.FoilFlag == "1"
	var edition string
	var variant string

	// Usually there is more information the JPN product line, but sometimes
	// we need to look at the English version too
	match := reBrackets.FindStringSubmatch(product.ProductName)
	if len(match) > 1 {
		edition = match[1]
		// Use the English information if present
		if reJapanese.MatchString(edition) {
			match = reBrackets.FindStringSubmatch(product.ProductNameEN)
			if len(match) > 1 {
				edition = match[1]
			}
		}
		if base, suffix, found := strings.Cut(edition, "-"); found {
			edition = dashSuffix(base, suffix)
		}
	}

	// Variant is always found in the English line
	match = reSquares.FindStringSubmatch(product.ProductNameEN)
	if len(match) > 1 {
		variant = match[1]
	}

	// The number is only found in the JPN line, which may name the series too
	number, series, _ := splitParens(product.ProductName)
	if series != "" {
		edition = series
	}
	if number != "" {
		if variant != "" {
			variant += " "
		}
		variant += number
	}

	fixup, found = editionTable[edition]
	if found {
		edition = fixup
	}

	switch edition {
	case "4ED":
		// Announced in the English name here, where the Japanese line
		// announces it in the group the finish otherwise occupies.
		if strings.Contains(product.ProductNameEN, "【Alternate】") {
			edition = "4EDALT"
		}
	case "Ampersand PROMOS":
		// The retail side never sees the Japanese marker: the English name
		// overrides it before this runs.
		if number != "" {
			edition = "PAFR"
			variant = number + "a"
		}
	case "Judge Foil":
		if mtgmatcher.IsBasicLand(cardName) && strings.Contains(product.ProductNameEN, "Jacinto") {
			edition = "P23"
			variant = ""
		}
	case "IE", "CE":
		cardName = strings.TrimPrefix(cardName, "【International Edition】")
		cardName = strings.TrimPrefix(cardName, "【Collector's Edition】")

		variants := mtgmatcher.SplitVariants(product.ProductNameEN)
		if len(variants) > 1 {
			variant = variants[1]
		}
	case "SLD":
		if strings.Contains(product.ProductNameEN, "SLD Commander Deck") {
			edition = "PLST"
		}
	default:
		if strings.Contains(edition, "P Stamped_") {
			edition = "Promo Pack"
			fields := strings.Split(edition, "_")
			if len(fields) > 1 {
				edition += " " + fields[1]
			}
		} else if strings.Contains(product.ProductNameEN, "Prerelease") {
			edition += " Prerelease"
		}

		variant = strings.Replace(variant, "RetroF ", "Retro Frame ", 1)
		cardName = strings.TrimPrefix(cardName, "【Gold Frame】")
	}

	override, found := promoMap[edition][cardName][variant]
	if found {
		edition = override.Edition
		variant = override.Variant
	}

	language := ""
	if product.Language == "1" {
		language = "Japanese"
	}

	if strings.Contains(product.ProductNameEN, "【No Emblem】") {
		variant += " No Symbol"
	}

	return &mtgmatcher.InputCard{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      foil,
		Language:  language,
	}, nil
}

// process titles like
// 【EN】【Foil】(168)《武器製造/Weapons Manufacturing》[EOE] 赤R
// 【EN】【Foil】(086)■プレリリース■《虚空間渡り/Weftwalking》[EOE] 青R
func preprocess(title string) (*mtgmatcher.InputCard, error) {
	if strings.Contains(title, "Ultra Pro Puzzle") {
		return nil, mtgmatcher.ErrUnsupported
	}

	// A store championship prize is printed with the winning shop's name on
	// it. The catalog holds the prize card once, without the name, so the
	// stamped copy has no printing of its own and answering with the plain
	// one prices that card off the stamp.
	if strings.Contains(title, storeStamped) {
		return nil, mtgmatcher.ErrUnsupported
	}

	// A misprint is not a printing of its own, and answering with the card it
	// is a misprint of prices that card off the error.
	if strings.Contains(title, "シンボル無し") {
		return nil, mtgmatcher.ErrUnsupported
	}

	var cardName string
	var edition string
	var variant string
	var foil bool

	title = strings.TrimPrefix(title, "【EN】")
	title = strings.Replace(title, "(Bottom)", "", -1)
	title = strings.Replace(title, "(Big Furry Monster)", "", -1)
	title = strings.Replace(title, "SDCC", "SDCC ", -1)
	title = strings.Replace(title, "No Emblem", "No Symbol", -1)

	// /Weapons Manufacturing
	matches := reCardName.FindStringSubmatch(title)
	if len(matches) > 1 {
		cardName = matches[1]
	}

	if strings.Contains(cardName, "/") {
		fields := strings.Split(cardName, "/")
		cardName = ""
		for _, field := range fields {
			if reJapanese.MatchString(field) {
				continue
			}
			cardName = field
		}
	}
	if cardName == "" {
		return nil, errors.New("invalid title format")
	}
	if reJapanese.MatchString(cardName) {
		return nil, mtgmatcher.ErrUnsupported
	}

	// [EOE]
	var promoLine bool
	matches = reBrackets.FindStringSubmatch(title)
	if len(matches) > 1 {
		edition = matches[1]
		if base, suffix, found := strings.Cut(edition, "-"); found {
			promoLine = suffix == "P"
			edition = dashSuffix(base, suffix)
		}
	}

	// (168) and (Junior Super Series)
	number, series, promoWording := splitParens(title)
	if series != "" {
		edition = series
	}

	// A -P edition is the set's promos rather than the set itself, and where
	// the promo is filed away from that set - the Arena League foils, the
	// game day and release printings - naming the set pins the listing to
	// one that does not hold it, before the treatment beside it is read.
	//
	// The number says which of the two this is. A modern set files its own
	// prerelease promo among its cards and the title gives that card's
	// number, so the set is right and keeping the suffix would lose it. The
	// older promos have no number of the set's to give, and the group the
	// title puts there instead is the treatment - a word, with no digit in
	// it, which is what tells the two apart.
	// A title that states the treatment past the number has said which of
	// the two it is outright, and needs no reading of the number at all.
	if promoLine && series == "" &&
		(promoWording != "" || !strings.ContainsFunc(number, unicode.IsDigit)) {
		edition += "-P"
	}

	// ■プレリリース■
	matches = reSquares.FindStringSubmatch(title)
	if len(matches) > 1 {
		variant = matches[1]
		variant = strings.TrimSpace(variant)
	}

	//【Foil】/【エッチング・Foil】
	matches = reThick.FindStringSubmatch(title)
	if len(matches) > 1 {
		foil = strings.Contains(matches[1], "Foil")
		// The treatment is announced where the plain finish would be, and
		// carries the same set tag and number as the printing it is a
		// treatment of, so nothing else in the title tells them apart.
		if treatment, found := treatmentTable[matches[1]]; found {
			if variant != "" {
				variant += " "
			}
			variant += treatment
		}
	}

	if number != "" {
		if variant != "" {
			variant += " "
		}
		variant += number
	}

	// The treatment follows the number rather than replacing it: the promo
	// line files more than one printing under the set's own number, and the
	// number is what picks between them.
	if promoWording != "" {
		if variant != "" {
			variant += " "
		}
		variant += editionTable[promoWording]
	}

	fixup, found := editionTable[edition]
	if found {
		edition = fixup
	}
	fixup, found = editionTable[variant]
	if found {
		// A value naming a set is the table saying which set the printing
		// is in, and only the edition can carry that. Left in the variant
		// it still reaches the printing where the edition is already a
		// promo line of its own, which is how the game day textless cards
		// resolve. A -P edition is not that: it names one set's promos and
		// pins the listing among them, and the Champs textless Imperious
		// Perfect is filed in PCMP rather than with Lorwyn's.
		_, setErr := mtgmatcher.GetSet(fixup)
		if promoLine && setErr == nil {
			edition, variant = fixup, ""
		} else {
			variant = fixup
		}
	}

	fixup, found = cardTable[cardName]
	if found {
		cardName = fixup
	}

	if strings.Contains(edition, "Pスタンプ_") ||
		strings.Contains(edition, "P Stamped_") ||
		strings.Contains(variant, "Promo Stamped") ||
		strings.Contains(variant, "プロモスタンプ付") {
		edition = "Promo Pack"
		fields := strings.Split(edition, "_")
		if len(fields) > 1 {
			edition += " " + fields[1]
		}
	}

	//variant = strings.Replace(variant, "RetroF ", "Retro Frame ", 1)
	//cardName = strings.TrimPrefix(cardName, "【Gold Frame】")

	override, found := promoMap[edition][cardName][variant]
	if found {
		edition = override.Edition
		variant = override.Variant
	}

	if strings.Contains(edition, "WC9") || strings.Contains(edition, "WC0") || edition == "PT96" {
		// The player the deck belonged to closes the title, past the last
		// Japanese field, and takes as many words as their name needs
		fields := strings.Fields(title)
		i := len(fields) - 1
		for i >= 0 && !reJapanese.MatchString(fields[i]) {
			i--
		}

		if i+1 < len(fields) {
			if variant != "" {
				variant += " "
			}
			variant += strings.Join(fields[i+1:], " ")
		}
	} else if strings.Contains(variant, "P30H") {
		edition = variant
	} else if strings.Contains(title, "プレリリース") {
		variant += " Prerelease"
	} else if strings.Contains(title, "シリアル入り") {
		variant += " Serialized"
	} else if edition == "4ED" && variant == "Alternate" {
		edition = "4EDALT"
	} else if strings.Contains(title, "アンパサンド") && number != "" {
		// The ampersand card reprints another at its own number, and the
		// promo set is where the two are kept apart: every one of them is
		// the base number with an "a" behind it, and they are the only
		// numbers in that set spelled that way. Without a number there is
		// nothing to suffix, and the promo set numbers the same card three
		// ways, so a listing that gives none is left where it was.
		edition = "PAFR"
		variant = number + "a"
	} else if strings.Contains(variant, doubleRainbow) && cardName != "Sol Ring" {
		// The marker travels with whatever else the title says about the
		// printing - its number, and the frame it is printed in - so the
		// variant is only ever equal to it on a title that says nothing
		// else, which the serialized listings do not.
		variant += " Serialized"
	}

	if strings.Contains(title, "(FNM)") ||
		strings.Contains(title, "(CardZ") ||
		strings.Contains(title, "SDCC") {
		// Wipes "[6ED-P]" tags which confuse the matcher
		edition = ""
	}

	// The same for a judge reward, which is a set of its own that the
	// storefront files under the set the card was first printed in: the
	// "-P" tag saying so is dropped with every other frame tag, so the base
	// set stands and deletes the promo printing before the wording naming
	// it is read. Gaea's Cradle answered with the $751 Urza's Saga land
	// where the listing is the $2660 judge foil, and it reaches the judge
	// set on its own once the edition stops contradicting it.
	//
	// The tag alone is not enough to go on - mapping "-P" onto a set's own
	// promo set was measured and lost 51 rows - and this asks for the
	// wording instead, which names the set rather than merely denying the
	// one the tag came from.
	if strings.Contains(variant, judgeRewards) {
		edition = ""
	}

	return &mtgmatcher.InputCard{
		Name:      cardName,
		Variation: strings.TrimSpace(variant),
		Edition:   edition,
		Foil:      foil,
	}, nil
}

var cardTable = map[string]string{
	"Chicken ? la King":                  "Chicken à la King",
	"Adorable | KittenAdorable | Kitten": "Adorable Kitten",
	"Tyrannosaurs Rex":                   "Tyrannosaurus Rex",
}

// treatmentTable names the treatments the storefront announces in the group
// the plain finish otherwise occupies, in the words the catalog labels them
// with. Only the marker says which printing a listing is: the treated
// printing shares its set tag and collector number with the plain one, so a
// marker read as a bare "Foil" prices the treatment as the plain card - and
// these are the printings a shop pays the most for.
//
// Every entry was read off the storefront's own buylist: each marker below
// appears there, and each names a promo type the catalog carries.
// doubleRainbow is the treatment every serialized printing is sold in: of
// the 289 printings the catalog files under it, 288 are serialized, and the
// one that is not is the Sol Ring excepted below.
const doubleRainbow = "Double Rainbow Foil"

var treatmentTable = map[string]string{
	"S&C・Foil":       "Step-and-Compleat Foil",
	"エッチング・Foil":     "Etched Foil",
	"オイルスリック・Foil":   "Oil Slick",
	"ギャラクシー・Foil":    "Galaxy Foil",
	"コンフェッティ・Foil":   "Confetti Foil",
	"サージ・Foil":       "Surge Foil",
	"ダブルレインボウ・Foil":  doubleRainbow,
	"テクスチャー・Foil":    "Textured Foil",
	"ドラゴンスケイル・Foil":  "Dragonscale Foil",
	"ネオンインク・Foil":    "Neon Ink",
	"ハロー・Foil":       "Halo Foil",
	"ファーストプレイス・Foil": "First Place Foil",
	"リップル・Foil":      "Ripple Foil",
	"レイズド・Foil":      "Raised Foil",
	"不可視インク":         "Invisible Ink",
	"銀幕・Foil":        "Silver Foil",

	// Not a finish: the alternate printing is announced in the same group.
	"アルターネイト版": "Alternate",
}

var editionTable = map[string]string{
	"2007年版ジャッジ褒賞":        "2007 Judge Rewards",
	"2010年版ジャッジ褒賞":        "2010 Judge Rewards",
	"2013年版ジャッジ褒賞":        "2013 Judge Rewards",
	"2015年版ジャッジ褒賞":        "2015 Judge Rewards",
	"2018年版ジャッジ褒賞":        "2018 Judge Rewards",
	"2020年版":              "2020 Edition",
	"30周年記念":              "30th Anniversary",
	"BOOKプロモ":             "Book Promo",
	"BOXプロモ":              "Box Promo",
	"CardZプロモ":            "CardZ Promo",
	"CSP構築済み":             "CST",
	"DCIマーク":              "DCI Promo",
	"Etched Foil 30周年プロモ": "P30M etched frame",
	"GPプロモ":               "Grand Prix Promos",
	"MCQプロモ":              "MCQ Promo",
	"Nationalプロモ":         "National Promos",
	"PWシンボル付き再版":          "Mystery Booster/The List",
	"RPTQプロモ":             "RPTQ Promos",
	"URL入りイベントプロモ":        "PURL",
	"WMCQプロモ":             "WMCQ Promo",
	"その他プロモ":              "Other Promos",
	"アリーナ":                "Arena",
	"アルターネイト版":            "Alternate",
	"アンパサンド":              "Ampersand",
	"アンパサンド・カード":          "Ampersand Promo",
	"イラスト違い":              "S-Chinese alt art",
	"ウギンの運命":              "Ugin's Fate",
	"エッチング・Foil":          "Etched Foil",
	"エラーカード":              "Misprint",
	"エンブレムあり":             "With Symbol",
	"エントリーセット":            "Intro Pack Promo",
	"エンブレムなし":             "No Symbol",
	"ゲートウェイ":              "Gateway",
	"ストアチャンピオンシップ":        "Store Championship",
	"ゲームデー":               "Game Day",
	"コマンドフェスト":            "Command Fest",
	"サージ・Foil":            "Surge Foil",
	"ショーダウン":              "Showdown",
	"ジャッジ褒賞":              "Judge Rewards",
	"基本セット系プロモ":           "Promo",
	"発売記念":                "Release",
	"スポットライトシリーズプロモ":      "Spotlight Series Promo",
	"ダブルレインボウ・Foil":       doubleRainbow,
	"テキストボックスレス ゲームデー":    "PCMP",
	"テキストレス Magic Fest":   "Textless Magic Fest",
	"テキストレス 褒賞プログラム":      "Textless Player Rewards",
	"テキストレス":              "Textless",
	"テストプリント":             "Test Print",
	"ヒストリープロモ":            " 30th Anniversary",
	"ファイレクシア語 その他プロモ":     "Phyrexian Other Promos",
	"ファイレクシア語 ジャッジ褒賞":     "Phyrexian Judge Reward",
	"フルアート 1":             "Full Art 1",
	"フルアート 2":             "Full Art 2",
	"フルアート コマンドフェスト":      "Fullart CommandFest",
	"プレリリース":              "Prerelease",
	"プロツアープロモ":            "Pro Tour Promos",
	"ボーダーレス Premier Play": "Borderless Premier Play",
	"ボーダーレス その他イベント記念":    "Borderless Other Event Commemoration",
	"ボーダーレス その他イベント記念系":   "Borderless Other Event",
	"ボーダーレス スポットライトシリーズプロモ": "Borderless Spotlight Series Promo",
	"ボーダーレス マーベル・レジェンドプロモ":  "LMAR",
	"ボーダーレス 褒賞プロモ":          "Borderless Player Rewards",
	"ボーダーレス":                "Borderless",
	"ボーダーレスショーダウン":          "Borderless Showdown",
	"マジックリーグ":               "Year of the Tiger 2022",
	"メディア系プロモ":              "Media Promo",
	"リセールプロモ":               "Resale Promo",
	"午年プロモ":                 "Year of the Horse 2026",
	"卯年プロモ":                 "Year of the Rabbit 2023",
	"大判カード":                 "Oversize",
	"対戦キット":                 "Clash Pack",
	"巳年プロモ":                 "Year of the Snake 2025",
	"拡張アート MagicConプロモ":     "Extended Art MagicCon Promo",
	"拡張アート その他プロモ":          "Extended Art Other Promos",
	"拡張アート":                 "Extended Art",
	"新枠 2008年版ジャッジ褒賞":       "Mordern Frame 2008 Judge Rewards",
	"旧枠 2000年版ジャッジ褒賞":       "Retro Frame 2000 Judge Rewards",
	"旧枠 ジャッジ褒賞":             "Retro Frame Judge Rewards",
	"旧枠 その他プロモ":             "Retro Frame Other Promos",
	"旧枠 ヒストリープロモ":           "Retro Frame 30th Anniversary",
	"旧枠 褒賞プログラム":            "Old Frame Rewards Program",
	"旧枠":                    "Retro Frame",
	"絵違いVer.":               "Alternate Art",
	"褒賞プログラム":               "Rewards Program",
	"辰年プロモ":                 "Year of the Dragon 2024",

	"S&C・Foil":             "Step-and-Compleat Foil",
	"Secret Lair Showdown": "SLP",
	"Retro Frame Promos":   "PLG21",
	"30th Promo":           "P30A",
	"POS Reward Promo":     "PW24",
	"CMA":                  "CM1",
	"WMC":                  "World Magic Cup Qualifiers",

	"DvD": "Duel Decks: Divine vs. Demonic",
	"EVG": "Duel Decks: Elves vs. Goblins",
	"EvG": "Duel Decks: Elves vs. Goblins",
	"GvL": "Duel Decks: Garruk vs. Liliana",
	"JvC": "Duel Decks: Jace vs. Chandra",

	"FNM": "Friday Night Magic",
}

var promoMap = map[string]map[string]map[string]struct {
	Edition string
	Variant string
}{
	"Other event promo": {
		"Swords to Plowshares": {
			"Borderless その他イベント記念系": {
				Edition: "PF25",
				Variant: "12",
			},
		},
	},
	"Other Event Promo": {
		"Swiftfoot Boots": {
			"卯年プロモ": {
				Edition: "PL23",
				Variant: "4",
			},
		},
	},
	"Other Event anniversary": {
		"Sol Ring": {
			"旧枠プロモ": {
				Edition: "PFDN",
				Variant: "1",
			},
		},
		"Vengevine": {
			"WMCQプロモ": {
				Edition: "WMC",
				Variant: "2013",
			},
		},
		"Sakura-Tribe Elder": {
			"E06": {
				Edition: "PJSE",
				Variant: "1E06",
			},
		},
		"Soltari Priest": {
			"E07": {
				Edition: "PJSE",
				Variant: "1E07",
			},
		},
		"Glorious Anthem": {
			"U08": {
				Edition: "PJAS",
				Variant: "1U08",
			},
		},
		"Steward of Valeron": {
			"URL入りイベントプロモ": {
				Edition: "PURL",
				Variant: "1",
			},
		},
		"Cryptic Command": {
			"MCQプロモ": {
				Edition: "PPRO",
				Variant: "2020-1",
			},
		},
		"Reya Dawnbringer": {
			"": {
				Edition: "P10E",
				Variant: "35",
			},
		},
		"Earl of Squirrel": {
			"": {
				Edition: "PUST",
				Variant: "108",
			},
		},
	},
	"Other Promos": {
		"Serra the Benevolent": {
			"Retro Frame その他プロモ": {
				Edition: "PF25",
				Variant: "1",
			},
		},
		"Ugin, the Spirit Dragon": {
			"Retro Frame その他プロモ": {
				Edition: "PF25",
				Variant: "6",
			},
		},
		"Ponder": {
			"その他プロモ": {
				Edition: "PF25",
				Variant: "2",
			},
		},
		"Sliver Hive": {
			"Retro Frame その他プロモ": {
				Edition: "PF25",
				Variant: "7",
			},
		},
		"Sakura-Tribe Elder": {
			"": {
				Edition: "PLG24",
				Variant: "1",
			},
		},
	},
	"PB・Draft Promos": {
		"Arcane Signet": {
			"Retro Frame PBドラフトプロモ": {
				Edition: "P30M",
				Variant: "1P",
			},
		},
		"Commander's Sphere": {
			"PBドラフトプロモ": {
				Edition: "PW24",
				Variant: "8",
			},
		},
		"Chaos Warp": {
			"PBドラフトプロモ": {
				Edition: "PW24",
				Variant: "7",
			},
		},
	},
	"Standard Showdown Promo": {
		"Monstrous Rage": {
			"Retro Frame Standard Showdown": {
				Edition: "PW25",
				Variant: "9",
			},
		},
	},
	"Showdown": {
		"Go for the Throat": {
			"Borderless ショーダウン": {
				Edition: "PCBB",
				Variant: "3",
			},
		},
	},
	"Spotlight Series Promo": {
		"Cloud, Midgar Mercenary": {
			"Borderless スポットライトシリーズプロモ": {
				Edition: "PPRO",
				Variant: "2025-1",
			},
		},
		"Terror of the Peaks": {
			"スポットライトシリーズプロモ": {
				Edition: "PSPL",
				Variant: "1",
			},
		},
	},
	"Commander Play": {
		"Palladium Myr": {
			"Retro Frame Commander Play": {
				Edition: "PW25",
				Variant: "6",
			},
		},
	},
	"Premier Play": {
		"Tifa Lockhart": {
			"Borderless Premier Play": {
				Edition: "PF25",
				Variant: "9",
			},
		},
	},
	"Magic Academy": {
		"Trinket Mage": {
			"Retro Frame Magic Academy": {
				Edition: "PW25",
				Variant: "8",
			},
		},
	},
	"MagicCon Promo": {
		"Sokka, Bold Boomeranger": {
			"Extended Art MagicConプロモ": {
				Edition: "PURL",
				Variant: "2025-4",
			},
		},
		"J. Jonah Jameson": {
			"Extended Art MagicConプロモ": {
				Edition: "PSPM",
				Variant: "3a",
			},
		},
	},
	"MagicFest": {
		"Lightning Bolt": {
			"": {
				Edition: "PF19",
				Variant: "1",
			},
		},
	},
	"Judge Foil": {
		"Demonic Tutor": {
			"2020年版": {
				Edition: "J20",
				Variant: "4",
			},
			"2008年版ジャッジ褒賞": {
				Edition: "G08",
				Variant: "3",
			},
		},
		"Vindicate": {
			"2007年版ジャッジ褒賞": {
				Edition: "G07",
				Variant: "4",
			},
			"2013年版ジャッジ褒賞": {
				Edition: "G13",
				Variant: "7",
			},
		},
		"Wasteland": {
			"2010Ver. 2010年版ジャッジ褒賞": {
				Edition: "G10",
				Variant: "8",
			},
			"2015Ver. 2015年版ジャッジ褒賞": {
				Edition: "J15",
				Variant: "8",
			},
		},
	},
	"Game Day Promos": {
		"Mutavault": {
			"ゲームデー": {
				Edition: "PCMP",
				Variant: "12",
			},
		},
		"Serra Avenger": {
			"ゲームデー": {
				Edition: "PCMP",
				Variant: "6",
			},
		},
	},
	"P30A": {
		"Arcane Signet": {
			"30周年プロモ": {
				Edition: "P30M",
				Variant: "1F",
			},
		},
	},
	"SLP": {
		"Lightning Bolt": {
			"": {
				Edition: "SLP",
				Variant: "37",
			},
		},
	},
	"": {
		"Celestine Reef": {
			"その他プロモ": {
				Edition: "DCI",
				Variant: "42",
			},
		},
	},
	"Misprint": {
		"Laquatus's Champion": {
			"印刷ミス": {
				Edition: "PTOR",
				Variant: "67†a",
			},
		},
	},
}
