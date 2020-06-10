package mtgdb

import (
	"fmt"
	"log"
	"os"
	"testing"
)

type MatchTest struct {
	Id   string
	Err  error
	Desc string
	In   Card
}

var MatchTests = []MatchTest{
	// Errors
	MatchTest{
		Desc: "card_does_not_exist",
		Err:  fmt.Errorf("card 'I do not exist' does not exist"),
		In: Card{
			Name: "I do not exist",
		},
	},
	MatchTest{
		Desc: "wrong_card_number",
		Err:  fmt.Errorf("edition 'Alliances' does not apply to 'Arcane Denial'"),
		In: Card{
			Name:      "Arcane Denial",
			Variation: "10",
			Edition:   "Alliances",
		},
	},
	MatchTest{
		Desc: "not_in_a_promo_pack",
		Err:  fmt.Errorf("edition 'Promo Pack' does not apply to 'Demonic Tutor'"),
		In: Card{
			Name:    "Demonic Tutor",
			Edition: "Promo Pack",
		},
	},
	MatchTest{
		Err:  fmt.Errorf("edition 'ignored' does not apply to 'Lobotomy'"),
		Desc: "not_a_prerelease",
		In: Card{
			Name:      "Lobotomy",
			Variation: "Prerelease",
			Edition:   "ignored",
		},
	},

	// ID lookup
	MatchTest{
		Id:   "7a0be3c2-accb-511c-bc97-96f9cd6eb1ea",
		Desc: "id_lookup",
		In: Card{
			Id: "7a0be3c2-accb-511c-bc97-96f9cd6eb1ea",
		},
	},
	MatchTest{
		Id:   "f5737cd9-b418-517b-9a69-705b8c1e402f",
		Desc: "id_lookup_scryfall",
		In: Card{
			Id: "281f6118-adb8-4a7d-9c77-5570f3399e6e",
		},
	},

	// Number duplicates
	MatchTest{
		Id:   "76a1a052-ea00-5ddf-9486-0da27cdccd6b",
		Desc: "full-art_land",
		In: Card{
			Name:      "Swamp",
			Variation: "241",
			Edition:   "Zendikar",
		},
	},
	MatchTest{
		Id:   "647f96d3-7d55-5052-aec2-a1f3b39008e0",
		Desc: "full-art_land_could_be_confused_with_suffix",
		In: Card{
			Name:      "Island",
			Variation: "234 A - Full Art",
			Edition:   "Zendikar",
		},
	},
	MatchTest{
		Id:   "e6c6a31a-7497-530f-a71f-3a8e3cd83a18",
		Desc: "non-full-art_land_could_be_confused_with_suffix",
		In: Card{
			Name:      "Forest",
			Variation: "274 A - Non-Full Art",
			Edition:   "Battle for Zendikar",
		},
	},
	MatchTest{
		Id:   "f793ed4a-0df6-5dcc-a217-5fa2f019a54e",
		Desc: "complex_number_variant",
		In: Card{
			Name:      "Plains",
			Variation: "87 - A",
			Edition:   "Unsanctioned",
		},
	},
	MatchTest{
		Id:   "df26e509-bf27-515a-975c-702647651870",
		Desc: "alternative_complex_number_variant",
		In: Card{
			Name:      "Brothers Yamazaki",
			Variation: "160 A",
			Edition:   "Champions of Kamigawa",
		},
	},
	MatchTest{
		Id:   "b84dceeb-d344-577e-9844-1b887b427e7d",
		Desc: "second_alternative_complex_number_variant",
		In: Card{
			Name:      "Brothers Yamazaki",
			Variation: "160 - B",
			Edition:   "Champions of Kamigawa",
		},
	},
	MatchTest{
		Id:   "db24e606-6288-548e-8df1-aa2438b1765e",
		Desc: "borderless_lands",
		In: Card{
			Name:    "Plains",
			Edition: "Unstable",
		},
	},
	MatchTest{
		Id:   "3e653dee-78ac-5112-8f72-a841b1cf3d76",
		Desc: "weekend_lands",
		In: Card{
			Name:      "Island",
			Variation: "Ravnica Weekend B02",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "8aeeabc4-6499-5560-a7ed-b49a92500645",
		Desc: "japanese_lands",
		In: Card{
			Name:    "Swamp",
			Edition: "Magic Premiere Shop 2006",
			Foil:    true,
		},
	},
	MatchTest{
		Id:   "c82d68bc-ed29-5c56-b3a5-dfe804e05a5d",
		Desc: "plains_from_set_with_special_cards_and_C_to_be_ignored",
		In: Card{
			Name:      "Plains",
			Variation: "366 C",
			Edition:   "Tenth Edition",
		},
	},
	MatchTest{
		Id:   "afeab69f-720e-507b-ba53-e4b7940d4e2c",
		Desc: "non_full_art_land_with_incomplete_number",
		In: Card{
			Name:      "Forest",
			Variation: "Non-Full Art 271",
			Edition:   "Battle for Zendikar",
		},
	},
	MatchTest{
		Id:   "704c923b-9da5-54a5-84a9-3498db36b1ed",
		Desc: "land_with_letter",
		In: Card{
			Name:    "Forest B",
			Edition: "Intl. Collectors' Edition",
		},
	},
	MatchTest{
		Id:   "334e1060-01bb-5480-b011-358e728e90dd",
		Desc: "forest_F",
		In: Card{
			Name:    "Forest F",
			Edition: "Battle Royale",
		},
	},
	MatchTest{
		Id:   "e6c6a31a-7497-530f-a71f-3a8e3cd83a18",
		Desc: "intro_land",
		In: Card{
			Name:      "Forest",
			Variation: "274 Intro",
			Edition:   "Battle for Zendikar",
		},
	},

	// Naming conventions
	MatchTest{
		Id:   "ea6011c0-59d0-55f8-97f1-83b4e5d4126d",
		Desc: "transform_card",
		In: Card{
			Name:    "Daybreak Ranger / Nightfall Predator",
			Edition: "Innistrad",
		},
	},
	MatchTest{
		Id:   "fd789e41-6bf4-544f-afd2-5cdec86e4a79",
		Desc: "aftermath_card",
		In: Card{
			Name:      "Commit to Memory",
			Variation: "Prerelease",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "bcef8350-ed57-5e3a-bffe-a3f9d955512e",
		Desc: "split_card",
		In: Card{
			Name:    "Fire // Ice",
			Edition: "Apocalypse",
		},
	},
	MatchTest{
		Id:   "ffe01c82-7082-5037-9ac0-36e3e6c8f386",
		Desc: "flip_card",
		In: Card{
			Name:      "Startled Awake / Persistent Nightmare",
			Variation: "Prerelease",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "16c1f28e-e61f-5971-8bd1-351be28f31d2",
		Desc: "meld_card",
		In: Card{
			Name:      "Bruna, the Fading Light / Brisela, Voice of Nightmares",
			Variation: "Prerelease",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "d816f77a-a10f-5cfa-8a01-cb5b9ca2b0b5",
		Desc: "meld_card_b",
		In: Card{
			Name:      "Gisela, the Broken Blade / Brisela, Voice of Nightmares",
			Variation: "Prerelease",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "f8f84c2c-b875-5960-803d-c07b2066fb99",
		Desc: "triple_card",
		In: Card{
			Name:    "Smelt // Herd // Saw",
			Edition: "Mystery Booster Playtest Ins",
		},
	},
	MatchTest{
		Id:   "ef8342d2-86b8-5f14-aee4-38b5a8ad61fa",
		Desc: "incorrect_name_but_salvageable",
		In: Card{
			Name:    "B.O.B.",
			Edition: "Unsanctioned",
		},
	},
	MatchTest{
		Id:   "8b65f1ba-afb9-5872-9cab-13a85de26287",
		Desc: "parenthesis_in_the_name",
		In: Card{
			Name:    "Erase (Not the Urza's Legacy One)",
			Edition: "Unhinged",
		},
	},
	MatchTest{
		Id:   "e53ae7f3-a0c0-5fb8-8753-c4917ab3a47b",
		Desc: "parenthesis_in_the_name_and_variation",
		In: Card{
			Name:      "B.F.M. (Big Furry Monster)",
			Variation: "29",
			Edition:   "Unglued",
		},
	},
	MatchTest{
		Id:   "57f22b61-f310-5fcc-bf5c-fdddb5b30467",
		Desc: "number_in_the_name",
		In: Card{
			Name:    "Serum Visions (30)",
			Edition: "Secret Lair Drop",
		},
	},
	MatchTest{
		Id:   "718043c6-ccf8-549c-b3d8-822e531874c8",
		Desc: "typo_in_split_card",
		In: Card{
			Name:    "Elbrus, The Binding Blade / Withengar Unbound",
			Edition: "Dark Ascension",
		},
	},

	// Incorrect editions
	MatchTest{
		Id:   "24d51f85-3ccc-5632-bf22-c7180de0cfd5",
		Desc: "incorrect_edition_but_card_has_a_single_printing",
		In: Card{
			Name:    "Mirrodin Besieged",
			Edition: "Scars of Phyrexia",
		},
	},
	MatchTest{
		Id:   "97d106d2-5270-5a8d-afc0-6e2fcd175edf",
		Desc: "incorrect_edition_set_name_does_not_interfere_with_number",
		In: Card{
			Name:      "Death Baron",
			Variation: "Convention Foil M19",
			Edition:   "ignored",
			Foil:      true,
		},
	},
	MatchTest{
		Id:   "25ee77c2-61c8-55ae-852c-72d7db1577da",
		Desc: "incorrect_edition_but_salvageable",
		In: Card{
			Name:    "Polluted Mire",
			Edition: "Duel Decks Anthology",
		},
	},
	MatchTest{
		Id:   "034b1941-b3a1-56ad-a0ef-09a11d57643e",
		Desc: "incorrect_edition_but_salvageable_and_could_alias_other_cards",
		In: Card{
			Name:    "Garruk Wildspeaker",
			Edition: "Duel Decks Anthology",
		},
	},
	MatchTest{
		Id:   "2ced4ecd-dfe9-51f7-8c4a-7f314fedfddf",
		Desc: "incorrect_edition_but_salvageable_missing_only_a_chunk",
		In: Card{
			Name:    "No One Will Hear Your Cries",
			Edition: "Archenemy: Nicol Bolas",
		},
	},
	MatchTest{
		Id:   "bdf4ee85-3af6-55b0-aecd-65456c733401",
		Desc: "incorrect_edition_belongs_to_a_foil-only_subset",
		In: Card{
			Name:      "Zur's Weirding",
			Variation: "Foil",
			Edition:   "Mystery Booster",
		},
	},
	MatchTest{
		Id:   "d1b7e6ed-5ce2-5a9a-89de-2b70188a4236",
		Desc: "incorrect_edition_year_should_not_interfere",
		In: Card{
			Name:      "Yule Ooze",
			Variation: "2011 Holiday",
			Edition:   "Happy Holidays",
		},
	},
	MatchTest{
		Id:   "007e7f41-7075-59be-a1d1-465425bfa877",
		Desc: "incorrect_edition_champs_and_states",
		In: Card{
			Name:      "Mutavault",
			Variation: "Extended art",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "202ee58e-c413-5edb-904b-2229491138c7",
		Desc: "champ_in_variant_but_not_champs_and_states",
		In: Card{
			Name:      "Champion of Lambholt",
			Variation: "commander-anthology-2018-champion-of-lambholt",
			Edition:   "Commander Anthology 2018",
		},
	},
	MatchTest{
		Id:   "9ee7e1b3-8661-51fc-a2cc-0d41afb5b6d0",
		Desc: "mismatching_year",
		In: Card{
			Name:      "Mountain",
			Variation: "Grand Prix 2018",
			Edition:   "MagicFest 2019",
			Foil:      true,
		},
	},

	// Promo pack
	MatchTest{
		Id:   "9cb505f0-4b5c-55a4-ba0d-958d43c4f537",
		Desc: "m20_promo_packs_lands",
		In: Card{
			Name:    "Plains",
			Edition: "Promo Pack",
		},
	},
	MatchTest{
		Id:   "77a62de3-9ca2-5d9c-aeca-1409a8173de5",
		Desc: "m20_promo_packs_lands_different",
		In: Card{
			Name:      "Swamp",
			Variation: "M20 Promo Pack",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "8e0090cc-ac0e-5ae0-8e23-c48bf8d9e0c6",
		Desc: "promo_pack_in_promos_with_pw_stamp",
		In: Card{
			Name:    "Zendikar Resurgent",
			Edition: "Promo Pack",
		},
	},
	MatchTest{
		Id:   "b464317f-89c9-54af-86eb-a345684cafd4",
		Desc: "promo_pack_in_expansion_with_inverted_frame",
		In: Card{
			Name:    "Alseid of Life's Bounty",
			Edition: "Promo Pack",
		},
	},
	MatchTest{
		Id:   "36180c43-5613-515d-82f2-b2153323586d",
		Desc: "promo_pack_in_promos_with_inverted_frame",
		In: Card{
			Name:    "Negate",
			Edition: "Promo Pack",
		},
	},
	MatchTest{
		Id:   "8b7a0e18-ca26-56f9-a44b-6694cd74cad7",
		Desc: "non-promo_pack_card_that_may_have_a_promo_pack_version",
		In: Card{
			Name:    "Slaying Fire",
			Edition: "Throne of Eldraine",
		},
	},
	MatchTest{
		Id:   "bf1c24e7-853e-57e5-8297-a41dceecf61b",
		Desc: "promo_pack_with_duplication",
		In: Card{
			Name:      "Sorcerous Spyglass",
			Variation: "Promo Pack XLN",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "c7bbbdab-1741-599c-a87c-562ce5f877c9",
		Desc: "non-promo_pack_is_fine_too",
		In: Card{
			Name:    "Sorcerous Spyglass",
			Edition: "XLN",
		},
	},

	// Prerelease
	MatchTest{
		Id:   "d86ac63d-c2d3-56bd-bee3-e40fc3f20a4b",
		Desc: "old_prerelease",
		In: Card{
			Name:      "Glory",
			Variation: "Prerelease",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "a7044e23-957a-5e51-86a9-317e6af07bea",
		Desc: "prerelease_in_promos_before_the_date_but_without_s_suffix",
		In: Card{
			Name:      "Scourge of Fleets",
			Variation: "Prerelease",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "443646fc-908f-5628-8b52-e4b15b07ecda",
		Desc: "prerelease_with_s_suffix",
		In: Card{
			Name:      "Pristine Skywise",
			Variation: "Prerelease",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "a4c63151-aa5f-55a7-af5b-c2c8d85f308e",
		Desc: "JPN_prerelease_with_s_suffix",
		In: Card{
			Name:      "Ugin, the Ineffable",
			Variation: "JPN Prerelease",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "b4718517-8fca-5940-8fb4-2eb77f4190f1",
		Desc: "JPN_prerelease_with_s_suffix_but_number_could_interfere",
		In: Card{
			Name:      "Teyo, the Shieldmage",
			Variation: "032 - JPN Alternate Art Prerelease Foil",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "543216f8-53b7-521f-acb0-3876a2690bde",
		Desc: "prerelease_in_promos_after_the_date_but_without_s_suffix",
		In: Card{
			Name:      "Astral Drift",
			Variation: "Prerelease",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "2f2a182a-921a-5334-be74-105baca499be",
		Desc: "prerelease_with_duplication",
		In: Card{
			Name:      "Sorcerous Spyglass",
			Variation: "Ixalan Prerelease",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "485c633b-2d07-5a81-8260-dac5641e7a06",
		Desc: "lubu_dedup_july",
		In: Card{
			Name:      "Lu Bu, Master-at-Arms",
			Variation: "July 4 Prerelease",
			Edition:   "ignored",
		},
	},

	// JPN alternate art
	MatchTest{
		Id:   "5e3be15e-e89e-5dc4-8fc8-d2c79f7bdc1d",
		Desc: "normal_non-JPN_version",
		In: Card{
			Name:    "Vraska, Swarm's Eminence",
			Edition: "War of the Spark",
		},
	},
	MatchTest{
		Id:   "6a66d62d-8993-578f-a752-9e56043dd6ef",
		Desc: "JPN_variant",
		In: Card{
			Name:      "Vraska, Swarm's Eminence",
			Variation: "JPN Alternate Art",
			Edition:   "War of the Spark",
		},
	},
	MatchTest{
		Id:   "a9ed85a2-ac31-5059-89d6-ed896d5888b5",
		Desc: "JPN_variant_but_number_could_interfere",
		In: Card{
			Name:      "Teyo, the Shieldmage",
			Variation: "032 - JPN Alternate Art",
			Edition:   "War of the Spark",
		},
	},

	// Borderless cards
	MatchTest{
		Id:   "7998ef11-85d3-5280-b880-bd8b3a896e66",
		Desc: "normal_non-borderless_variant",
		In: Card{
			Name:    "Oko, Thief of Crowns",
			Edition: "Throne of Eldraine",
		},
	},
	MatchTest{
		Id:   "202c6f3d-2094-516d-bd3e-1c3e7d16be3e",
		Desc: "borderless_variant",
		In: Card{
			Name:      "Oko, Thief of Crowns",
			Variation: "Borderless",
			Edition:   "Throne of Eldraine",
		},
	},
	MatchTest{
		Id:   "4030ee0a-aa7c-5e9f-bea7-f1158cfc13ec",
		Desc: "borderless_but_from_a_funny_set",
		In: Card{
			Name:    "Sap Sucker",
			Edition: "Unstable",
		},
	},
	MatchTest{
		Id:   "4fd6ab70-ebfd-52db-a274-ed736a4446c7",
		Desc: "borderless_boxtopper",
		In: Card{
			Name:    "Ancient Tomb",
			Edition: "PUMA",
		},
	},

	// Box topper-style extended art
	MatchTest{
		Id:   "0fa4cc6f-a4f6-5b65-bad8-6b31a7b8ab93",
		Desc: "normal_non-extendedart_variant",
		In: Card{
			Name:    "Heliod's Intervention",
			Edition: "Theros Beyond Death",
		},
	},
	MatchTest{
		Id:   "0546b07f-ea01-524f-8dbb-d661a360528e",
		Desc: "extendedart_variant",
		In: Card{
			Name:      "Heliod's Intervention",
			Variation: "Extended Art",
			Edition:   "Theros Beyond Death",
		},
	},

	// Showcase frame
	MatchTest{
		Id:   "f66a9d9e-9c70-5fe2-86fb-83e7d1ab2fdb",
		Desc: "normal_non-showcase_variant",
		In: Card{
			Name:    "Brazen Borrower",
			Edition: "Throne of Eldraine",
		},
	},
	MatchTest{
		Id:   "973fb2bb-f6ce-5639-b282-25a62dbb6dd6",
		Desc: "showcase_variant",
		In: Card{
			Name:      "Brazen Borrower",
			Variation: "Showcase",
			Edition:   "Throne of Eldraine",
		},
	},
	MatchTest{
		Id:   "043b2460-0892-5cb2-b81c-99a42e2d2fb7",
		Desc: "showcase_borderless",
		In: Card{
			Name:      "Zagoth Triome",
			Variation: "Showcase",
			Edition:   "Ikoria: Lair of Behemoths",
		},
	},

	// Reskinned frame
	MatchTest{
		Id:   "f5737cd9-b418-517b-9a69-705b8c1e402f",
		Desc: "normal_nongodzilla_variant",
		In: Card{
			Name:    "Sprite Dragon",
			Edition: "Ikoria: Lair of Behemoths",
		},
	},
	MatchTest{
		Id:   "d5fda5e1-1d13-5545-be43-04b9ec56d21d",
		Desc: "godzilla_variant",
		In: Card{
			Name:      "Sprite Dragon",
			Variation: "Godzilla",
			Edition:   "Ikoria: Lair of Behemoths",
		},
	},
	MatchTest{
		Id:   "d5fda5e1-1d13-5545-be43-04b9ec56d21d",
		Desc: "godzilla_variant_alt_name",
		In: Card{
			Name:    "Dorat, the Perfect Pet",
			Edition: "Ikoria: Lair of Behemoths",
		},
	},

	// Arabian Nights different mana symbol
	MatchTest{
		Id:   "7cfca2f3-cc9d-5834-8812-31bee017dfbb",
		Desc: "ARN_light_variant",
		In: Card{
			Name:      "Wyluli Wolf",
			Variation: "light circle",
			Edition:   "Arabian Nights",
		},
	},
	MatchTest{
		Id:   "f3a2f42f-02eb-5e1f-bd90-af69997608f5",
		Desc: "ARN_dark_variant",
		In: Card{
			Name:      "Oubliette",
			Variation: "dark circle",
			Edition:   "Arabian Nights",
		},
	},
	MatchTest{
		Id:   "f3a2f42f-02eb-5e1f-bd90-af69997608f5",
		Desc: "ARN_dark_variant_implied",
		In: Card{
			Name:      "Oubliette",
			Variation: "",
			Edition:   "Arabian Nights",
		},
	},
	MatchTest{
		Id:   "8f2426c7-7523-56b8-a5a3-19b2c6b437c7",
		Desc: "ARN_no_variant",
		In: Card{
			Name:    "Abu Ja'far",
			Edition: "Arabian Nights",
		},
	},

	// Variants
	MatchTest{
		Id:   "4f92d091-cc91-5780-92ba-a31041859361_f",
		Desc: "single_variant_with_no_special_tag",
		In: Card{
			Name:    "Will Kenrith",
			Edition: "Battlebond",
			Foil:    true,
		},
	},
	MatchTest{
		Id:   "5e6b6d41-7a08-5eec-8003-8606e85d8b23",
		Desc: "kaya_is_special",
		In: Card{
			Name:    "Kaya, Ghost Assassin",
			Edition: "Conspiracy: Take the Crown",
		},
	},
	MatchTest{
		Id:   "58799c83-290c-58ff-82e3-1f4239f0b1f1_f",
		Desc: "kaya_is_very_special",
		In: Card{
			Name:    "Kaya, Ghost Assassin",
			Edition: "Conspiracy: Take the Crown",
			Foil:    true,
		},
	},
	MatchTest{
		Id:   "4e3106c7-93fa-5f94-8f9e-f5204f1b9d3e",
		Desc: "too_many_variations",
		In: Card{
			Name:    "Tamiyo's Journal",
			Edition: "Shadows over Innistrad",
		},
	},
	MatchTest{
		Id:   "1d640d35-90f4-53f6-8328-b80033f40d8f",
		Desc: "too_many_variations_foil",
		In: Card{
			Name:    "Tamiyo's Journal",
			Edition: "Shadows over Innistrad",
			Foil:    true,
		},
	},
	MatchTest{
		Id:   "69f21ce4-c82d-5e45-adfa-274306fbcdee",
		Desc: "too_many_variations_what_did_i_say",
		In: Card{
			Name:      "Tamiyo's Journal",
			Variation: "Entry 546",
			Edition:   "Shadows over Innistrad",
		},
	},
	MatchTest{
		Id:   "fa763da2-eb6d-5f1e-9311-d76c955d5c57",
		Desc: "custom_variant",
		In: Card{
			Name:      "Urza's Tower",
			Variation: "Mountains",
			Edition:   "Chronicles",
		},
	},
	MatchTest{
		Id:   "7d6ed48a-4634-5f58-a7c0-37bbefcae2f0",
		Desc: "number_with_suffix_in_variant",
		In: Card{
			Name:      "Arcane Denial",
			Variation: "22b",
			Edition:   "Alliances",
		},
	},
	MatchTest{
		Id:   "2e2777f2-833e-5b5b-ae0a-6466e2d25bf3",
		Desc: "one_funny_variation",
		In: Card{
			Name:      "Secret Base",
			Variation: "Version 2",
			Edition:   "Unstable",
		},
	},
	MatchTest{
		Id:   "4dded966-5630-5150-8378-5a6377eb9d6e",
		Desc: "correct_number_but_no_showcase_tag",
		In: Card{
			Name:      "Renata, Called to the Hunt",
			Variation: "267",
			Edition:   "Theros Beyond Death",
		},
	},
	MatchTest{
		Id:   "4d67367e-a5e9-53b6-ba83-9908067a98f9",
		Desc: "mps_lands",
		In: Card{
			Name:      "Island",
			Variation: "Rob Alexander MPS 2009",
			Edition:   "Promos: MPS Lands",
		},
	},
	MatchTest{
		Id:   "3a0cd31b-a6d6-5c9a-a184-91271814bc68",
		Desc: "single_printintg_multiple_variants",
		In: Card{
			Name:      "Taste of Paradise",
			Variation: "TasteOfParadise",
			Edition:   "Alliances",
		},
	},
	MatchTest{
		Id:   "7a0be3c2-accb-511c-bc97-96f9cd6eb1ea",
		Desc: "apac_lands",
		In: Card{
			Name:      "Forest",
			Variation: "Pete Venters",
			Edition:   "Asia Pacific Land Program",
		},
	},
	MatchTest{
		Id:   "0774fb91-081f-577c-b738-dc1ee129dc6c",
		Desc: "euro_lands",
		In: Card{
			Name:      "Plains",
			Variation: "EURO Land Steppe Tundra Ben Thompson art",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "c6fd0db2-2133-51da-962f-6eb5bacf8477",
		Desc: "euro_lands_comma",
		In: Card{
			Name:      "Island",
			Variation: "EURO Land, Venezia",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "ba1b009e-6c3c-54a9-bcdd-d667e69f6425",
		Desc: "artist_last_name_too_many_s",
		In: Card{
			Name:      "Simic Signet",
			Variation: "Mike Sass",
			Edition:   "Commander Anthology Volume II",
		},
	},

	// FNM promos (often confused with set promos)
	MatchTest{
		Id:   "cee1d7d4-3c95-5020-a4f6-20a5efbbb56d",
		Desc: "normal_fnm",
		In: Card{
			Name:      "Aether Hub",
			Variation: "FNM",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "774fb036-b8c4-5af5-86ad-183fe318000a",
		Desc: "fnm_plus_year",
		In: Card{
			Name:      "Goblin Warchief",
			Variation: "FNM 2016",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "913edc80-0d6f-5987-8f58-53271ec5f9e6",
		Desc: "non_fnm_with_inverted_frame",
		In: Card{
			Name:      "Dovin's Veto",
			Variation: "FNM",
			Edition:   "Promo",
		},
	},
	MatchTest{
		Id:   "d779fed9-ae9e-5279-8c1c-d9a5ca60521e",
		Desc: "fnm_with_promo_alias",
		In: Card{
			Name:      "Reliquary Tower",
			Variation: "FNM",
			Edition:   "Promo",
		},
	},
	MatchTest{
		Id:   "7260ac60-e673-5da3-85c0-c6346cec2aac",
		Desc: "non_fnm_with_fnm_alias",
		In: Card{
			Name:      "Reliquary Tower",
			Variation: "Promo",
			Edition:   "Promo",
		},
	},
	MatchTest{
		Id:   "def399de-012a-5608-88f9-81c0bdccea79",
		Desc: "non_fnm_wrong_info",
		In: Card{
			Name:      "Shanna, Sisay's Legacy",
			Variation: "FNM Foil",
			Edition:   "Promos: FNM",
		},
	},

	// Arena
	MatchTest{
		Id:   "9b1cd4a2-2792-5c77-a1a7-0618ae6f8dc5",
		Desc: "normal_arena",
		In: Card{
			Name:      "Enlightened Tutor",
			Variation: "Arena",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "fac9ad88-7d23-5cad-a6e7-6b48d6fbfea0",
		Desc: "arena_with_year",
		In: Card{
			Name:      "Mountain",
			Variation: "Arena 1999",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "b5f48d93-9f68-52a1-a3c3-abf98604e8af",
		Desc: "arena_land_missing_year",
		In: Card{
			Name:      "Forest",
			Variation: "Arena Foil - Mercadian Masques",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "cde0850c-df01-57d6-998b-f92eb21b8eb1",
		Desc: "misprint_within_arena",
		In: Card{
			Name:      "Island",
			Variation: "Arena 1999 misprint",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "8354caf4-0e58-599d-8b31-2d8ef50a9607",
		Desc: "the_only_arena_land_with_number",
		In: Card{
			Name:      "Forest",
			Variation: "Arena 2001 1",
			Edition:   "ignored",
		},
	},

	// Various promos
	MatchTest{
		Id:   "4074ec88-ff9e-54dc-838a-be68edb85a3c",
		Desc: "normal_judge",
		In: Card{
			Name:      "Tradewind Rider",
			Variation: "Judge",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "ac886c46-34b5-54d1-94c3-3b100de28c22",
		Desc: "judge_with_year",
		In: Card{
			Name:      "Vindicate",
			Variation: "Judge 2007",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "3e4f9814-9b8e-56eb-bb9c-96cdb1a0f54c",
		Desc: "normal_sdcc",
		In: Card{
			Name:      "Liliana Vess",
			Variation: "2014 SDCC",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "3e4f9814-9b8e-56eb-bb9c-96cdb1a0f54c",
		Desc: "normal_sdcc_extended_name",
		In: Card{
			Name:      "Liliana Vess",
			Variation: "San Diego Comic Con",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "ce697f23-a87f-5579-9611-069e0d9cdc97",
		Desc: "normal_textless",
		In: Card{
			Name:      "Fireball",
			Variation: "Textless",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "92815b2a-ace8-5200-9f82-97b3edff359b",
		Desc: "normal_idw",
		In: Card{
			Name:      "Wash Out",
			Variation: "IDW",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "ca98ee1e-42fe-5244-8502-e561e341e47e",
		Desc: "normal_gateway",
		In: Card{
			Name:      "Lava Axe",
			Variation: "Gateway",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "efdbe923-5227-5c96-b2bc-371440ed33a3",
		Desc: "normal_wpn",
		In: Card{
			Name:      "Curse of Thirst",
			Variation: "WPN",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "efdbe923-5227-5c96-b2bc-371440ed33a3",
		Desc: "maybe_gateway_or_wpn",
		In: Card{
			Name:      "Curse of Thirst",
			Variation: "Gateway WPN",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "0e69c149-c8a0-507e-968b-c35f9b360e8f_f",
		Desc: "foil_only_booster",
		In: Card{
			Name:      "Skyship Weatherlight",
			Variation: "Alternate Art",
			Edition:   "Planeshift",
			Foil:      true,
		},
	},
	MatchTest{
		Id:   "1ccd68e3-9293-5c59-ac45-28a8b330da16_f",
		Desc: "foil_only_booster_normal_counterpart",
		In: Card{
			Name:    "Skyship Weatherlight",
			Edition: "Planeshift",
			Foil:    true,
		},
	},
	MatchTest{
		Id:   "3b17073e-d0e0-5458-abb3-a102b1faf2f7",
		Desc: "book_promo",
		In: Card{
			Name:      "Jace Beleren",
			Variation: "Book",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "4429b96c-8d56-55d7-a679-9f7afa7d3080",
		Desc: "heros_path_promo",
		In: Card{
			Name:      "The Explorer",
			Variation: "Hero's Path",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "17253c33-5268-5c02-b49e-86e3de212732",
		Desc: "duels_of_the_pw",
		In: Card{
			Name:      "Vigor",
			Variation: "Duels",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "84d1d310-2c91-5d87-a966-0187f53daf31",
		Desc: "duels_with_year",
		In: Card{
			Name:      "Ogre Battledriver",
			Variation: "Duels of the Planeswalkers 2014",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "faf2b777-63a3-5b6e-a9a4-d38d290085bc",
		Desc: "clash_pack",
		In: Card{
			Name:      "Temple of Mystery",
			Variation: "Clash Pack",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "efc5a359-1290-532b-8b20-14af00eaa19b",
		Desc: "japanese_magazine_insert",
		In: Card{
			Name:      "Shivan Dragon",
			Variation: "Japanese Gotta Comic Promo",
			Edition:   "ignored",
		},
	},

	// Release cards
	MatchTest{
		Id:   "68d6d809-9707-5064-acdf-1b0219f620ed",
		Desc: "release_but_it_is_a_promo",
		In: Card{
			Name:      "Valakut, the Molten Pinnacle",
			Variation: "Release Event",
			Edition:   "ignored",
			Foil:      true,
		},
	},
	MatchTest{
		Id:   "a206787f-e354-5141-a9d5-a504269a666a",
		Desc: "release_but_it's_from_launch_parties",
		In: Card{
			Name:      "Vexing Shusher",
			Variation: "Release Event",
			Edition:   "ignored",
			Foil:      true,
		},
	},
	MatchTest{
		Id:   "006358ad-4e4e-5fcd-8ad5-3e79ea72c5cf",
		Desc: "release_events",
		In: Card{
			Name:      "Shriekmaw",
			Variation: "Release Event",
			Edition:   "ignored",
			Foil:      true,
		},
	},
	MatchTest{
		Id:   "69b5853b-9343-588b-b17f-35b0215faaf4",
		Desc: "release_but_there_is_a_prerelease_too",
		In: Card{
			Name:      "Identity Thief",
			Variation: "Release Event",
			Edition:   "ignored",
			Foil:      true,
		},
	},
	MatchTest{
		Id:   "7836a3dd-46a8-5b84-be27-69358ad12639",
		Desc: "prerelease_but_there_is_a_release_too",
		In: Card{
			Name:      "Identity Thief",
			Variation: "Prerelease Event",
			Edition:   "ignored",
			Foil:      true,
		},
	},
	MatchTest{
		Id:   "559059c9-08f2-5b79-9f3d-11ba48fd615d",
		Desc: "non_release_non_prerelease_version",
		In: Card{
			Name:    "Identity Thief",
			Edition: "Eldritch Moon",
		},
	},
	MatchTest{
		Id:   "69b5853b-9343-588b-b17f-35b0215faaf4",
		Desc: "release_too_much_info",
		In: Card{
			Name:      "Identity Thief",
			Variation: "Eldritch Moon Launch Foil 22 July 2016",
			Edition:   "Promos: Miscellaneous",
		},
	},

	// Generic promo
	MatchTest{
		Id:   "822104a3-6ea9-5a82-8b1a-c6ae6908a245",
		Desc: "variation_has_no_useful_info",
		In: Card{
			Name:      "Zombie Apocalypse",
			Variation: "Some Kind of Promo",
			Edition:   "ignored",
			Foil:      true,
		},
	},
	MatchTest{
		Id:   "7454b0ab-b3ae-5789-ba15-c1e057e83ea0",
		Desc: "variation_has_no_useful_info_may_trigger_dupes_if_incorrectly_handled",
		In: Card{
			Name:      "Unclaimed Territory",
			Variation: "League Promo",
			Edition:   "ignored",
			Foil:      true,
		},
	},
	MatchTest{
		Id:   "b969d9fc-ef87-52a5-8e1c-7929f943a451",
		Desc: "unknown_promo",
		In: Card{
			Name:      "Trueheart Duelist",
			Variation: "Game Day Extended",
			Edition:   "ignored",
		},
	},

	// Buy-a-Box promo
	MatchTest{
		Id:   "2dd0da3d-0df8-5ba1-9fb3-378525d5804d",
		Desc: "bab_marked_as_promo_but_it's_really_in_the_set",
		In: Card{
			Name:      "Impervious Greatwurm",
			Variation: "BIBB",
			Edition:   "ignored",
			Foil:      true,
		},
	},
	MatchTest{
		Id:   "efa5a09d-c829-5b77-84b1-4c0137524b45",
		Desc: "bab_marked_as_promo_but_it's_really_in_the_set_set_is_not_an_expansion",
		In: Card{
			Name:      "Flusterstorm",
			Variation: "buy a box",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "f110dc89-178b-5eb6-9f96-542a5ad15a67",
		Desc: "bab_marked_as_promo_but_it's_really_in_the_set_set_is_core",
		In: Card{
			Name:      "Rienne Angel of Rebirth",
			Variation: "M20 BIBB",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "e1a8180f-c67a-5727-97dc-5cba992e6eec",
		Desc: "bab_old_style_it_is_in_Promos",
		In: Card{
			Name:      "Sylvan Caryatid",
			Variation: "buy-a-box",
			Edition:   "ignored",
			Foil:      true,
		},
	},
	MatchTest{
		Id:   "bc9aeef4-0901-5030-a7ed-cc1763de436f",
		Desc: "bab_but_also_pro_tour",
		In: Card{
			Name:      "Surgical Extraction",
			Variation: "BIBB Promo",
			Edition:   "ignored",
			Foil:      true,
		},
	},
	MatchTest{
		Id:   "6f06d428-d70a-55fd-9e74-f3cfec362194",
		Desc: "bab_but_also_in_normal_set",
		In: Card{
			Name:      "Mirran Crusader",
			Variation: "Buy-a-Box",
			Edition:   "ignored",
			Foil:      true,
		},
	},
	MatchTest{
		Id:   "bc6e0fcd-e517-5eb1-8187-928db72944e7",
		Desc: "bab_in_separate_set_with_wrong_info",
		In: Card{
			Name:      "Growing Rites of Itlimoc",
			Variation: "buy-a-box",
			Edition:   "Ixalan Promos",
		},
	},

	// Bundle promo
	MatchTest{
		Id:   "64c13dd8-a5dc-51a1-b69e-94cbffa7da33",
		Desc: "non_bundle_in_the_same_set",
		In: Card{
			Name:    "Piper of the Swarm",
			Edition: "Throne of Eldraine",
		},
	},
	MatchTest{
		Id:   "e44ecf94-bcbb-526c-9e28-7ae245b99fd9",
		Desc: "bundle_in_the_same_set",
		In: Card{
			Name:      "Piper of the Swarm",
			Variation: "Bundle",
			Edition:   "Throne of Eldraine",
		},
	},
	MatchTest{
		Id:   "e44ecf94-bcbb-526c-9e28-7ae245b99fd9",
		Desc: "bundle_in_the_same_set_but_unknown_set",
		In: Card{
			Name:      "Piper of the Swarm",
			Variation: "Bundle Promo",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "54b7b9f8-d821-5e14-bbfb-a16100b75967",
		Desc: "non_bundle_in_the_same_set_but_special_version",
		In: Card{
			Name:      "Piper of the Swarm",
			Variation: "Extended Art",
			Edition:   "Throne of Eldraine",
		},
	},

	// MagicFest
	MatchTest{
		Id:   "f78bf80a-a9c0-5b18-89b9-eb99fcf4c8d7_f",
		Desc: "mf_pte",
		In: Card{
			Name:      "Path to Exile",
			Variation: "Magic Fest",
			Edition:   "ignored",
			Foil:      true,
		},
	},
	MatchTest{
		Id:   "9f799aed-dc03-5e83-b322-092e27a42f03",
		Desc: "bfz_std_with_wrong_info",
		In: Card{
			Name:      "Sunken Hollow",
			Variation: "alt art",
			Edition:   "Dominaria",
		},
	},
	MatchTest{
		Id:   "8be4ff47-6e0b-5c3b-8bf0-f6a7d6b2f318",
		Desc: "unstable_letter_variant",
		In: Card{
			Name:      "Very Cryptic Command",
			Variation: "E Counter/Return/Untap/Roll",
			Edition:   "Unstable",
		},
	},
	MatchTest{
		Id:   "f78bf80a-a9c0-5b18-89b9-eb99fcf4c8d7",
		Desc: "magicfest_textless",
		In: Card{
			Name:      "Path to Exile",
			Variation: "MagicFest Textless",
			Edition:   "ignored",
		},
	},

	// Homelands and Fallen Empires
	MatchTest{
		Id:   "74746784-4158-5498-aef2-39ac4e29e965",
		Desc: "homelands_flavor",
		In: Card{
			Name:      "Abbey Matron",
			Variation: "Quote Halina, Dwarven Trader",
			Edition:   "Homelands",
		},
	},
	MatchTest{
		Id:   "1a6801e3-e4a9-5dce-8925-59acb624cffd",
		Desc: "homelands_flavor_alt",
		In: Card{
			Name:      "Folk of An-Havva",
			Variation: "Quote Joskun, An-Havva Constable",
			Edition:   "Homelands",
		},
	},
	MatchTest{
		Id:   "64d73c3e-35f3-5378-b8f7-f61c7ae9f3ac",
		Desc: "homelands_flavor_with_extra",
		In: Card{
			Name:      "Memory Lapse",
			Variation: "Quote Chandler, Female Art",
			Edition:   "Homelands",
		},
	},
	MatchTest{
		Id:   "1271a401-45ea-5309-b0e6-de4b3bc08246",
		Desc: "fem_artist",
		In: Card{
			Name:      "Armor Thrull",
			Variation: "Jeff A. Menges",
			Edition:   "Fallen Empires",
		},
	},
	MatchTest{
		Id:   "75691ff3-7b01-546c-9285-a24a54eb035c",
		Desc: "fem_artist_incomplete",
		In: Card{
			Name:      "Icatian Javelineers",
			Variation: "Melissa Benson",
			Edition:   "Fallen Empires",
		},
	},
	MatchTest{
		Id:   "a8d08f7e-1da6-5270-a8c5-93a83a37a739",
		Desc: "variant_is_number_suffix",
		In: Card{
			Name:      "Homarid Warrior",
			Variation: "B",
			Edition:   "Fallen Empires",
		},
	},
	MatchTest{
		Id:   "56f84558-0802-5d35-aa5b-8e3b55cced70",
		Desc: "variant_is_polluted",
		In: Card{
			Name:      "Basal Thrull",
			Variation: "Artist Phil Foglio",
			Edition:   "Fallen Empires",
		},
	},

	// Duel Decks
	MatchTest{
		Id:   "88b2a7d9-0b56-5db7-9eb3-de96a5c22033",
		Desc: "duel_decks_variant",
		In: Card{
			Name:    "Goblin Rabblemaster",
			Edition: "DD: Merfolk vs Goblins",
		},
	},
	MatchTest{
		Id:   "4bd321f5-d0c9-570e-9227-969ee141f897",
		Desc: "dda_deck_in_variation",
		In: Card{
			Name:      "Flamewave Invoker",
			Variation: "Jace vs Chandra",
			Edition:   "Duel Decks Anthology",
		},
	},
	MatchTest{
		Id:   "6354ec62-b96d-5408-85d2-af3423b45b2e",
		Desc: "dda_deck_in_variation_inverted",
		In: Card{
			Name:      "Flamewave Invoker",
			Variation: "Goblins vs Elves",
			Edition:   "Duel Decks Anthology",
		},
	},
	MatchTest{
		Id:   "25fd12bc-c4ae-5f0a-bd6c-5c676238d542",
		Desc: "duel_decks_variant_with_number",
		In: Card{
			Name:      "Forest",
			Variation: "#38",
			Edition:   "DD: Zendikar vs. Eldrazi",
		},
	},
	MatchTest{
		Id:   "e63ea080-3ea6-5a80-bfce-db6dffa4025c",
		Desc: "confusing_dd_with_mb1_tag",
		In: Card{
			Name:      "Elvish Warrior",
			Variation: "Mystery Booster",
			Edition:   "Elves vs. Goblins",
		},
	},

	// Deckmasters variants
	MatchTest{
		Id:   "c4e91c2a-3747-5b5a-8b1d-f4083dbcb717",
		Desc: "number_in_variation_with_other_text",
		In: Card{
			Name:      "Phyrexian War Beast",
			Variation: "37A Propeller Right",
			Edition:   "DKM",
		},
	},
	MatchTest{
		Id:   "c4e91c2a-3747-5b5a-8b1d-f4083dbcb717",
		Desc: "variation_use_first_if_empty",
		In: Card{
			Name:      "Phyrexian War Beast",
			Variation: "",
			Edition:   "DKM",
		},
	},
	MatchTest{
		Id:   "b9b3308d-f0ef-57dd-91ed-ef0597011f44_f",
		Desc: "variation_deckmasters_foil",
		In: Card{
			Name:      "Incinerate",
			Variation: "Foil",
			Edition:   "Deckmasters",
			Foil:      true,
		},
	},
	MatchTest{
		Id:   "4846fe2e-90a5-564c-a2ce-b1636fd97b0a",
		Desc: "variation_deckmasters_non_foil",
		In: Card{
			Name:    "Incinerate",
			Edition: "Deckmasters",
		},
	},
	MatchTest{
		Id:   "7b2244af-2539-5b30-8149-7ee9dfd2b956_f",
		Desc: "variation_deckmasters_foil_but_untagged",
		In: Card{
			Name:      "Icy Manipulator",
			Variation: "Promo",
			Edition:   "Deckmasters",
		},
	},

	// Champs
	MatchTest{
		Id:   "0d10eae1-3203-5e77-acf7-04662e819a9c",
		Desc: "states_but_is_gateway",
		In: Card{
			Name:      "Dauntless Dourbark",
			Variation: "2008 States",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "71a80ef0-6cd3-5001-9d2c-7487eb9a1d8a",
		Desc: "champs_and_states",
		In: Card{
			Name:      "Voidslime",
			Variation: "Champs",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "d798e324-c143-5fcb-b8b4-ab03e767a12f",
		Desc: "not_champs",
		In: Card{
			Name:      "Ghalta, Primal Hunger",
			Variation: "Champs / States",
			Edition:   "ignored",
		},
	},

	// IDW and Comic promos
	MatchTest{
		Id:   "a129c134-546f-5038-9498-9a547fa4bf37",
		Desc: "idw_also_magazine",
		In: Card{
			Name:      "Duress",
			Variation: "IDW Promo",
			Edition:   "ignored",
		},
	},
	MatchTest{
		Id:   "010057a9-b30e-590a-908e-2c41983e7ef1",
		Desc: "magazine_also_idw",
		In: Card{
			Name:      "Duress",
			Variation: "Japanese Magazine Promo",
			Edition:   "ignored",
		},
	},

	// Core sets
	MatchTest{
		Id:   "da99586d-41ca-5de6-83bb-18373c28ec69",
		Desc: "coreset",
		In: Card{
			Name:    "Guttersnipe",
			Edition: "Core Set 2019 / M19",
		},
	},
	MatchTest{
		Id:   "2fa8e0c4-e41a-5c6c-a8bf-8c83e3b929b8",
		Desc: "confusing_promo_and_coreset",
		In: Card{
			Name:      "Naya Sojourners",
			Variation: "Magic 2010 Game Day",
			Edition:   "Promo Magic 2010 Game Day",
		},
	},

	// WCD
	MatchTest{
		Id:   "fe2b7fed-4cd7-5072-ae7f-2016a5714cd9",
		Desc: "wcd_pick_the_first_one_if_not_enough_info",
		In: Card{
			Name:      "Ancient Tomb",
			Variation: "Tokyo 1999 - Not Tournament Legal",
			Edition:   "World Championships",
		},
	},
	MatchTest{
		Id:   "d7b0b59a-5bb8-5770-b263-1fe6da80f2db",
		Desc: "wcd_with_number",
		In: Card{
			Name:      "Plains",
			Variation: "8th Edition 332 Julien Nuijten 2004",
			Edition:   "World Championship",
		},
	},
	MatchTest{
		Id:   "2007d75a-b8e0-5a78-943f-9b4332cdc2d4",
		Desc: "wcd_with_variant",
		In: Card{
			Name:      "Memory Lapse",
			Variation: "Statue A Sideboard Shawn Hammer Regnier",
			Edition:   "World Championship",
		},
	},
	MatchTest{
		Id:   "d89a3dc2-4b46-5940-b766-47a52d098c2c",
		Desc: "wcd_with_variant_embedded_in_number",
		In: Card{
			Name:      "Plains",
			Variation: "Odyssey 331 Brian Kibler 2002",
			Edition:   "World Championship",
		},
	},
	MatchTest{
		Id:   "54435363-233c-50b8-a13e-50155e6071f5",
		Desc: "wcd_with_player_name_aliasing",
		In: Card{
			Name:      "Cursed Scroll",
			Variation: "Matt Linde 1999",
			Edition:   "World Championship",
		},
	},
	MatchTest{
		Id:   "f3659f55-3e3e-5ade-8db0-5099b4dac1ac",
		Desc: "wcd_with_correct_number",
		In: Card{
			Name:      "Strip Mine",
			Variation: "ll363",
			Edition:   "Pro Tour Collector Set",
		},
	},
	MatchTest{
		Id:   "ebd2ea21-16d4-5e84-b54d-a74d81560ce2",
		Desc: "wcd_only_the_year",
		In: Card{
			Name:      "Karplusan Forest",
			Variation: "Brussels, August 2000",
			Edition:   "World Championships 2000",
		},
	},

	// Foil-only special category
	MatchTest{
		Id:   "467d0a0a-ba76-5b99-853e-91fd8deb8989_f",
		Desc: "foil_only_booster",
		In: Card{
			Name:    "Time Stop",
			Edition: "Tenth Edition",
			Foil:    true,
		},
	},
	MatchTest{
		Id:   "bef2dc94-74fe-5ea1-936e-acf3c8e5e169_f",
		Desc: "foil_only_booster_normal",
		In: Card{
			Name:    "Angelic Chorus",
			Edition: "Tenth Edition",
			Foil:    true,
		},
	},

	// Portal variants
	MatchTest{
		Id:   "0ac5854b-4952-56ea-ab6f-051e8c2f1d98",
		Desc: "portal_starter_deck",
		In: Card{
			Name:      "Blaze",
			Variation: "reminder text",
			Edition:   "Portal",
		},
	},
	MatchTest{
		Id:   "a85c8128-ec69-54f0-8aa6-32067587a139",
		Desc: "portal_demo_game",
		In: Card{
			Name:      "Cloud Pirates",
			Variation: "reminder text",
			Edition:   "Portal",
		},
	},
	MatchTest{
		Id:   "937b8e57-d3c8-52ba-aecd-fb3897ab2416",
		Desc: "portal_starter_deck",
		In: Card{
			Name:      "Raging Goblin",
			Variation: "No flavor text",
			Edition:   "Portal",
		},
	},
}

func TestMain(m *testing.M) {
	allprintingsPath := os.Getenv("ALLPRINTINGS_PATH")
	allcardsPath := os.Getenv("ALLCARDS_PATH")
	if allprintingsPath == "" || allcardsPath == "" {
		fmt.Println("Need both ALLPRINTINGS_PATH and ALLCARDS_PATH set to run tests")
		os.Exit(1)
	}

	var err error
	internal, err = NewDatabaseFromPaths(allprintingsPath, allcardsPath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestMatch(t *testing.T) {
	logger := log.New(os.Stderr, "", 0)

	for _, probe := range MatchTests {
		test := probe
		t.Run(test.Desc, func(t *testing.T) {
			//t.Parallel()

			card, err := internal.Match(&test.In, logger)
			if err == nil && test.Err != nil {
				t.Errorf("FAIL: Expected error: %s", test.Err.Error())
				return
			}
			if err != nil {
				if test.Err == nil {
					t.Errorf("FAIL: Unexpected error: %s", err.Error())
					return
				}
				if test.Err.Error() != err.Error() {
					t.Errorf("FAIL: Mismatched error: expected '%s', got '%s'", test.Err.Error(), err.Error())
					return
				}
			} else if card.Id != test.Id {
				t.Errorf("FAIL: Id mismatch: expected '%s', got '%s'", test.Id, card.Id)
				return
			}

			t.Log("PASS:", test.Desc)
		})
	}
}
