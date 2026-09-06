package fleshandblood

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// labelFixture is the published datastore cut down to the printings these
// tests turn on, every row copied verbatim from it: the marvels numbered
// under a label ("DTD193-CF", "ROS001-MV", "IAR145-MV") beside the plain
// printings wearing the number, the Helm of Isen's Peak product numbered
// "WTR042-C" in every finish beside the plain Unlimited rows, the three
// lettered cold foils of Lightning Flow, the pitch-labelled Drop of Dragon
// Blood beside its extended art, the unnumbered counter card, the fused
// tokens a storefront spells face by face, the cold foils of Ash // Aether
// Ashwing numbered by one face, and the two backs of Marlynn sharing a
// pair.
const labelFixture = `{
	"game": "fleshandblood",
	"sets": {
		"DTD": {"name": "Dusk till Dawn", "releaseDate": "2023-07-14"},
		"ROS": {"name": "Rosetta", "releaseDate": "2024-09-20"},
		"OMN": {"name": "Omens of the Third Age", "releaseDate": "2026-06-05"},
		"IAR": {"name": "Usurp the Shadow Throne", "releaseDate": "2026-09-25"},
		"GEM-24720": {"name": "Gem Pack 5", "releaseDate": "2026-06-05"},
		"TH": {"name": "The Hunted", "releaseDate": "2025-01-31"},
		"ADMN": {"name": "Armory Deck: Maxx Nitro", "releaseDate": "2025-04-17"},
		"SUP": {"name": "Super Slam", "releaseDate": "2025-09-26"},
		"HS": {"name": "High Seas", "releaseDate": "2025-06-06"},
		"UPR": {"name": "Uprising", "releaseDate": "2022-07-01"},
		"WTR": {"name": "Welcome to Rathe", "releaseDate": "2019-10-11"}
	},
	"cards": [
		{"externalLinks": {"fabId": "DTD193", "tcgPlayerId": 502936}, "fabId": "DTD193", "finish": "Cold Foil", "id": "dtd193-cf_502936_cold", "image": "https://tcgplayer-cdn.tcgplayer.com/product/502936_400w.jpg", "name": "Nasreth, the Soul Harrower (Marvel)", "number": "DTD193-CF", "rarity": "Marvel", "setCode": "DTD"},
		{"externalLinks": {"fabId": "DTD193", "tcgPlayerId": 502937}, "fabId": "DTD193", "finish": "Normal", "id": "dtd193_502937", "image": "https://tcgplayer-cdn.tcgplayer.com/product/502937_400w.jpg", "name": "Nasreth, the Soul Harrower", "number": "DTD193", "rarity": "Common", "setCode": "DTD"},
		{"externalLinks": {"fabId": "ROS001", "tcgPlayerId": 561243}, "fabId": "ROS001", "finish": "Normal", "id": "ros001_561243", "image": "https://tcgplayer-cdn.tcgplayer.com/product/561243_400w.jpg", "name": "Florian, Rotwood Harbinger", "number": "ROS001", "rarity": "Majestic", "setCode": "ROS"},
		{"externalLinks": {"fabId": "ROS001", "tcgPlayerId": 561247}, "fabId": "ROS001", "finish": "Cold Foil", "id": "ros001-mv_561247_cold", "image": "https://tcgplayer-cdn.tcgplayer.com/product/561247_400w.jpg", "name": "Florian, Rotwood Harbinger (Marvel)", "number": "ROS001-MV", "rarity": "Marvel", "setCode": "ROS"},
		{"externalLinks": {"tcgPlayerId": 696172}, "finish": "Cold Foil", "id": "iar145-mv_696172_cold", "image": "https://tcgplayer-cdn.tcgplayer.com/product/696172_400w.jpg", "name": "Runechant of Greed (Yellow) (Marvel)", "number": "IAR145-MV", "rarity": "Marvel", "setCode": "OMN"},
		{"externalLinks": {"tcgPlayerId": 706701}, "finish": "Normal", "id": "iar145_706701", "image": "https://tcgplayer-cdn.tcgplayer.com/product/706701_400w.jpg", "name": "Runechant of Greed", "number": "IAR145", "rarity": "Majestic", "setCode": "IAR"},
		{"externalLinks": {"tcgPlayerId": 696218}, "finish": "Rainbow Foil", "id": "gem177_696218_rainbow", "image": "https://tcgplayer-cdn.tcgplayer.com/product/696218_400w.jpg", "name": "Runechant of Greed (Yellow)", "number": "GEM177", "rarity": "Promo", "setCode": "GEM-24720"},
		{"externalLinks": {"fabId": "OMN203", "tcgPlayerId": 682858}, "fabId": "OMN203", "finish": "Cold Foil", "id": "omn203_682858_cold", "image": "https://tcgplayer-cdn.tcgplayer.com/product/682858_400w.jpg", "name": "Lightning Flow", "number": "OMN203", "promoTypes": ["c"], "rarity": "Basic", "setCode": "OMN", "variant": "C"},
		{"externalLinks": {"fabId": "OMN203", "tcgPlayerId": 682883}, "fabId": "OMN203", "finish": "Cold Foil", "id": "omn203_682883_cold", "image": "https://tcgplayer-cdn.tcgplayer.com/product/682883_400w.jpg", "name": "Lightning Flow", "number": "OMN203", "promoTypes": ["b"], "rarity": "Basic", "setCode": "OMN", "variant": "B"},
		{"externalLinks": {"fabId": "OMN203", "tcgPlayerId": 682884}, "fabId": "OMN203", "finish": "Cold Foil", "id": "omn203_682884_cold", "image": "https://tcgplayer-cdn.tcgplayer.com/product/682884_400w.jpg", "name": "Lightning Flow", "number": "OMN203", "promoTypes": ["a"], "rarity": "Basic", "setCode": "OMN", "variant": "A"},
		{"externalLinks": {"fabId": "OMN203", "tcgPlayerId": 682889}, "fabId": "OMN203", "finish": "Normal", "id": "omn203_682889", "image": "https://tcgplayer-cdn.tcgplayer.com/product/682889_400w.jpg", "name": "Lightning Flow", "number": "OMN203", "rarity": "Basic", "setCode": "OMN"},
		{"externalLinks": {"fabId": "HNT155", "tcgPlayerId": 611179}, "fabId": "HNT155", "finish": "Normal", "id": "hnt155_611179", "image": "https://tcgplayer-cdn.tcgplayer.com/product/611179_400w.jpg", "name": "Drop of Dragon Blood", "number": "HNT155", "promoTypes": ["red"], "rarity": "Rare", "setCode": "TH", "variant": "Red"},
		{"externalLinks": {"fabId": "HNT155", "tcgPlayerId": 611179}, "fabId": "HNT155", "finish": "Rainbow Foil", "id": "hnt155_611179_rainbow", "image": "https://tcgplayer-cdn.tcgplayer.com/product/611179_400w.jpg", "name": "Drop of Dragon Blood", "number": "HNT155", "promoTypes": ["red"], "rarity": "Rare", "setCode": "TH", "variant": "Red"},
		{"externalLinks": {"fabId": "HNT155", "tcgPlayerId": 614550}, "fabId": "HNT155", "finish": "Rainbow Foil", "id": "hnt155_614550_rainbow", "image": "https://tcgplayer-cdn.tcgplayer.com/product/614550_400w.jpg", "name": "Drop of Dragon Blood", "number": "HNT155", "promoTypes": ["extended art"], "rarity": "Rare", "setCode": "TH", "variant": "Extended Art"},
		{"externalLinks": {"tcgPlayerId": 629875}, "finish": "Normal", "id": "629875", "image": "https://tcgplayer-cdn.tcgplayer.com/product/629875_400w.jpg", "name": "Steam Counter // Defense Counter", "rarity": "None", "setCode": "ADMN"},
		{"externalLinks": {"tcgPlayerId": 618208}, "finish": "Normal", "id": "hnt244-hnt167_618208", "image": "https://tcgplayer-cdn.tcgplayer.com/product/618208_400w.jpg", "name": "Marked // Fealty", "number": "HNT244//HNT167", "rarity": "Token", "setCode": "TH"},
		{"externalLinks": {"fabId": "HNT167", "tcgPlayerId": 618124}, "fabId": "HNT167", "finish": "Normal", "id": "hnt167-hnt053_618124", "image": "https://tcgplayer-cdn.tcgplayer.com/product/618124_400w.jpg", "name": "Fealty // Graphene Chelicera", "number": "HNT167//HNT053", "rarity": "Token", "setCode": "TH"},
		{"externalLinks": {"tcgPlayerId": 617583}, "finish": "Normal", "id": "hnt167_617583", "image": "https://tcgplayer-cdn.tcgplayer.com/product/617583_400w.jpg", "name": "Fealty", "number": "HNT167", "rarity": "Token", "setCode": "TH"},
		{"externalLinks": {"fabId": "HNT167", "tcgPlayerId": 617584}, "fabId": "HNT167", "finish": "Cold Foil", "id": "hnt167_617584_cold", "image": "https://tcgplayer-cdn.tcgplayer.com/product/617584_400w.jpg", "name": "Fealty", "number": "HNT167", "promoTypes": ["marvel"], "rarity": "Marvel", "setCode": "TH", "variant": "Marvel"},
		{"externalLinks": {"fabId": "HNT244"}, "fabId": "HNT244", "finish": "Normal", "id": "hnt244", "image": "https://legendstory-production-s3-public.s3.amazonaws.com/media/cards/large/HNT244.webp", "name": "Marked", "number": "HNT244", "rarity": "Token", "setCode": "TH"},
		{"externalLinks": {"tcgPlayerId": 656557}, "finish": "Normal", "id": "sup242-sup239-sup240-sup241_656557", "image": "https://tcgplayer-cdn.tcgplayer.com/product/656557_400w.jpg", "name": "Vigor // Confidence // Might // Toughness", "number": "SUP242//SUP239//SUP240//SUP241", "rarity": "Basic", "setCode": "SUP"},
		{"externalLinks": {"fabId": "SEA042", "tcgPlayerId": 638043}, "fabId": "SEA042", "finish": "Normal", "id": "sea042-sea244_638043", "image": "https://tcgplayer-cdn.tcgplayer.com/product/638043_400w.jpg", "name": "Golden Cog // Gold", "number": "SEA042//SEA244", "rarity": "Basic", "setCode": "HS"},
		{"externalLinks": {"fabId": "SEA042", "tcgPlayerId": 624371}, "fabId": "SEA042", "finish": "Rainbow Foil", "id": "sea042_624371_rainbow", "image": "https://tcgplayer-cdn.tcgplayer.com/product/624371_400w.jpg", "name": "Golden Cog", "number": "SEA042", "promoTypes": ["treasure"], "rarity": "Pirate Booty", "setCode": "HS", "variant": "Treasure"},
		{"externalLinks": {"fabId": "SEA244", "tcgPlayerId": 624373}, "fabId": "SEA244", "finish": "Rainbow Foil", "id": "sea244_624373_rainbow", "image": "https://tcgplayer-cdn.tcgplayer.com/product/624373_400w.jpg", "name": "Gold", "number": "SEA244", "promoTypes": ["treasure"], "rarity": "Marvel", "setCode": "HS", "variant": "Treasure"},
		{"externalLinks": {"tcgPlayerId": 275748}, "finish": "Cold Foil", "id": "upr043_275748_cold", "image": "https://tcgplayer-cdn.tcgplayer.com/product/275748_400w.jpg", "name": "Ash // Aether Ashwing", "number": "UPR043", "promoTypes": ["cold foil"], "rarity": "Common", "setCode": "UPR", "variant": "Cold Foil"},
		{"externalLinks": {"tcgPlayerId": 275749}, "finish": "Cold Foil", "id": "upr043_275749_cold", "image": "https://tcgplayer-cdn.tcgplayer.com/product/275749_400w.jpg", "name": "Ash // Aether Ashwing", "number": "UPR043", "promoTypes": ["marvel"], "rarity": "Marvel", "setCode": "UPR", "variant": "Marvel"},
		{"externalLinks": {"tcgPlayerId": 275840}, "finish": "Normal", "id": "upr042-upr043_275840", "image": "https://tcgplayer-cdn.tcgplayer.com/product/275840_400w.jpg", "name": "Aether Ashwing // Ash", "number": "UPR042//UPR043", "rarity": "Token", "setCode": "UPR"},
		{"externalLinks": {"fabId": "UPR042"}, "fabId": "UPR042", "finish": "Normal", "id": "upr042", "image": "https://legendstory-production-s3-public.s3.amazonaws.com/media/cards/large/UPR042.webp", "name": "Aether Ashwing", "number": "UPR042", "rarity": "Token", "setCode": "UPR"},
		{"externalLinks": {"fabId": "UPR042"}, "fabId": "UPR042", "finish": "Cold Foil", "id": "upr042_cold", "image": "https://legendstory-production-s3-public.s3.amazonaws.com/media/cards/large/UPR042.webp", "name": "Aether Ashwing", "number": "UPR042", "rarity": "Token", "setCode": "UPR"},
		{"externalLinks": {"fabId": "WTR042", "tcgPlayerId": 225064}, "fabId": "WTR042", "finish": "1st Edition Normal", "id": "wtr042-c_225064_1e", "image": "https://tcgplayer-cdn.tcgplayer.com/product/225064_400w.jpg", "name": "Helm of Isen's Peak", "number": "WTR042-C", "rarity": "Common", "setCode": "WTR"},
		{"externalLinks": {"fabId": "WTR042", "tcgPlayerId": 225064}, "fabId": "WTR042", "finish": "1st Edition Cold Foil", "id": "wtr042-c_225064_1ecold", "image": "https://tcgplayer-cdn.tcgplayer.com/product/225064_400w.jpg", "name": "Helm of Isen's Peak", "number": "WTR042-C", "rarity": "Common", "setCode": "WTR"},
		{"externalLinks": {"fabId": "WTR042", "tcgPlayerId": 225064}, "fabId": "WTR042", "finish": "Unlimited Edition Normal", "id": "wtr042-c_225064_unl", "image": "https://tcgplayer-cdn.tcgplayer.com/product/225064_400w.jpg", "name": "Helm of Isen's Peak", "number": "WTR042-C", "rarity": "Common", "setCode": "WTR"},
		{"externalLinks": {"fabId": "WTR042", "tcgPlayerId": 225064}, "fabId": "WTR042", "finish": "Unlimited Edition Rainbow Foil", "id": "wtr042-c_225064_unlrainbow", "image": "https://tcgplayer-cdn.tcgplayer.com/product/225064_400w.jpg", "name": "Helm of Isen's Peak", "number": "WTR042-C", "rarity": "Common", "setCode": "WTR"},
		{"externalLinks": {"fabId": "WTR042"}, "fabId": "WTR042", "finish": "Unlimited Edition Normal", "id": "wtr042_unl", "image": "https://legendstory-production-s3-public.s3.amazonaws.com/media/cards/large/WTR042.webp", "name": "Helm of Isen's Peak", "number": "WTR042", "rarity": "Common", "setCode": "WTR"},
		{"externalLinks": {"fabId": "WTR042"}, "fabId": "WTR042", "finish": "Unlimited Edition Rainbow Foil", "id": "wtr042_unlrainbow", "image": "https://legendstory-production-s3-public.s3.amazonaws.com/media/cards/large/WTR042.webp", "name": "Helm of Isen's Peak", "number": "WTR042", "rarity": "Common", "setCode": "WTR"},
		{"externalLinks": {"tcgPlayerId": 635619}, "finish": "Normal", "id": "sea083-sea247_635619", "image": "https://tcgplayer-cdn.tcgplayer.com/product/635619_400w.jpg", "name": "Marlynn // Treasure Island", "number": "SEA083//SEA247", "rarity": "Basic", "setCode": "HS"},
		{"externalLinks": {"tcgPlayerId": 651892}, "finish": "Normal", "id": "sea083-sea247_651892", "image": "https://tcgplayer-cdn.tcgplayer.com/product/651892_400w.jpg", "name": "Marlynn // Arrows Back", "number": "SEA083//SEA247", "rarity": "Basic", "setCode": "HS"},
		{"externalLinks": {"fabId": "SEA083", "tcgPlayerId": 634589}, "fabId": "SEA083", "finish": "Cold Foil", "id": "sea083_634589_cold", "image": "https://tcgplayer-cdn.tcgplayer.com/product/634589_400w.jpg", "name": "Marlynn (Marvel)", "number": "SEA083", "rarity": "Marvel", "setCode": "HS"}
	]
}`

// TestLabelledNumbers pins how a number the catalog wrote under a label, a
// letter, a pitch or not at all still reaches the printing a storefront
// names by the plain number, the finish and the faces.
func TestLabelledNumbers(t *testing.T) {
	b, err := Load(strings.NewReader(labelFixture))
	if err != nil {
		t.Fatal(err)
	}
	mtgmatcher.SetGlobalDatastore(b)
	for _, tt := range []struct {
		desc                             string
		name, edition, variation, finish string
		foil                             bool
		want                             string
	}{
		{"a label stem reaches the marvel in the finish only it is sold in", "Nasreth, the Soul Harrower", "Dusk till Dawn", "DTD193", "Cold Foil", true, "dtd193-cf_502936_cold"},
		{"and the plain printing keeps the plain finish", "Nasreth, the Soul Harrower", "Dusk till Dawn", "DTD193", "Non-foil", false, "dtd193_502937"},
		{"a majestic sold plain hands its cold foil to the marvel beside it", "Florian, Rotwood Harbinger", "Rosetta", "ROS001", "Cold Foil", true, "ros001-mv_561247_cold"},
		{"and keeps its own plain printing", "Florian, Rotwood Harbinger", "Rosetta", "ROS001", "Non-foil", false, "ros001_561243"},
		{"a bare name grows a pitch and the label together", "Runechant of Greed", "Omens of the Third Age", "IAR145", "Cold Foil", true, "iar145-mv_696172_cold"},
		{"while the plain printing of the same number stays plain", "Runechant of Greed", "Usurp the Shadow Throne", "IAR145", "Non-foil", false, "iar145_706701"},
		{"a letter on the number names the lettered printing", "Lightning Flow", "Omens of the Third Age", "OMN203a", "Cold Foil", true, "omn203_682884_cold"},
		{"whichever letter", "Lightning Flow", "Omens of the Third Age", "OMN203c", "Cold Foil", true, "omn203_682858_cold"},
		{"and no letter is the plain printing", "Lightning Flow", "Omens of the Third Age", "OMN203", "Non-foil", false, "omn203_682889"},
		{"a pitch label on every printing tells none apart", "Drop of Dragon Blood", "The Hunted", "HNT155", "Rainbow Foil", true, "hnt155_611179_rainbow"},
		{"so the extended art is a variant of the pitch-labelled base", "Drop of Dragon Blood", "The Hunted", "HNT155 Extended Art", "Rainbow Foil", true, "hnt155_614550_rainbow"},
		{"an unnumbered printing cannot disagree with a number", "Defense Counter // Steam Counter", "Armory Deck: Maxx Nitro", "H01", "Non-foil", false, "629875"},
		{"a pair names the fused card whatever way the faces were written", "Fealty / Marked // Fealty / Marked", "The Hunted", "HNT167 // HNT244", "Non-foil", false, "hnt244-hnt167_618208"},
		{"with the letters a storefront hangs on the halves", "Gold / Golden Cog // Gold / Golden Cog", "High Seas", "SEA244a // SEA042a", "Non-foil", false, "sea042-sea244_638043"},
		{"and however many faces it has", "Might / Toughness // Vigor / Confidence", "Super Slam", "SUP240 // SUP241 // SUP242 // SUP239", "Non-foil", false, "sup242-sup239-sup240-sup241_656557"},
		{"the token spelling is the pair's own while sold plain", "Aether Ashwing // Ash", "Uprising", "UPR042 // UPR043", "Non-foil", false, "upr042-upr043_275840"},
		{"its cold foil is filed by one face, and the finish names the label", "Aether Ashwing // Ash", "Uprising", "UPR042b // UPR043b", "Cold Foil", true, "upr043_275748_cold"},
		{"and the marvel wording names the marvel", "Aether Ashwing // Ash", "Uprising", "UPR042 // UPR043 Marvel", "Cold Foil", true, "upr043_275749_cold"},
		{"the labelled product is the only 1st Edition of its number", "Helm of Isen's Peak", "Welcome to Rathe", "WTR042", "1st Edition Normal", false, "wtr042-c_225064_1e"},
		{"and the printing wearing the number outranks it where both are sold", "Helm of Isen's Peak", "Welcome to Rathe", "WTR042", "Unlimited Edition Normal", false, "wtr042_unl"},
	} {
		in := mtgmatcher.InputCard{Name: tt.name, Edition: tt.edition, Variation: tt.variation, Finish: tt.finish, Foil: tt.foil}
		got, err := mtgmatcher.Match(&in)
		if err != nil {
			t.Errorf("%s: Match(%q, %q, %q, %q) = %v", tt.desc, tt.name, tt.edition, tt.variation, tt.finish, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%s: Match(%q, %q, %q, %q) = %q, want %q", tt.desc, tt.name, tt.edition, tt.variation, tt.finish, got, tt.want)
		}
	}
	// Two fused cards share Marlynn's pair, one per back, and a listing
	// naming the face alone cannot say which it sells.
	in := mtgmatcher.InputCard{Name: "Marlynn", Edition: "High Seas", Variation: "SEA083", Finish: "Non-foil"}
	if got, err := mtgmatcher.Match(&in); err == nil {
		t.Errorf("Match(Marlynn, SEA083) = %q, want a refusal", got)
	}
}

func TestLabelStem(t *testing.T) {
	for _, tt := range []struct{ number, want string }{
		{"ROS001-MV", "ROS001"},
		{"DTD193-CF", "DTD193"},
		{"WTR042-C", "WTR042"},
		{"MST158-A", "MST158"},
		{"WTR042", "WTR042"},
		{"UPR042//UPR043", "UPR042//UPR043"},
		{"HNT2-155", "HNT2-155"},
	} {
		if got := labelStem(tt.number); got != tt.want {
			t.Errorf("labelStem(%q) = %q, want %q", tt.number, got, tt.want)
		}
	}
}
