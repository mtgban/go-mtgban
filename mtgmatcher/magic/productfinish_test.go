package magic

import (
	"errors"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// productFinishFixture holds printings sold under a TCGplayer product the
// printing its id files at does not carry every finish of: 7th Edition's
// star foils, printings of their own sold under their card's product - Raise
// Dead's id files at the nonfoil, Scathe Zombies' at the foil - beside the
// Chinese alt-arts; Fate Reforged's star foil, with no product of its own;
// Aether Revolt's Alley Strangler, one product with its starter deck
// printing; the etched foils Strixhaven, Double Masters 2022 and Secret Lair
// sell as products of their own; and the surge foil the loader files apart
// from Meteor Golem. Every row is the datastore's, cut to what the loader
// reads.
const productFinishFixture = `{"data": {
	"7ED": {"baseSetSize": 350, "code": "7ED", "isFoilOnly": false, "isOnlineOnly": false, "keyruneCode": "7ED", "name": "Seventh Edition", "releaseDate": "2001-04-11", "totalSetSize": 708, "type": "core", "cards": [
		{"availability": ["mtgo", "paper"], "borderColor": "white", "finishes": ["nonfoil"], "frameVersion": "1997", "identifiers": {"scryfallId": "929cdbb4-d8b3-4e01-918c-2f46d74bb455", "tcgplayerProductId": "3040"}, "language": "English", "layout": "normal", "name": "Raise Dead", "number": "157", "rarity": "common", "setCode": "7ED", "type": "Sorcery", "types": ["Sorcery"], "uuid": "25fdc9a4-fe3c-5a51-b09c-d2b3420ec619", "variations": ["b41379c9-ad2a-5d4a-973b-d6aabe1934a7", "f7731317-6e2c-587a-80b0-304df2d2973b", "f9223895-3828-5c5b-8d8e-6258a9753c38"]},
		{"availability": ["mtgo", "paper"], "borderColor": "white", "finishes": ["nonfoil"], "frameVersion": "1997", "identifiers": {"scryfallId": "d936478a-c9a9-45b6-afd7-de30f1c3221f"}, "language": "Chinese Simplified", "layout": "normal", "name": "Raise Dead", "number": "157s", "promoTypes": ["schinesealtart"], "rarity": "common", "setCode": "7ED", "type": "Sorcery", "types": ["Sorcery"], "uuid": "f9223895-3828-5c5b-8d8e-6258a9753c38", "variations": ["25fdc9a4-fe3c-5a51-b09c-d2b3420ec619", "b41379c9-ad2a-5d4a-973b-d6aabe1934a7", "f7731317-6e2c-587a-80b0-304df2d2973b"]},
		{"availability": ["mtgo", "paper"], "borderColor": "black", "finishes": ["foil"], "frameVersion": "1997", "identifiers": {"scryfallId": "062f3566-ff6a-4f2a-9896-9547e46e6bac", "tcgplayerProductId": "3040"}, "language": "English", "layout": "normal", "name": "Raise Dead", "number": "157★", "rarity": "common", "setCode": "7ED", "type": "Sorcery", "types": ["Sorcery"], "uuid": "f7731317-6e2c-587a-80b0-304df2d2973b", "variations": ["25fdc9a4-fe3c-5a51-b09c-d2b3420ec619", "b41379c9-ad2a-5d4a-973b-d6aabe1934a7", "f9223895-3828-5c5b-8d8e-6258a9753c38"]},
		{"availability": ["mtgo", "paper"], "borderColor": "black", "finishes": ["foil"], "frameVersion": "1997", "identifiers": {"scryfallId": "3deb1c2e-49ae-4f26-9f7d-6643cb79eea5"}, "language": "Chinese Simplified", "layout": "normal", "name": "Raise Dead", "number": "157★s", "promoTypes": ["schinesealtart"], "rarity": "common", "setCode": "7ED", "type": "Sorcery", "types": ["Sorcery"], "uuid": "b41379c9-ad2a-5d4a-973b-d6aabe1934a7", "variations": ["25fdc9a4-fe3c-5a51-b09c-d2b3420ec619", "f7731317-6e2c-587a-80b0-304df2d2973b", "f9223895-3828-5c5b-8d8e-6258a9753c38"]},
		{"availability": ["mtgo", "paper"], "borderColor": "white", "finishes": ["nonfoil"], "frameVersion": "1997", "identifiers": {"scryfallId": "ed515f6f-432e-4455-a871-5cefdd15a37c", "tcgplayerProductId": "3064"}, "language": "English", "layout": "normal", "name": "Scathe Zombies", "number": "161", "rarity": "common", "setCode": "7ED", "type": "Creature — Zombie", "types": ["Creature"], "uuid": "f7f1ae2c-2abb-5387-b2b0-587855c6fdc5", "variations": ["11b1aeee-119a-5a76-b5f1-4fbdf732fa72", "338f7cd3-0bf3-535b-8728-12f15f3279ce", "3f6f0f8d-b464-5b33-bf09-84a2bcfdd12b"]},
		{"availability": ["mtgo", "paper"], "borderColor": "white", "finishes": ["nonfoil"], "frameVersion": "1997", "identifiers": {"scryfallId": "a7167902-939f-4aae-819a-8b941ce45cb1"}, "language": "Chinese Simplified", "layout": "normal", "name": "Scathe Zombies", "number": "161s", "promoTypes": ["schinesealtart"], "rarity": "common", "setCode": "7ED", "type": "Creature — Zombie", "types": ["Creature"], "uuid": "338f7cd3-0bf3-535b-8728-12f15f3279ce", "variations": ["11b1aeee-119a-5a76-b5f1-4fbdf732fa72", "3f6f0f8d-b464-5b33-bf09-84a2bcfdd12b", "f7f1ae2c-2abb-5387-b2b0-587855c6fdc5"]},
		{"availability": ["mtgo", "paper"], "borderColor": "black", "finishes": ["foil"], "frameVersion": "1997", "identifiers": {"scryfallId": "88bddc0b-f000-4679-bbfd-ab7674e30343", "tcgplayerProductId": "3064"}, "language": "English", "layout": "normal", "name": "Scathe Zombies", "number": "161★", "rarity": "common", "setCode": "7ED", "type": "Creature — Zombie", "types": ["Creature"], "uuid": "3f6f0f8d-b464-5b33-bf09-84a2bcfdd12b", "variations": ["11b1aeee-119a-5a76-b5f1-4fbdf732fa72", "338f7cd3-0bf3-535b-8728-12f15f3279ce", "f7f1ae2c-2abb-5387-b2b0-587855c6fdc5"]},
		{"availability": ["mtgo", "paper"], "borderColor": "black", "finishes": ["foil"], "frameVersion": "1997", "identifiers": {"scryfallId": "6be514ca-dd8a-4b84-9125-03a0bf8a047d"}, "language": "Chinese Simplified", "layout": "normal", "name": "Scathe Zombies", "number": "161★s", "promoTypes": ["schinesealtart"], "rarity": "common", "setCode": "7ED", "type": "Creature — Zombie", "types": ["Creature"], "uuid": "11b1aeee-119a-5a76-b5f1-4fbdf732fa72", "variations": ["338f7cd3-0bf3-535b-8728-12f15f3279ce", "3f6f0f8d-b464-5b33-bf09-84a2bcfdd12b", "f7f1ae2c-2abb-5387-b2b0-587855c6fdc5"]}
	]},
	"FRF": {"baseSetSize": 185, "code": "FRF", "isFoilOnly": false, "isOnlineOnly": false, "keyruneCode": "FRF", "name": "Fate Reforged", "releaseDate": "2015-01-23", "totalSetSize": 191, "type": "expansion", "cards": [
		{"availability": ["mtgo", "paper"], "borderColor": "black", "finishes": ["nonfoil"], "frameVersion": "2015", "identifiers": {"scryfallId": "e1d45374-a41b-4b3f-a7c8-3eb5ca767cf6", "tcgplayerProductId": "95037"}, "language": "English", "layout": "normal", "name": "Crux of Fate", "number": "65", "rarity": "rare", "setCode": "FRF", "type": "Sorcery", "types": ["Sorcery"], "uuid": "1b019d3c-63a9-5aa7-93d1-05db927e97db", "variations": ["fa59488e-a199-58cb-92bd-ff0e29c92278"]},
		{"availability": ["mtgo", "paper"], "borderColor": "black", "finishes": ["foil"], "frameVersion": "2015", "identifiers": {"scryfallId": "e92c13a4-c077-4f9b-ba0f-9fc81b53e58a"}, "language": "English", "layout": "normal", "name": "Crux of Fate", "number": "65★", "rarity": "rare", "setCode": "FRF", "type": "Sorcery", "types": ["Sorcery"], "uuid": "fa59488e-a199-58cb-92bd-ff0e29c92278", "variations": ["1b019d3c-63a9-5aa7-93d1-05db927e97db"]}
	]},
	"AER": {"baseSetSize": 184, "code": "AER", "isFoilOnly": false, "isOnlineOnly": false, "keyruneCode": "AER", "name": "Aether Revolt", "releaseDate": "2017-01-20", "totalSetSize": 197, "type": "expansion", "cards": [
		{"availability": ["mtgo", "paper"], "borderColor": "black", "finishes": ["foil", "nonfoil"], "frameVersion": "2015", "identifiers": {"scryfallId": "a131d558-5f6b-448b-a378-1882e2d02bd2", "tcgplayerProductId": "126455"}, "language": "English", "layout": "normal", "name": "Alley Strangler", "number": "52", "rarity": "common", "setCode": "AER", "type": "Creature — Aetherborn Rogue", "types": ["Creature"], "uuid": "62cdb518-f3d5-55a6-a6c3-83c2aca18af5", "variations": ["24d3dea3-444a-5bbe-bc8d-733fd1d4a9c5"]},
		{"availability": ["mtgo", "paper"], "borderColor": "black", "finishes": ["nonfoil"], "frameVersion": "2015", "identifiers": {"scryfallId": "d0b97fa0-4c5b-4211-9c8d-99197dc20636", "tcgplayerProductId": "126455"}, "language": "English", "layout": "normal", "name": "Alley Strangler", "number": "52†", "promoTypes": ["starterdeck"], "rarity": "common", "setCode": "AER", "type": "Creature — Aetherborn Rogue", "types": ["Creature"], "uuid": "24d3dea3-444a-5bbe-bc8d-733fd1d4a9c5", "variations": ["62cdb518-f3d5-55a6-a6c3-83c2aca18af5"]}
	]},
	"STA": {"baseSetSize": 63, "code": "STA", "isFoilOnly": false, "isOnlineOnly": false, "keyruneCode": "STA", "name": "Strixhaven Mystical Archive", "parentCode": "STX", "releaseDate": "2021-04-23", "totalSetSize": 126, "type": "masterpiece", "cards": [
		{"availability": ["arena", "mtgo", "paper"], "borderColor": "borderless", "finishes": ["etched", "foil", "nonfoil"], "frameVersion": "2015", "identifiers": {"scryfallId": "cc9ece2f-7eda-4fc5-a562-3e16e71560e9", "tcgplayerEtchedProductId": "233370", "tcgplayerProductId": "233369"}, "language": "English", "layout": "normal", "name": "Swords to Plowshares", "number": "10", "rarity": "rare", "setCode": "STA", "type": "Instant", "types": ["Instant"], "uuid": "0590be09-b330-5ebe-b4bb-4fa5d2b3906d", "variations": ["54146218-c228-5e3f-b7de-58fe8df5aabd"]}
	]},
	"2X2": {"baseSetSize": 331, "code": "2X2", "isFoilOnly": false, "isOnlineOnly": false, "keyruneCode": "2X2", "name": "Double Masters 2022", "releaseDate": "2022-07-08", "totalSetSize": 579, "type": "masters", "cards": [
		{"availability": ["paper"], "borderColor": "black", "finishes": ["etched"], "frameVersion": "2015", "identifiers": {"scryfallId": "e1721464-f9e5-434a-b8e6-9c8157199264", "tcgplayerEtchedProductId": "277297"}, "language": "English", "layout": "normal", "name": "Lord of Extinction", "number": "518", "promoTypes": ["boosterfun"], "rarity": "mythic", "setCode": "2X2", "type": "Creature — Elemental", "types": ["Creature"], "uuid": "5e9b2e4b-da3a-5692-83b8-d4def2f7f958", "variations": ["10020527-fbad-500a-9996-219202ed797b"]}
	]},
	"SLD": {"baseSetSize": 1, "code": "SLD", "isFoilOnly": false, "isOnlineOnly": false, "keyruneCode": "PMEI", "name": "Secret Lair Drop", "releaseDate": "2019-12-02", "totalSetSize": 2704, "type": "box", "cards": [
		{"availability": ["mtgo", "paper"], "borderColor": "black", "finishes": ["foil", "nonfoil"], "frameVersion": "2015", "identifiers": {"scryfallId": "27197660-8489-419b-9ad6-29a8713e4673", "tcgplayerEtchedProductId": "251773", "tcgplayerProductId": "251772"}, "language": "English", "layout": "normal", "name": "Demonlord Belzenlok", "number": "159", "rarity": "mythic", "setCode": "SLD", "type": "Legendary Creature — Elder Demon", "types": ["Creature"], "uuid": "09be9392-39b1-5901-bcc5-0bdfd73f3fb0", "variations": ["e771d473-2b2e-54c9-824e-1e309cdf42ec"]},
		{"availability": ["paper"], "borderColor": "black", "finishes": ["etched"], "frameVersion": "2015", "identifiers": {"scryfallId": "5d58cb4d-2091-40c8-b97c-09bf9c022a8b", "tcgplayerEtchedProductId": "251773"}, "language": "English", "layout": "normal", "name": "Demonlord Belzenlok", "number": "159★", "rarity": "mythic", "setCode": "SLD", "type": "Legendary Creature — Elder Demon", "types": ["Creature"], "uuid": "e771d473-2b2e-54c9-824e-1e309cdf42ec", "variations": ["09be9392-39b1-5901-bcc5-0bdfd73f3fb0"]}
	]},
	"MSC": {"baseSetSize": 866, "code": "MSC", "isFoilOnly": false, "isOnlineOnly": false, "keyruneCode": "MSC", "name": "Marvel Super Heroes Commander", "parentCode": "MSH", "releaseDate": "2026-06-26", "totalSetSize": 866, "type": "commander", "cards": [
		{"availability": ["mtgo", "paper"], "borderColor": "black", "finishes": ["foil", "nonfoil"], "frameVersion": "2015", "identifiers": {"scryfallId": "a1a1cfec-72ca-499b-b621-fa2d8502ea20", "tcgplayerAlternativeFoilProductId": "698282", "tcgplayerProductId": "698215"}, "language": "English", "layout": "normal", "name": "Meteor Golem", "number": "205", "promoTypes": ["surgefoil", "universesbeyond"], "rarity": "uncommon", "setCode": "MSC", "type": "Artifact Creature — Golem", "types": ["Artifact", "Creature"], "uuid": "f05e1a10-75dd-5beb-8760-02260e219a97"}
	]}
}}`

// TestMatchIDFinishProduct pins that a vendor's product id answers every
// finish its product is sold in, whichever printing the id files at, and no
// finish it is not: TCGplayer prices a sku by product and printing, and the
// printing is only ever Normal or Foil.
func TestMatchIDFinishProduct(t *testing.T) {
	b, err := Load(strings.NewReader(productFinishFixture))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		desc   string
		id     string
		finish string
		want   string
		err    error
	}{
		{
			desc:   "a card's product answers its nonfoil",
			id:     "3040",
			finish: "Normal",
			want:   "25fdc9a4-fe3c-5a51-b09c-d2b3420ec619",
		},
		{
			desc:   "and its star foil, not the alt-art starred beside it",
			id:     "3040",
			finish: "Foil",
			want:   "f7731317-6e2c-587a-80b0-304df2d2973b",
		},
		{
			desc:   "whichever of the two the id files at",
			id:     "3064",
			finish: "Normal",
			want:   "f7f1ae2c-2abb-5387-b2b0-587855c6fdc5",
		},
		{
			desc:   "a star foil with no product of its own is its card's",
			id:     "95037",
			finish: "Foil",
			want:   "fa59488e-a199-58cb-92bd-ff0e29c92278",
		},
		{
			desc:   "but an etched star twin is not, being sold as a product of its own",
			id:     "251772",
			finish: "Etched",
			err:    mtgmatcher.ErrCardWrongFinish,
		},
		{
			desc:   "a product two printings share answers from the one sold in the finish",
			id:     "126455",
			finish: "Foil",
			want:   "62cdb518-f3d5-55a6-a6c3-83c2aca18af5_f",
		},
		{
			desc:   "and from the one the id files at where that one is",
			id:     "126455",
			finish: "Normal",
			want:   "24d3dea3-444a-5bbe-bc8d-733fd1d4a9c5",
		},
		{
			desc:   "an etched product's foil is the etched printing",
			id:     "233370",
			finish: "Foil",
			want:   "0590be09-b330-5ebe-b4bb-4fa5d2b3906d_e",
		},
		{
			desc:   "beside the foil the card's own product sells",
			id:     "233369",
			finish: "Foil",
			want:   "0590be09-b330-5ebe-b4bb-4fa5d2b3906d_f",
		},
		{
			desc:   "a printing sold only etched answers foil with it",
			id:     "277297",
			finish: "Foil",
			want:   "5e9b2e4b-da3a-5692-83b8-d4def2f7f958",
		},
		{
			desc:   "by its Scryfall id too",
			id:     "e1721464-f9e5-434a-b8e6-9c8157199264",
			finish: "Foil",
			want:   "5e9b2e4b-da3a-5692-83b8-d4def2f7f958",
		},
		{
			desc:   "a Scryfall id names the card a surge foil is filed apart from",
			id:     "a1a1cfec-72ca-499b-b621-fa2d8502ea20",
			finish: "Foil",
			want:   "f05e1a10-75dd-5beb-8760-02260e219a97_f",
		},
		{
			desc:   "the surge foil's product sells no nonfoil, though the card carries its id",
			id:     "698282",
			finish: "Normal",
			err:    mtgmatcher.ErrCardWrongFinish,
		},
		{
			desc:   "a uuid names one printing and never reaches the one filed apart",
			id:     "f05e1a10-75dd-5beb-8760-02260e219a97",
			finish: "Foil",
			err:    mtgmatcher.ErrCardWrongFinish,
		},
		{
			desc:   "and an etched uuid promotes to its own plain foil",
			id:     "0590be09-b330-5ebe-b4bb-4fa5d2b3906d_e",
			finish: "Foil",
			want:   "0590be09-b330-5ebe-b4bb-4fa5d2b3906d_f",
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := b.MatchIDFinish(tt.id, tt.finish)
			if !errors.Is(err, tt.err) {
				t.Fatalf("MatchIDFinish(%s, %s) = %s, %v, want error %v", tt.id, tt.finish, got, err, tt.err)
			}
			if got != tt.want {
				t.Errorf("MatchIDFinish(%s, %s) = %s, want %s", tt.id, tt.finish, got, tt.want)
			}
			// A listing sends the same through Match, with the flag its
			// printing implies
			in := mtgmatcher.InputCard{ID: tt.id, Finish: tt.finish, Foil: tt.finish != "Normal"}
			got, err = b.Match(&in)
			if !errors.Is(err, tt.err) || got != tt.want {
				t.Errorf("Match(%s, %s) = %s, %v, want %s, %v", tt.id, tt.finish, got, err, tt.want, tt.err)
			}
		})
	}
}
