package magic

import (
	"slices"
	"strconv"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Rules implements mtgmatcher.GameRules for Magic: the Gathering. The bodies of
// the hooks are being relocated here from core mtgmatcher one subsystem at a
// time; hooks whose body has not moved yet delegate to the core method.
type Rules struct{}

func (Rules) AdjustEdition(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	edition := inCard.Edition
	variation := inCard.Variation

	set, found := b.Sets[strings.ToUpper(edition)]
	if found {
		edition = set.Name
	}
	ed, found := mtgmatcher.EditionTable[edition]
	if found {
		edition = ed
	}
	set, found = b.Sets[strings.ToUpper(variation)]
	if found && (inCard.IsJudge() || inCard.IsDuelDecks() || inCard.IsDuelDecksAnthology()) {
		edition = set.Name
	}
	ed, found = mtgmatcher.EditionTable[variation]
	// The Anthologies set has one land with a variant named as an expansion,
	// so what is found should not overwrite the edition in this case
	// As for The List, ignore any further variation
	if found && edition != "Anthologies" && !inCard.IsMysteryList() {
		edition = ed

		// If edition was found through the variation tag, drop it
		variation = ""
		// Only keep this information if needed
		if inCard.IsEtched() {
			variation = "Etched"
		}
	}
	inCard.Edition = edition
	inCard.Variation = variation

	// Adjust box set
	switch {
	case inCard.Contains("Mythic Edition"):
		edition = "Mythic Edition"
	case strings.Contains(edition, "Invocation") ||
		((edition == "Hour of Devastation" || edition == "Amonkhet") &&
			strings.Contains(inCard.Variation, "Invocation")):
		edition = "Amonkhet Invocations"
	case strings.Contains(edition, "Inventions"):
		edition = "Kaladesh Inventions"
	case strings.Contains(edition, "Expeditions") && !strings.Contains(edition, "Rising"):
		edition = "Zendikar Expeditions"
	case strings.Contains(edition, "Expeditions") && strings.Contains(edition, "Rising"):
		edition = "Zendikar Rising Expeditions"
	case inCard.Contains("Timeshift") && inCard.Contains("Spiral") && !inCard.IsMysteryList():
		if len(b.MatchInSet(inCard.Name, "TSB")) != 0 {
			edition = b.Sets["TSB"].Name
		} else if len(b.MatchInSet(inCard.Name, "TSR")) != 0 {
			edition = b.Sets["TSR"].Name
		}
	default:
		edition = strings.TrimPrefix(edition, "Magic: The Gathering - ")
		edition = strings.TrimPrefix(edition, "Magic: the Gathering - ")
		edition = strings.TrimPrefix(edition, "Magic: The Gathering | ")
		edition = strings.TrimPrefix(edition, "Magic: the Gathering | ")

		// Cut the edition at the first dash, but avoid Prerelease and PromoPack and MB1/List cards
		// since they are often separated with a dash, but are processed elsewhere
		// Test for "- " and " -" to avoid catching dashes in the name of the edition
		if !inCard.IsPrerelease() && !inCard.IsPromoPack() && !inCard.IsMysteryList() &&
			(strings.Contains(edition, "- ") || strings.Contains(edition, " -")) {
			edition = strings.Split(edition, "-")[0]
			edition = strings.TrimSpace(edition)

			// Check if the edition name needs further processing
			ed, found = mtgmatcher.EditionTable[edition]
			if found {
				edition = ed
			}

			if variation == "" {
				inCard.BeyondBaseSet = true
			}
		}
		// Loop through known editions tags
		for _, tag := range []string{
			"Box Toppers",
			"(Collector Edition)",
			"Collectors",
			"Collector Booster",
			"Extras",
			"Variants",
			"Etched",
			"Serialized",
			"Surge Foil",
			"Holiday Release",
			"Alternate Foil",
			"Retro Frame",
		} {
			// Strip away any extra tags
			if strings.HasSuffix(edition, tag) {
				edition = strings.TrimSuffix(edition, tag)
				edition = strings.TrimSpace(edition)
				edition = strings.TrimSuffix(edition, ":")
				edition = strings.TrimSuffix(edition, "-")
				edition = strings.TrimSpace(edition)

				// Check if the edition name needs further processing
				ed, found = mtgmatcher.EditionTable[edition]
				if found {
					edition = ed
				}

				// If no other variation, set this flag to do a best effort search
				if variation == "" {
					inCard.BeyondBaseSet = true
				}
			}
		}
	}

	switch {
	case strings.HasPrefix(edition, "Universes Beyond"),
		strings.HasPrefix(edition, "UB:"):
		edition = strings.TrimPrefix(edition, "Universes Beyond")
		edition = strings.TrimPrefix(edition, "UB")
		edition = strings.TrimLeft(edition, ":- ")

		ed, found = mtgmatcher.EditionTable[edition]
		if found {
			edition = ed
		}
	case strings.Contains(edition, "Commander") &&
		(!inCard.Contains("Oversize") || inCard.Contains("Plane") || inCard.Contains("Phenomenon")) &&
		!inCard.Contains("Party"):
		ed := b.ParseCommanderEdition(edition, variation)
		if ed != "" {
			edition = ed
		}
	case inCard.Contains("Ravnica Weekend"):
		edition, variation = inCard.RavnicaWeekend()
	case inCard.Contains("Guild Kit"):
		edition = inCard.RavnicaGuidKit()
	case strings.Contains(variation, "APAC Set") || strings.Contains(variation, "Euro Set"):
		num := mtgmatcher.ExtractNumber(variation)
		if num != "" {
			variation = strings.Replace(variation, num+" ", "", 1)
		}
	case strings.HasPrefix(variation, "Junior") && strings.Contains(variation, "APAC"),
		strings.HasPrefix(variation, "Junior APAC Series") && strings.Contains(variation, "U"):
		edition = "Junior APAC Series"
	case strings.HasPrefix(variation, "Junior Super Series"),
		strings.HasPrefix(variation, "MSS Foil"),
		strings.HasPrefix(variation, "MSS #J"),
		strings.HasPrefix(variation, "MSS Promo J"),
		strings.HasPrefix(variation, "JSS #J"),
		strings.Contains(variation, "JSS Foil") && !mtgmatcher.Contains(variation, "euro"):
		edition = "Junior Super Series"
	case strings.HasPrefix(variation, "Junior Series Europe"),
		strings.HasPrefix(variation, "Junior Series E"),
		strings.HasPrefix(variation, "Junior Series #E"),
		strings.HasPrefix(variation, "Junior Series Promo E"),
		strings.HasPrefix(variation, "Junior Series Promo Foil E"),
		strings.HasPrefix(variation, "ESS Foil E"),
		strings.HasPrefix(variation, "European JrS E"),
		strings.HasPrefix(variation, "European JSS Foil E"):
		edition = "Junior Series Europe"
	case mtgmatcher.Contains(variation, "Boosterfun"):
		inCard.BeyondBaseSet = true
	}
	inCard.Edition = edition
	inCard.Variation = variation

	// Special handling since so many providers get this wrong
	switch {
	// Prevent tags from being mixed up, only take care of edition changes
	case inCard.IsMysteryList():
		switch inCard.Name {
		case "Rafiq of the Many":
			edition = "Shards of Alara"
			variation = "250"
		default:
			// Decouple wrong SLX cards bundled in PLST, as long as they are not reprinted in PLST
			// In that case we trust the source has been properly tagged and will be decoupled later
			if !inCard.IsReskin() && len(b.MatchInSet(inCard.Name, "SLX")) != 0 && len(b.MatchInSet(inCard.Name, "PLST")) == 0 {
				edition = b.Sets["SLX"].Name
			}
		}

	// XLN Treasure Chest
	case inCard.IsBaB() && len(b.MatchInSet(inCard.Name, "PXTC")) != 0:
		edition = b.Sets["PXTC"].Name

	// BFZ Standard Series
	case inCard.IsGenericAltArt() && len(b.MatchInSet(inCard.Name, "PSS1")) != 0:
		edition = b.Sets["PSS1"].Name

	// Champs and States
	case inCard.IsGenericExtendedArt() && len(b.MatchInSet(inCard.Name, "PCMP")) != 0:
		edition = b.Sets["PCMP"].Name

	// Secret Lair {Ultimate,Drop}
	case inCard.IsSecretLair():
		// Check if there are also FlavorNames associated to this card
		// It might happen that a non-FlavorName is requested, so check number too
		altProps, found := b.AlternateProps[inCard.Name]
		if found && len(b.MatchInSet(altProps.OriginalName, "SLD")) != 0 {
			var shouldRename bool
			cards := b.MatchInSet(altProps.OriginalName, "SLD")
			num := mtgmatcher.ExtractNumber(inCard.Variation)
			for _, card := range cards {
				if card.Number == num || (card.FaceFlavorName != "" && mtgmatcher.Contains(inCard.Variation, card.FaceFlavorName)) {
					shouldRename = true
					break
				}
			}

			if shouldRename {
				inCard.Name = altProps.OriginalName
			}
		}

		// This needs to be repeated because this could be skipped if the
		// actual name is used instead
		switch {
		case inCard.Contains("Plains") || inCard.Contains("Battlefield Forge"):
			if inCard.Contains("Unpeeled") || inCard.Contains("669") {
				inCard.Name = "Battlefield Forge"
				inCard.Variation = "669"
			} else if inCard.Contains("Peeled") || inCard.Contains("670") {
				inCard.Name = "Plains"
				inCard.Variation = "670"
			}
		case mtgmatcher.Contains(inCard.Name, "Blightsteel Colossus"), mtgmatcher.Contains(inCard.Name, "Megatron"), mtgmatcher.Contains(inCard.Name, "FAS-BOR7 Horus"),
			inCard.Contains("Blightsteel Colossus"), inCard.Contains("Megatron"), inCard.Contains("FAS-BOR7 Horus"):
			if mtgmatcher.Contains(inCard.Name, "Megatron") || inCard.Contains("Megatron") {
				variation = "1079"
			} else if mtgmatcher.Contains(inCard.Name, "FAS-BOR7 Horus") || inCard.Contains("FAS-BOR7 Horus") {
				variation = "2223"

			}
		}

	// Untagged Planeshift Alternate Art - these could be solved with the
	// Promo handling, but they are not set as such in scryfall
	case (inCard.IsGenericPromo() || inCard.IsGenericAltArt()) && len(b.MatchInSet(inCard.Name, "PLS")) == 2:
		edition = "PLS"
		variation = "Alternate Art"

	// Rename the official name to the the more commonly used name
	case inCard.Edition == "Commander Legends" && inCard.IsShowcase():
		variation = "Etched"

	// Planechase deduplication
	case inCard.Contains("Planechase") && len(b.MatchInSet(inCard.Name, "DCI")) != 0 && (inCard.IsRelease() || inCard.IsDCIPromo() || inCard.IsWPNGateway()):
		edition = b.Sets["DCI"].Name
	case inCard.Equals("Planechase") && len(b.MatchInSet(inCard.Name, "OHOP")) != 0:
		edition = b.Sets["OHOP"].Name
	case inCard.Equals("Planechase 2012") && len(b.MatchInSet(inCard.Name, "OPC2")) != 0:
		edition = b.Sets["OPC2"].Name
	case inCard.Equals("Planechase Anthology") && len(b.MatchInSet(inCard.Name, "OPCA")) != 0:
		edition = b.Sets["OPCA"].Name

	// The first Gift Pack often get folded in the main Core Set 2019 or in the
	// related Promos set, so use a lax way to dected the original expansion
	case ((mtgmatcher.Contains(inCard.Edition, "Core") && mtgmatcher.Contains(inCard.Edition, "2019")) || inCard.IsGenericPromo()) && len(b.MatchInSet(inCard.Name, "G18")) == 1:
		edition = b.Sets["G18"].Name

	// Adjust edition for non-English sets
	case (inCard.Edition == "Legends" || inCard.Edition == "The Dark") && mtgmatcher.Contains(inCard.Variation, "Italian"):
		edition += " Italian"
	case inCard.Edition == "Renaissance" && mtgmatcher.Contains(inCard.Variation, "Italian"):
		edition = "Rinascimento"
		// This set has lots of variants, strip away any excess data
		variation = strings.ToLower(inCard.Variation)
		variation = strings.Replace(variation, "italian", "", 1)
		variation = strings.TrimSpace(variation)
	case strings.Contains(inCard.Edition, "Chronicles") && (inCard.Contains("Japanese") || inCard.Contains("FBB")):
		edition = "Chronicles Foreign Black Border"
		// This set has lots of variants, strip away any excess data
		variation = strings.ToLower(inCard.Variation)
		variation = strings.Replace(variation, "japanese", "", 1)
		variation = strings.TrimSpace(variation)
	case inCard.Edition == "Fourth Edition" && mtgmatcher.Contains(inCard.Variation, "Japanese"):
		edition = "Fourth Edition Foreign Black Border"
		// Helps with land variants
		variation = strings.ToLower(inCard.Variation)
		variation = strings.Replace(variation, "japanese", "", 1)
		variation = strings.Replace(variation, "bb", "", 1)
		variation = strings.TrimSpace(variation)

	// Separate timeshifted cards
	case inCard.Contains("Modern Horizons") &&
		(inCard.Contains("Retro Frame") || inCard.Contains("Timeshift")) &&
		(len(b.MatchInSet(inCard.Name, "H1R")) != 0 || len(b.MatchInSet(inCard.Name, "H2R")) != 0):
		if len(b.MatchInSet(inCard.Name, "H1R")) != 0 {
			edition = b.Sets["H1R"].Name
		} else if len(b.MatchInSet(inCard.Name, "H2R")) != 0 {
			edition = b.Sets["H2R"].Name
		}

	// Clash pack promos
	case (inCard.Contains("Clash") || inCard.IsGenericPromo()) && len(b.MatchInSet(inCard.Name, "CP1")) == 1:
		edition = b.Sets["CP1"].Name
	case (inCard.Contains("Clash") || inCard.IsGenericPromo()) && len(b.MatchInSet(inCard.Name, "CP2")) == 1:
		edition = b.Sets["CP2"].Name
	case (inCard.Contains("Clash") || inCard.IsGenericPromo()) && len(b.MatchInSet(inCard.Name, "CP3")) == 1:
		edition = b.Sets["CP3"].Name

	// Challenger decks promos
	case (inCard.Contains("Challenger Decks") || inCard.IsGenericPromo()) && len(b.MatchInSet(inCard.Name, "Q06")) != 0:
		edition = b.Sets["Q06"].Name

	// Open the Helvault oversized cards
	case (inCard.Contains("Oversize") || inCard.Contains("Helvault Promo") || inCard.IsPrerelease()) && len(b.MatchInSet(inCard.Name, "PHEL")) == 1:
		edition = b.Sets["PHEL"].Name
		variation = ""

	// All the oversized commander cards
	case inCard.Contains("Oversize") && !inCard.Contains("Plane") && !inCard.Contains("Phenomenon"):
		for _, tag := range []string{
			"OCM1", "PCMD", "OCMD", "OC13", "OC14", "OC15", "OC16", "OC17", "OC18", "OC19", "OC20",
		} {
			if inCard.Name == "Mayael the Anima" && !inCard.Contains("Arsenal") {
				edition = b.Sets["OC13"].Name
				break
			} else if len(b.MatchInSet(inCard.Name, tag)) == 1 {
				edition = b.Sets[tag].Name
				break
			}
		}

	// Lunar Year Promos
	case (inCard.IsGenericPromo() || inCard.Contains("Lunar")) && len(b.MatchInSet(inCard.Name, "PL21")) == 1:
		edition = b.Sets["PL21"].Name

	// Love Your LGS 2021, often confused with WPN
	case (inCard.IsWPNGateway() || inCard.IsGenericPromo()) && inCard.Contains("Retro Frame") && len(b.MatchInSet(inCard.Name, "PLG21")) == 1:
		edition = b.Sets["PLG21"].Name

	// WPN 2021
	case inCard.Name != "Mind Stone" && inCard.IsGenericPromo() && len(b.MatchInSet(inCard.Name, "PW21")) == 1:
		edition = b.Sets["PW21"].Name

	// Unfinity Sticker Sheets
	case inCard.Edition == "Unfinity" && len(b.MatchInSet(inCard.Name, "SUNF")) == 1:
		edition = b.Sets["SUNF"].Name

	// Move Release to Prerelease for Battlebond
	case inCard.IsRelease() && strings.Contains(edition, "Battlebond") && len(b.MatchInSet(inCard.Name, "PBBD")) == 1:
		edition = b.Sets["PBBD"].Name

	// Remove edition since the cards are either in ONE or in another set, but single printed
	case inCard.Contains("Phyrexia: All") && inCard.Contains("Concept"):
		switch inCard.Name {
		default:
			edition = "ignored"
		}

	// Decouple P30A from P30H and P30T
	case inCard.Contains("30th Anniversary") && !inCard.Contains("Edition") && !inCard.Contains("Tokyo") && !inCard.Contains("Misc") && len(b.MatchInSet(inCard.Name, "P30H")) > 0:
		maybeEdition := b.Sets["P30H"].Name
		if inCard.Name == "Serra Angel" && (!inCard.Contains("History") || mtgmatcher.ExtractYear(inCard.Variation) != "") {
			maybeEdition = b.Sets["P30A"].Name
		}
		edition = maybeEdition

	// Oilslick lands may not have the bundle tag attached to them
	case isBasicLand(inCard) && inCard.IsOilSlick() && !inCard.IsBundle():
		variation += " Bundle"

	// Many providers don't tag these promos correctly
	case inCard.IsRelease() && len(b.MatchInSet(inCard.Name, "PBBD")) == 1:
		edition = b.Sets["PBBD"].Name
		variation = "Prerelease"

	// Single card mismatches
	default:
		switch inCard.Name {
		case "Rhox":
			if inCard.IsGenericAltArt() || inCard.IsGenericPromo() {
				edition = "Starter 2000"
			}
		case "Balduvian Horde":
			if inCard.IsJudge() || inCard.IsGenericPromo() || inCard.IsDCIPromo() {
				edition = "World Championship Promos"
			}
		case "Disenchant":
			if inCard.IsArena() && inCard.Foil {
				edition = "Friday Night Magic 2003"
			}
		case "Nalathni Dragon":
			edition = "Dragon Con"
			variation = ""
		case "Ass Whuppin'",
			"Rukh Egg",
			"Scholar of the Lost Trove":
			if inCard.IsPrerelease() {
				variation = "Release"
				edition = "Release Events"
			}
		case "Reya Dawnbringer":
			if inCard.IsRelease() {
				edition = "Tenth Edition Promos"
			}
		case "Ajani Vengeant":
			if inCard.IsRelease() {
				variation = "Prerelease"
			}
		case "Tamiyo's Journal":
			if (inCard.Variation == "" || mtgmatcher.ExtractNumber(inCard.Variation) == "265") && inCard.Foil {
				variation = "Foil"
			}
		case "Underworld Dreams":
			if inCard.IsDCIPromo() || inCard.IsArena() || inCard.Contains("2HG") || inCard.Contains("Two-Headed Giant") {
				edition = "Two-Headed Giant Tournament"
			}
		case "Jace Beleren":
			if inCard.IsDCIPromo() {
				edition = "Miscellaneous Book Promos"
			}
		case "Serra Angel":
			if inCard.IsDCIPromo() || inCard.IsBaB() {
				edition = "Wizards of the Coast Online Store"
			}
		case "Incinerate", "Counterspell":
			if inCard.IsDCIPromo() || (inCard.Contains("Legend") && (inCard.Contains("Promo") || inCard.Contains("Member"))) {
				edition = "DCI Legend Membership"
			}
		case "Faerie Conclave", "Treetop Village":
			if inCard.IsWPNGateway() || inCard.Contains("Summer") {
				edition = "Tenth Edition Promos"
			}
		case "Kamahl, Pit Fighter", "Char":
			if inCard.IsDCIPromo() || inCard.Contains("15th Anniversary") || inCard.IsGenericPromo() {
				edition = "15th Anniversary Cards"
			}
		case "Fling":
			if (inCard.IsDCIPromo() || inCard.IsWPNGateway()) && mtgmatcher.ExtractNumber(inCard.Variation) == "" {
				edition = "DCI Promos"
				if inCard.IsDCIPromo() {
					variation = "50"
				} else if inCard.IsWPNGateway() {
					variation = "69"
				}
			}
		case "Sylvan Ranger":
			if (inCard.IsDCIPromo() || inCard.IsWPNGateway()) && mtgmatcher.ExtractNumber(inCard.Variation) == "" {
				edition = "DCI Promos"
				if inCard.IsDCIPromo() {
					variation = "51"
				} else if inCard.IsWPNGateway() {
					variation = "70"
				}
			}
		case "Naya Sojourners":
			if inCard.IsGenericPromo() {
				edition = "DCI Promos"
			}
		case "Hall of Triumph":
			if inCard.IsGenericPromo() {
				edition = "Journey into Nyx Promos"
			}
		case "Reliquary Tower":
			if inCard.Contains("League") {
				edition = "Core Set 2019 Promos"
			} else if inCard.Contains("Bring a Friend") {
				edition = "Love Your LGS 2020"
			}
		case "Bolas's Citadel":
			if inCard.IsGenericPromo() {
				edition = "War of the Spark Promos"
			}
		case "Llanowar Elves":
			if inCard.IsGenericPromo() {
				edition = "Dominaria Promos"
			}
		case "Evolving Wilds":
			if inCard.IsGenericPromo() {
				edition = "Rivals of Ixalan Promos"
			}
		case "Teferi, Master of Time":
			num := mtgmatcher.ExtractNumber(variation)
			_, err := strconv.Atoi(num)
			if err == nil {
				if inCard.IsPrerelease() {
					variation = num + "s"
				} else if inCard.IsPromoPack() {
					variation = num + "p"
				}
			}
			if num == "" {
				if inCard.IsPrerelease() {
					variation = "75s"
				} else if inCard.IsPromoPack() {
					variation = "75p"
				} else if inCard.IsBorderless() {
					variation = "281"
				} else if inCard.IsShowcase() {
					variation = "290"
				} else {
					variation = "75"
				}
			}
			if strings.HasSuffix(variation, "s") || strings.HasSuffix(variation, "p") {
				edition = "Core Set 2021 Promos"
			}
		case "Mind Stone":
			switch edition {
			// Skip the check if this card already has the right edition
			case "DCI Promos",
				"Wizards Play Network 2021":
			default:
				if inCard.IsWPNGateway() || inCard.Contains("Bring a Friend") {
					edition = "Wizards Play Network 2021"
					if inCard.Contains("Gateway") {
						edition = "DCI Promos"
					}
				}
			}
		case "Runo Stromkirk", "Runo Stromkirk // Krothuss, Lord of the Deep":
			if inCard.IsShowcase() || mtgmatcher.Contains(inCard.Variation, "Eternal") {
				num := mtgmatcher.ExtractNumber(inCard.Variation)
				if num == "" {
					if mtgmatcher.Contains(inCard.Variation, "Eternal") {
						variation = "327"
					} else if inCard.IsShowcase() {
						variation = "316"
					}
				}
			}
		case "Diabolic Tutor":
			if inCard.IsIDWMagazineBook() {
				edition = "Secret Lair Drop"
			}
		case "Magister of Worth":
			if inCard.IsBaB() {
				variation = "Launch"
			}
		case "Hangarback Walker":
			if inCard.IsReskin() || inCard.IsGenericPromo() || strings.Contains(inCard.Edition, "LGS") {
				edition = "Love Your LGS 2020"
			}
		// Sometimes these cards are not marked as prerelease because they are showcase
		case "Goro-Goro and Satoru", "Katilda and Lier", "Slimefoot and Squee":
			if inCard.IsShowcase() && !inCard.IsPrerelease() {
				variation += " Prerelease"
			}
		// There are three Prerelease editions across two editions
		case "Delighted Halfling",
			"Lobelia Sackville-Baggins",
			"Frodo Baggins",
			"Bilbo, Retired Burglar",
			"Gandalf, Friend of the Shire",
			"Wizard's Rockets":
			if inCard.IsBorderless() && !inCard.IsPrerelease() {
				variation += " Prerelease"
			}
		case "Diabolic Edict":
			if inCard.IsIDWMagazineBook() {
				edition = "Media and Collaboration Promos"

				if strings.Contains(variation, "31") || inCard.IsJPN() || inCard.Language == "Japanese" {
					variation = "2019-2"
				} else {
					variation = "2024-5"
				}
			}
		case "Shock":
			if inCard.IsIDWMagazineBook() {
				edition = "Media and Collaboration Promos"

				if strings.Contains(variation, "32") || inCard.IsJPN() || inCard.Language == "Japanese" {
					variation = "2019-3"
				} else {
					variation = "2025-1"
				}
			}
		case "Duress":
			if inCard.IsIDWMagazineBook() {
				if inCard.Contains("IDW") {
					edition = "IDW Comics Inserts"
					variation = "17"
				} else {
					edition = "Media and Collaboration Promos"

					if strings.Contains(variation, "34") || inCard.IsJPN() || inCard.Language == "Japanese" {
						variation = "2019-6"
					} else {
						variation = "2025-7"
					}
				}
			}
		case "Voltaic Key":
			if inCard.IsIDWMagazineBook() {
				edition = "Media and Collaboration Promos"

				if strings.Contains(variation, "35") || inCard.IsJPN() || inCard.Language == "Japanese" {
					variation = "2020-1"
				} else {
					variation = "2025-4"
				}
			}
		case "Dark Ritual":
			if inCard.IsIDWMagazineBook() {
				edition = "Media and Collaboration Promos"

				if strings.Contains(variation, "38") || inCard.IsJPN() || inCard.Language == "Japanese" {
					variation = "2020-4"
				} else {
					variation = "2025-8"
				}
			}
		case "Arcbound Ravager":
			if inCard.Contains("Qualifiers") || inCard.Contains("WMCQ") {
				edition = "Pro Tour Promos"
			}
		case "Hornet Queen":
			if inCard.Contains("Specimen") {
				variation = strings.Replace(variation, "73", "", -1)
			}
		default:
			// Attempt a best effor match for known promotional tags if card or edition
			// wasn't found in previous steps
			if inCard.IsGenericPromo() {
				mtgmatcher.Logger.Printf("Precise matching for promo failed, attempting best effort")
				inCard.PromoWildcard = true
			}
		}
	}
	inCard.Edition = edition
	inCard.Variation = variation

	// Adjust incorrect numbers sometimes used for Etched
	num := mtgmatcher.ExtractNumber(inCard.Variation)
	if num != "" && strings.HasSuffix(num, "e") && mtgmatcher.HasEtchedPrinting(inCard.Name, inCard.Edition) {
		fixedNum := strings.TrimSuffix(num, "e")
		variation = strings.Replace(variation, num, fixedNum, -1)
		if !mtgmatcher.Contains(variation, "Etched") {
			variation += " Etched"
		}
	}
	inCard.Variation = variation
}

func (Rules) FilterPrintings(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, editions []string) (printings []string) {
	maybeYear := mtgmatcher.ExtractYear(inCard.Variation)
	if maybeYear == "" {
		maybeYear = mtgmatcher.ExtractYear(inCard.Edition)
	}

	for _, setCode := range editions {
		set, found := b.Sets[setCode]
		if !found {
			continue
		}

		setDate := set.ReleaseDateTime

		switch {
		// If the edition matches, use it as is
		// except for two "catch all" sometimes overlapping sets
		case mtgmatcher.Equals(inCard.Edition, set.Name) && !inCard.IsMysteryList() && !inCard.IsSecretLair():
			// pass-through

		case inCard.IsPrerelease():
			switch set.Name {
			// Sets that could be marked as prerelease, but they aren't really
			case "M15 Prerelease Challenge",
				"Open the Helvault":
			// Sets that have prerelease cards mixed in
			case "Innistrad: Double Feature",
				"March of the Machine Commander",
				"The Lord of the Rings: Tales of Middle-earth":
				skip := true
				foundCards := b.MatchInSet(inCard.Name, setCode)
				for _, card := range foundCards {
					if card.HasPromoType(mtgmatcher.PromoTypePrerelease) {
						skip = false
						break
					}
				}
				if skip {
					continue
				}
			case "Duels of the Planeswalkers 2012 Promos",
				"Grand Prix Promos",
				"Pro Tour Promos",
				"World Championship Promos":
				continue
			default:
				if strings.HasPrefix(set.Name, "30th Anniversary") {
					continue
				}
				if !strings.HasSuffix(set.Name, "Promos") {
					continue
				}
			}

		case inCard.IsPromoPack():
			switch set.Name {
			case "Dragon's Maze Promos", // due to Plains
				"Grand Prix Promos":
				continue
			case "M20 Promo Packs":
			default:
				switch {
				case strings.HasPrefix(set.Name, "30th Anniversary"):
					continue
				case strings.HasSuffix(set.Name, "Promos"):
				case setDate.After(mtgmatcher.PromosForEverybodyYay) && (set.Type == "expansion" || set.Type == "core"):
					skip := true
					foundCards := b.MatchInSet(inCard.Name, setCode)
					for _, card := range foundCards {
						if card.HasPromoType(mtgmatcher.PromoTypePromoPack) || card.HasPromoType(mtgmatcher.PromoTypePlayPromo) {
							skip = false
							break
						}
					}
					if skip {
						continue
					}
				default:
					continue
				}
			}

		case inCard.IsRelease():
			skip := true
			foundCards := b.MatchInSet(inCard.Name, setCode)
			for _, card := range foundCards {
				if card.HasPromoType(mtgmatcher.PromoTypeRelease) ||
					card.HasPromoType(mtgmatcher.PromoTypeDraftWeekend) ||
					card.HasPromoType(mtgmatcher.PromoTypeWPN) {
					skip = false
					break
				}
			}
			if skip {
				continue
			}

		case inCard.IsBaB():
			skip := true
			foundCards := b.MatchInSet(inCard.Name, setCode)
			for _, card := range foundCards {
				if card.HasPromoType(mtgmatcher.PromoTypeBuyABox) {
					skip = false
					break
				}
			}
			if skip {
				continue
			}

		case inCard.IsBundle():
			skip := true
			foundCards := b.MatchInSet(inCard.Name, setCode)
			for _, card := range foundCards {
				if card.HasPromoType(mtgmatcher.PromoTypeBundle) {
					skip = false
					break
				}
			}
			if skip {
				continue
			}

		case inCard.IsFNM():
			switch {
			case strings.HasPrefix(set.Name, "Friday Night Magic "+maybeYear):
			case set.Name == "Magic × Duel Masters Promos":
			case strings.HasSuffix(set.Name, "Promos"):
				skip := true
				foundCards := b.MatchInSet(inCard.Name, setCode)
				for _, card := range foundCards {
					if card.HasPromoType(mtgmatcher.PromoTypeFNM) {
						inCard.Variation = "FNM Promo"
						skip = false
						break
					}
				}
				if skip {
					continue
				}
			default:
				continue
			}

		case inCard.IsJudge():
			switch {
			case strings.HasPrefix(set.Name, "Judge Gift Cards "+maybeYear):
			default:
				continue
			}

		case inCard.IsArena():
			maybeYear = inCard.ArenaYear(maybeYear)
			switch {
			case set.Name == "DCI Legend Membership":
			case strings.HasPrefix(set.Name, "Arena League "+maybeYear):
			default:
				continue
			}

		// This needs to be above any possible printing type below
		// Both kinds need to be checked in the same place as there is
		// a lot of overlap in the product and naming across stores
		case inCard.IsMysteryList() || inCard.IsSecretLair():
			noSymbol := inCard.Contains("No") && inCard.Contains("Symbol")
			switch set.Code {
			case "CMB1":
				if noSymbol || strings.Contains(inCard.Variation, "V.2") {
					continue
				}
			case "CMB2":
				if !noSymbol && !strings.Contains(inCard.Variation, "V.2") {
					continue
				}
			case "MB2":
				if !inCard.Contains("Mystery Booster 2") {
					continue
				}
			case "PLST":
				// Check if there is an exact match in plain SLD
				num := mtgmatcher.ExtractNumber(inCard.Variation)
				if len(b.MatchInSetNumber(inCard.Name, "SLD", num)) != 0 {
					// If there is a match, make sure there are no other cards in PLST with the same number
					shouldNotContinue := false
					cardsWithSameName := b.MatchInSet(inCard.Name, "PLST")
					for _, altCard := range cardsWithSameName {
						var altNum string
						altNums := strings.Split(altCard.Number, "-")
						if len(altNums) > 1 {
							altNum = altNums[1]
						}
						if altNum == num {
							shouldNotContinue = true
							break
						}
					}
					if !shouldNotContinue {
						continue
					}
				}
				if inCard.IsSecretLair() {
					skip := true
					for _, name := range b.SLDDeckNames {
						if mtgmatcher.Contains(inCard.Edition, name) || mtgmatcher.Contains(inCard.Variation, name) {
							skip = false
						}
					}
					if skip {
						continue
					}
				}
			case "ULST":
			case "SLX", "SLU", "SLC", "SLP":
				// If these have no strict matches AND are not properly tagged, skip them
				if len(b.MatchInSetNumber(inCard.Name, set.Code, mtgmatcher.ExtractNumber(inCard.Variation))) == 0 && !inCard.HasSecretLairTag(set.Code) {
					continue
				}
			case "SLD":
				skip := false

				// Iterate on all possible combinations of tags, and skip if a
				// condition is unmet
				for _, code := range []string{"SLU", "SLX", "SLC", "SLP"} {
					// The only card with the same name within and without
					if code == "SLX" && inCard.Name == "Themberchaud" {
						continue
					}
					if len(b.MatchInSet(inCard.Name, code)) > 0 && inCard.HasSecretLairTag(code) {
						skip = true
						break
					}
				}

				// No PLST in SLD
				if inCard.IsMysteryList() {
					skip = true
				}

				// Check that the card reported is not coming from a SLD Deck
				// or if it does, make sure it is actually from SLD
				if len(b.MatchInSetNumber(inCard.Name, "SLD", mtgmatcher.ExtractNumber(inCard.Variation))) == 0 && len(b.MatchInSet(inCard.Name, "PLST")) > 0 {
					for _, name := range b.SLDDeckNames {
						deckNameInCard := mtgmatcher.Contains(inCard.Edition, name) || mtgmatcher.Contains(inCard.Variation, name)
						if deckNameInCard {
							skip = true
							break
						}
					}
				}

				if skip {
					continue
				}
			default:
				continue
			}

		// Some providers use "Textless" for MF cards
		case inCard.IsRewards() && !inCard.IsMagicFest():
			maybeYear = inCard.PlayerRewardsYear(maybeYear)
			switch {
			case strings.HasPrefix(set.Name, "Magic Player Rewards "+maybeYear):
			default:
				continue
			}

		case inCard.IsWPNGateway():
			switch set.Name {
			case "DCI Promos":
			case "Innistrad: Crimson Vow",
				"The Lost Caverns of Ixalan":
				skip := true
				foundCards := b.MatchInSet(inCard.Name, set.Code)
				for _, card := range foundCards {
					if card.HasPromoType(mtgmatcher.PromoTypeWPN) {
						skip = false
						break
					}
				}
				if skip {
					continue
				}
			default:
				switch {
				case strings.HasPrefix(set.Name, "Wizards Play Network "+maybeYear):
				case strings.HasPrefix(set.Name, "Love Your LGS "+maybeYear):
				default:
					continue
				}
			}

		case inCard.IsIDWMagazineBook():
			// No Media cards in these sets
			switch set.Code {
			case "P30A", "P30H", "P30M":
				continue
			case "PGPX", "PWOR", "PPRO", "PWCS":
				continue
			}

			switch {
			case !inCard.IsJPN() && (set.Name == "IDW Comics Inserts" || set.Name == "HarperPrism Book Promos"):
			case !inCard.IsJPN() && strings.HasPrefix(set.Name, "Duels of the Planeswalkers "+maybeYear):
			default:
				switch set.Code {
				case "PURL", "JP1", "DLGM":
				case "PDOM":
					// This set contains both FNM and Media cards
					skip := false
					foundCards := b.MatchInSet(inCard.Name, set.Code)
					for _, card := range foundCards {
						if card.HasPromoType(mtgmatcher.PromoTypeFNM) {
							skip = true
							break
						}
					}
					if skip {
						continue
					}
				case "P9ED":
					if inCard.IsJPN() {
						continue
					}
				case "PMEI":
					// This is the only card present in IDW and Media Inserts
					// so make sure it is properly tagged
					if inCard.Name == "Duress" && !inCard.IsJPN() {
						continue
					}
					// This could be mixed in P9ED Russian
					if inCard.Name == "Shivan Dragon" && !inCard.IsJPN() {
						continue
					}
				default:
					if !strings.HasSuffix(set.Name, "Promos") {
						continue
					}
				}
			}

		case inCard.IsResale():
			switch set.Code {
			case "DCI", "P30A", "P30H", "P30M":
				continue
			case "PDOM":
				// This might conflict with Llanowar Elves
				continue
			case "PMEI", "PLTC":
				// These sets may actually contain Resale cards
			default:
				if !strings.HasSuffix(set.Name, "Promos") {
					continue
				}
			}

		case inCard.Contains("Hero") && inCard.Contains("Path"):
			switch set.Name {
			case "Born of the Gods Hero's Path",
				"Journey into Nyx Hero's Path",
				"Journey into Nyx Promos",
				"Theros Hero's Path",
				"Defat a God",
				"Face the Hydra",
				"Battle the Horde":
			default:
				continue
			}

		case inCard.Contains("Convention"):
			switch set.Name {
			case "URL/Convention Promos":
			default:
				if strings.HasPrefix(set.Name, "30th Anniversary") {
					continue
				}
				if !strings.HasSuffix(set.Name, "Promos") {
					continue
				}
			}

		case inCard.Contains("Premier Play") || inCard.Contains("Regional Championship Qualifiers"):
			switch {
			case set.Name == "Pro Tour Promos":
				// Special case for a card that could be in PR23
				if inCard.Name == "Snapcaster Mage" && !inCard.Contains("Pro Tour") {
					continue
				}
			case strings.HasPrefix(set.Name, "Regional Championship Qualifiers "+maybeYear):
			default:
				continue
			}

		case inCard.Contains("Game Day") || inCard.Contains("Store Championship"):
			switch set.Code {
			case "SCH":
			case "LTR":
			case "PEWK":
			default:
				skip := true
				switch {
				case strings.HasSuffix(set.Name, "Promos"):
					foundCards := b.MatchInSet(inCard.Name, set.Code)
					for _, card := range foundCards {
						if card.HasPromoType(mtgmatcher.PromoTypeStoreChampionship) ||
							card.HasPromoType(mtgmatcher.PromoTypeGameDay) {
							skip = false
							break
						}
					}
				case strings.HasPrefix(set.Name, "Wizards Play Network "+maybeYear):
					skip = false
				}
				if skip {
					continue
				}
			}

		case inCard.IsWorldChamp():
			switch {
			case (maybeYear == "1996" || maybeYear == "") && set.Name == "Pro Tour Collector Set":
			case maybeYear != "" && strings.HasPrefix(set.Name, "World Championship Decks "+maybeYear):
			case maybeYear == "" && strings.HasPrefix(set.Name, "World Championship Decks"):
				skip := true
				num, _ := mtgmatcher.ParseWorldChampPrefix(inCard.Variation)
				foundCards := b.MatchInSet(inCard.Name, set.Code)
				if num == "" || len(foundCards) == 1 {
					skip = false
				} else {
					for _, card := range foundCards {
						if card.Number == num {
							skip = false
							break
						}
					}
				}
				if skip {
					continue
				}
			default:
				continue
			}

		case inCard.IsMagicFest():
			// Some providers use GP2018 instead of MF2019
			if maybeYear == "2018" && isBasicLand(inCard) {
				maybeYear = "2019"
			}
			switch {
			case strings.HasPrefix(set.Name, "MagicFest "+maybeYear):
				if len(b.MatchInSet(inCard.Name, "SLP")) > 0 && !inCard.Contains("Fest") {
					continue
				}
			case set.Code == "PLG21":
			case set.Code == "PEWK":
			case set.Code == "SLP":
				// If the 'Secret' tag is missing, confirm that this could not be found in other
				// MagicFest sets
				if (len(b.MatchInSet(inCard.Name, "PF19")) > 0 ||
					len(b.MatchInSet(inCard.Name, "PF25")) > 0) && !inCard.Contains("Secret") {
					continue
				}
			default:
				continue
			}

		case inCard.IsSDCC():
			switch {
			case strings.HasPrefix(set.Name, "San Diego Comic-Con "+maybeYear):
			default:
				continue
			}

		case inCard.IsDuelsOfThePW():
			switch {
			case strings.HasPrefix(set.Name, "Duels of the Planeswalkers"):
			default:
				continue
			}

		// DDA with number or deck variant specified in the variation
		case inCard.IsDuelDecksAnthology():
			switch {
			case strings.HasPrefix(set.Name, "Duel Decks Anthology"):
				found := false
				// Look if the variation or the edition contain part of set name
				for _, location := range []string{inCard.Variation, inCard.Edition} {
					if !strings.Contains(strings.ToLower(location), "vs") {
						continue
					}
					fields := strings.Fields(location)
					for _, field := range fields {
						// Skip elements that are too short to be representative
						if len(field) < 4 {
							continue
						}
						if mtgmatcher.Contains(set.Name, field) {
							found = true
							break
						}
					}
				}
				// Do number check only if well known elements are missing
				wellKnownTags := inCard.Contains("Divine") || inCard.Contains("Garruk") ||
					inCard.Contains("Chandra") || inCard.Contains("Goblins")
				if !found && !wellKnownTags {
					num := mtgmatcher.ExtractNumber(inCard.Variation)
					if num != "" {
						foundCards := b.MatchInSet(inCard.Name, setCode)
						for _, card := range foundCards {
							if card.Number == num {
								found = true
								break
							}
						}
					}
				}
				if wellKnownTags && !found {
					continue
				}
			default:
				continue
			}

		case inCard.IsDuelDecks():
			variant := inCard.DuelDecksVariant()
			switch {
			case strings.HasPrefix(set.Name, "Duel Decks") &&
				!strings.Contains(set.Name, "Anthology"):
				if !mtgmatcher.Contains(set.Name, variant) {
					continue
				}
			default:
				continue
			}

		case strings.Contains(inCard.Variation, "Champs") ||
			strings.Contains(inCard.Variation, "States"):
			switch set.Name {
			case "Champs and States":
			case "Grand Prix Promos":
				continue
			default:
				if strings.HasPrefix(set.Name, "30th Anniversary") {
					continue
				}
				if !strings.HasSuffix(set.Name, "Promos") {
					continue
				}
			}

		case inCard.IsPremiereShop():
			if maybeYear == "" {
				guilds := append(mtgmatcher.GRNGuilds, mtgmatcher.ARNGuilds...)
				for _, guild := range guilds {
					if strings.Contains(inCard.Variation, guild) {
						maybeYear = "2005"
						break
					}
				}
				if maybeYear == "" && strings.HasSuffix(inCard.Variation, "Cycle") {
					maybeYear = map[string]string{
						"Time Spiral Cycle":       "2006",
						"Lorwyn Cycle":            "2007",
						"Shards of Alara Cycle":   "2008",
						"Zendikar Cycle":          "2009",
						"Scars of Mirrodin Cycle": "2010",
						"Innistrad Cycle":         "2011",
					}[inCard.Variation]
				}
			}
			switch {
			case strings.HasPrefix(set.Name, "Magic Premiere Shop "+maybeYear):
			default:
				continue
			}

		case isBasicLand(inCard) && strings.Contains(inCard.Variation, "APAC"):
			if set.Name != "Asia Pacific Land Program" {
				continue
			}

		case isBasicLand(inCard) && mtgmatcher.Contains(inCard.Variation, "EURO"):
			if set.Name != "European Land Program" {
				continue
			}

		case strings.Contains(inCard.Edition, "Core Set") ||
			strings.Contains(inCard.Edition, "Core 20") ||
			strings.Contains(inCard.Edition, "Magic 20"):
			switch {
			case !inCard.IsGenericPromo() && strings.HasSuffix(set.Name, "Promos"):
				continue
			case strings.HasPrefix(set.Name, "Core Set "+maybeYear):
			case strings.HasPrefix(set.Name, "Magic "+maybeYear):
			default:
				continue
			}

		case inCard.Contains("Grand Prix"):
			switch {
			case strings.HasPrefix(set.Name, "MagicFest "+maybeYear):
			default:
				if set.Name != "Grand Prix Promos" {
					continue
				}
			}

		case inCard.Contains("Lunar New Year") || inCard.Contains("Year of the "):
			switch set.Code {
			case "PLNY":
			default:
				if !strings.HasPrefix(set.Name, "Year of the ") {
					continue
				}
			}

		case inCard.IsThickDisplay():
			switch set.Code {
			// The sets with thick display cards separate from the main commander set
			case "OC21", "OAFC", "OMIC", "OVOC":
			// SLD may contain DFC with thick display
			case "SLD":
			default:
				// Skip any set before this date if not from the sets above
				if setDate.Before(mtgmatcher.SeparateFinishCollectorNumberDate) {
					continue
				}
			}

		case inCard.Contains("Bring-A-Friend") ||
			inCard.Contains("Love Your LGS") ||
			inCard.Contains("Love Your Local Game Store") ||
			inCard.Contains("Welcome Back") ||
			inCard.Contains("Open House") ||
			inCard.Contains("LGS Promo"):
			switch {
			case strings.HasPrefix(set.Name, "Love Your LGS "+maybeYear):
			case strings.HasPrefix(set.Name, "Wizards Play Network "+maybeYear):
			case set.Name == "Grand Prix Promos":
				continue
			default:
				if strings.HasPrefix(set.Name, "30th Anniversary") ||
					strings.HasPrefix(set.Name, "Duels of the Planeswalkers") {
					continue
				}
				if !strings.HasSuffix(set.Name, "Promos") {
					continue
				}
			}

		case inCard.Contains("30th"):
			switch set.Code {
			case "P30A", "P30H", "P30M":
			case "P30T":
				if inCard.IsRetro() || !inCard.IsJPN() {
					continue
				}
			default:
				continue
			}

		case inCard.Contains("Planeswalker") && inCard.Contains("Promos"):
			switch set.Code {
			case "PWCS":
			default:
				if !strings.HasSuffix(set.Name, "Promos") {
					continue
				}
			}

		// For all the promos with "Extended Art" which refer to the full art promo
		case inCard.IsExtendedArt() && !inCard.Contains("Game Day"):
			if setDate.Before(mtgmatcher.PromosForEverybodyYay) {
				switch set.Code {
				case "DCI":
				default:
					continue
				}
			}

		case inCard.Contains("Eternal Weekend"):
			switch set.Code {
			case "PEWK":
			default:
				continue
			}

		// Last resort, if this is set on the input card, and there were
		// no better descriptors earlier, try looking at the set type
		case inCard.PromoWildcard:
			switch set.Type {
			case "promo":
				// Skip common promos, they are usually correctly listed
				if strings.HasPrefix(set.Name, "Judge Gift") ||
					strings.HasPrefix(set.Name, "30th Anniversary") {
					continue
				}
				skip := false
				foundCards := b.MatchInSet(inCard.Name, setCode)
				// It is required to set a proper tag to parse non-English
				// cards or well-known promos
				for _, card := range foundCards {
					if card.Language == mtgmatcher.LanguageJapanese {
						skip = true
						break
					}
				}

				if skip {
					continue
				}
			case "starter":
				if !strings.HasSuffix(set.Name, "Clash Pack") {
					continue
				}
			case "expansion", "core", "masters", "draft_innovation":
				skip := true
				foundCards := b.MatchInSet(inCard.Name, setCode)
				for _, card := range foundCards {
					// Skip boosterfun because they are inherently non-promo
					if card.IsPromo && !card.HasPromoType(mtgmatcher.PromoTypeBoosterfun) {
						skip = false
						break
					}
				}
				if skip {
					continue
				}
			case "box":
				skip := true
				switch setCode {
				// Only keep the planeswalkers from SLD for this category
				case "SLD":
					foundCards := b.MatchInSet(inCard.Name, setCode)
					for _, card := range foundCards {
						if slices.Contains(card.Types, "Planeswalker") {
							skip = false
							break
						}
					}
				}
				if skip {
					continue
				}
			case "funny":
				if set.Name != "HasCon 2017" {
					continue
				}
			default:
				continue
			}

		// Tokens need correct set names or special handling earlier
		case (strings.HasSuffix(inCard.Name, "Token") &&
			b.UUIDs[b.Hashes[mtgmatcher.Normalize(inCard.Name)][0]].Layout == "token") ||
			(!strings.HasSuffix(inCard.Name, "Token") &&
				b.UUIDs[b.Hashes[mtgmatcher.Normalize(inCard.Name)][0]].Layout == "token"):
			if !mtgmatcher.Equals(inCard.Edition, set.Name) {
				continue
			}
		}

		printings = append(printings, setCode)
	}

	return
}

func (Rules) FilterCards(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, cardSet map[string][]mtgmatcher.Card) (outCards []mtgmatcher.Card) {
	for setCode, inCards := range cardSet {
		set := b.Sets[setCode]

		for _, card := range inCards {
			// Super lucky case, we were expecting the card
			num, found := mtgmatcher.VariantsTable[set.Name][card.Name][strings.ToLower(inCard.Variation)]
			if found {
				if num == card.Number {
					outCards = append(outCards, card)
				}

				// If a variant is expected we assume that all cases are covered
				continue
			}

			checkNum := true
			// Lucky case, variation is just the collector number
			num = mtgmatcher.ExtractNumber(inCard.Variation)
			// Special case for SLD, finally breaking the check against years
			if num == "" && card.SetCode == "SLD" {
				num = mtgmatcher.ExtractNumberAny(inCard.Variation)
			}
			if inCard.ShouldIgnoreNumber(set.Name, num) {
				checkNum = false
				mtgmatcher.Logger.Println("Skipping number check")
			}
			if checkNum && num != "" {
				// The empty string will allow to test the number without any
				// additional prefix first
				possibleSuffixes := []string{""}
				variation := strings.Replace(inCard.Variation, "-", "", 1)
				fields := strings.Fields(strings.ToLower(variation))
				possibleSuffixes = append(possibleSuffixes, fields...)

				// Check if edition-specific numbers need special suffixes
				numFilterFunc, found := numberFilterCallbacks[set.Code]
				if found {
					overrides := numFilterFunc(inCard)
					if overrides != nil {
						possibleSuffixes = overrides
					}
				} else if card.Identifiers["tcgplayerAlternativeFoilProductId"] != "" {
					// Special case when we deal with duplicated entries
					if inCard.Foil || inCard.IsFoil() {
						possibleSuffixes = []string{mtgmatcher.SuffixSpecial}
					} else {
						possibleSuffixes = []string{""}
					}
				}

				// Add any possible extra suffixes if we know what we're dealing with
				switch {
				case inCard.IsPrerelease():
					possibleSuffixes = append(possibleSuffixes, "s")
				case inCard.IsPromoPack():
					possibleSuffixes = append(possibleSuffixes, "p")
				case inCard.IsChineseAltArt():
					possibleSuffixes = append(possibleSuffixes, "s", mtgmatcher.SuffixSpecial+"s", mtgmatcher.SuffixVariant+"s")
				case inCard.IsSerialized():
					possibleSuffixes = append(possibleSuffixes, "z")
				case inCard.IsJudge() || inCard.IsResale():
					possibleSuffixes = append(possibleSuffixes, mtgmatcher.SuffixSpecial)
				case inCard.IsJPN():
					possibleSuffixes = append(possibleSuffixes, "jpn")
				case inCard.Edition == "Alternate Fourth Edition":
					possibleSuffixes = append(possibleSuffixes, "alt")
				}

				for _, numSuffix := range possibleSuffixes {
					// The self test is already expressed by the empty string
					// This avoids an odd case of testing 1.1 = 11
					if num == numSuffix {
						continue
					}
					number := strings.ToLower(num)
					if numSuffix != "" && !strings.HasSuffix(number, numSuffix) {
						number += numSuffix
					}

					if number == strings.ToLower(card.Number) {
						mtgmatcher.Logger.Println("Found match with card number", card.Number)
						outCards = append(outCards, card)

						// Card was found, skip any other suffix
						break
					}
				}

				// If a variant is a number we expect that this information is
				// reliable, so skip anything else
				continue
			}

			// The last-ditch effort from above - when this is set, only check
			// the non-promo sets as some promos can be mixed next to the
			// normal cards - in this way, promo sets can process as normal and
			// deduplicate all the various prerelease and promo packs
			switch set.Type {
			case "expansion", "core", "masters", "draft_innovation":
				if inCard.PromoWildcard &&
					!card.HasPromoType(mtgmatcher.PromoTypeBoosterfun) &&
					!card.HasPromoType(mtgmatcher.PromoTypePromoPack) &&
					!card.HasPromoType(mtgmatcher.PromoTypeStarterDeck) &&
					!card.HasPromoType(mtgmatcher.PromoTypeIntroPack) &&
					!strings.HasSuffix(card.Number, mtgmatcher.SuffixSpecial) &&
					!card.IsPromo {
					continue
				}
			}

			outCards = append(outCards, card)
		}
	}

	// Sort through the array of promo types
	if len(outCards) > 1 {
		var filteredOutCards []mtgmatcher.Card
		for _, card := range outCards {
			set, found := b.Sets[card.SetCode]
			if !found {
				continue
			}
			setDate := set.ReleaseDateTime

			var shouldContinue bool
			for _, promoElement := range promoTypeElements {
				if setDate.Before(promoElement.ValidDate) {
					continue
				}
				if promoElement.CanBeWild && inCard.PromoWildcard {
					continue
				}

				var tagPresent bool
				if promoElement.TagFunc != nil {
					tagPresent = promoElement.TagFunc(inCard)
				} else {
					for _, tag := range promoElement.Tags {
						if inCard.Contains(tag) {
							tagPresent = true
							break
						}
					}
				}

				if tagPresent && !card.HasPromoType(promoElement.PromoType) {
					shouldContinue = true
					break
				} else if !tagPresent && card.HasPromoType(promoElement.PromoType) {
					shouldContinue = true
					break
				}
			}
			if shouldContinue {
				continue
			}

			if inCard.BeyondBaseSet {
				// Filter out any card that is located in the base set only
				num, err := strconv.Atoi(card.Number)
				if err == nil && num < set.BaseSetSize {
					continue
				}
			}
			filteredOutCards = append(filteredOutCards, card)
		}

		// Don't throw away what was found if filtering checks is too aggressive
		if len(filteredOutCards) > 0 {
			outCards = filteredOutCards
		}
	}

	// Sort through any custom per-edition filter(s)
	if len(outCards) > 1 {
		var filteredOutCards []mtgmatcher.Card
		for _, card := range outCards {
			cardFilterFunc, foundSimple := simpleFilterCallbacks[card.SetCode]
			cardFilterFuncs, foundComplex := complexFilterCallbacks[card.SetCode]
			if foundSimple {
				if cardFilterFunc(inCard, &card) {
					continue
				}
			} else if foundComplex {
				shouldContinue := false
				for _, fn := range cardFilterFuncs {
					if fn(inCard, &card) {
						shouldContinue = true
						break
					}
				}
				if shouldContinue {
					continue
				}
			} else {
				if misprintCheck(inCard, &card) {
					continue
				}
			}

			filteredOutCards = append(filteredOutCards, card)
		}

		// Don't throw away what was found if filtering checks is too aggressive
		if len(filteredOutCards) > 0 {
			outCards = filteredOutCards
		}
	}

	if len(outCards) > 1 {
		mtgmatcher.Logger.Println("Filtering status after main loop")
		for _, card := range outCards {
			mtgmatcher.Logger.Println(card.SetCode, card.Name, card.Number)
		}
	}

	// Check if there are multiple printings for Prerelease and Promo Pack cards
	// Sometimes these contain the ParentCode or the parent edition name in the field
	if len(outCards) > 1 && (inCard.IsPrerelease() || inCard.IsPromoPack()) {
		allSameEdition := true
		for _, card := range outCards {
			if card.Name != outCards[0].Name || !strings.HasPrefix(card.SetCode, "P") {
				allSameEdition = false
				break
			}
		}

		if allSameEdition {
			mtgmatcher.Logger.Println("allSameEdition pass needed")
			var filteredOutCards []mtgmatcher.Card
			for _, card := range outCards {
				set := b.Sets[card.SetCode]
				// The year is necessary to decouple PM20 and PM21 cards
				year := mtgmatcher.ExtractYear(set.Name)
				// Check if the parent set code is present in the variation or edition
				if strings.Contains(inCard.Variation, set.ParentCode) ||
					strings.Contains(inCard.Edition, set.ParentCode) ||
					(year != "" && inCard.Contains(year)) {
					filteredOutCards = append(filteredOutCards, card)
				} else {
					for probe, number := range mtgmatcher.MultiPromosTable[set.Name][card.Name] {
						if inCard.Contains(probe) && mtgmatcher.ExtractNumberValue(card.Number) == number {
							filteredOutCards = append(filteredOutCards, card)
						}
					}
				}
			}

			// Don't throw away what was found if filtering checks is too aggressive
			if len(filteredOutCards) > 0 {
				outCards = filteredOutCards
			}
		}
	}

	if len(outCards) > 1 && mtgmatcher.ExtractNumber(inCard.Variation) == "" {
		// Separate finishes have different collector numbers after this date
		if len(outCards) > 1 {
			var filteredOutCards []mtgmatcher.Card
			for _, card := range outCards {
				set, found := b.Sets[card.SetCode]
				if !found {
					continue
				}
				setDate := set.ReleaseDateTime
				if setDate.After(mtgmatcher.SeparateFinishCollectorNumberDate) && etchedCheck(inCard, &card) {
					continue
				}
				filteredOutCards = append(filteredOutCards, card)
			}

			// Don't throw away what was found if filtering checks are too aggressive
			if len(filteredOutCards) > 0 {
				outCards = filteredOutCards
			}
		}

		// If above filters were not enough, check the boder, but dont skip card if it's showcase
		// ExtendedArt is fine as there cannot be a borderless one
		if len(outCards) > 1 {
			var filteredOutCards []mtgmatcher.Card
			for _, card := range outCards {
				if borderlessCheck(inCard, &card) {
					continue
				}
				filteredOutCards = append(filteredOutCards, card)
			}

			// Don't throw away what was found if filtering checks are too aggressive
			if len(filteredOutCards) > 0 {
				outCards = filteredOutCards
			}
		}

		// If above filters were not enough, check the frameEffects
		// Extended Art
		if len(outCards) > 1 {
			var filteredOutCards []mtgmatcher.Card
			for _, card := range outCards {
				// This needs date check because some old full art promos are marked
				// as extended art, in a different way of what modern Extended Art is
				set, found := b.Sets[card.SetCode]
				if !found {
					continue
				}
				setDate := set.ReleaseDateTime
				if setDate.After(mtgmatcher.PromosForEverybodyYay) && extendedartCheck(inCard, &card) {
					continue
				}
				filteredOutCards = append(filteredOutCards, card)
			}

			// Don't throw away what was found if filtering checks are too aggressive
			if len(filteredOutCards) > 0 {
				outCards = filteredOutCards
			}
		}

		// Showcase
		if len(outCards) > 1 {
			var filteredOutCards []mtgmatcher.Card
			for _, card := range outCards {
				if showcaseCheck(inCard, &card) {
					continue
				}
				filteredOutCards = append(filteredOutCards, card)
			}

			// Don't throw away what was found if filtering checks are too aggressive
			if len(filteredOutCards) > 0 {
				outCards = filteredOutCards
			}
		}
	}

	return
}

func (Rules) Prefilter(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	switch inCard.Name {
	case "Red Herring",
		"Bind // Liberate",
		"Pick Your Poison":
		if inCard.IsMysteryList() || inCard.Contains("Playtest") {
			inCard.Name += " Playtest"
		}
	case "Unquenchable Fury":
		if inCard.Contains("Battle the Horde") || inCard.Contains("Hero's Path") {
			inCard.Name += " Token"
		}
	case "Shapeshifter":
		if !(inCard.Contains("Edition") ||
			inCard.Contains("Foreign") ||
			inCard.Contains("Antiquities") ||
			inCard.Contains("Reinassance") ||
			inCard.Contains("Rinascimento")) {
			inCard.Name += " Token"
		}
	}
}

func (Rules) AdjustName(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	// Sticker sheet adjustments
	if strings.Contains(inCard.Name, "Sticker") {
		inCard.Name = strings.Replace(inCard.Name, "Sticker", "", 1)
		inCard.Name = strings.Replace(inCard.Name, "Sheet", "", 1)
		inCard.Name = strings.TrimSpace(inCard.Name)
	}

	// Skip for tokens, we need them to be exact or the prefix search interferes
	if strings.Contains(strings.ToLower(inCard.Name), "token") {
		return
	}
	_, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name+" Token")]
	if found {
		inCard.Name += " Token"
		return
	}
	if b.IsToken(inCard.Name) {
		return
	}

	// Move the card number from name to variation
	num := mtgmatcher.ExtractNumber(inCard.Name)
	if num != "" {
		fields := strings.Fields(inCard.Name)
		for i, field := range fields {
			if strings.Contains(field, num) {
				fields = append(fields[:i], fields[i+1:]...)
				break
			}
		}
		// Check card exists before updating the name
		tmpName := strings.Join(fields, " ")
		_, found := b.CanonicalNames[mtgmatcher.Normalize(tmpName)]
		if found {
			inCard.Name = tmpName
			inCard.AddToVariant(num)
			return
		}
	}

	// Move any single letter variation from name to beginning variation
	if inCard.IsBasicLand() {
		fields := strings.Fields(inCard.Name)
		if len(fields) > 1 {
			_, err := strconv.Atoi(strings.TrimPrefix(fields[1], "0"))
			isNum := err == nil
			isLetter := len(fields[1]) == 1

			if isNum || isLetter {
				oldVariation := inCard.Variation
				cuts := mtgmatcher.Cut(inCard.Name, " "+fields[1])

				inCard.Name = cuts[0]
				inCard.Variation = cuts[1]
				if oldVariation != "" {
					inCard.Variation += " " + oldVariation
				}
				return
			}
		}
	}

	// Rename cards that were translated differently
	if strings.Contains(inCard.Edition, "Ikoria") {
		if strings.Contains(inCard.Name, "Mothra's") && strings.Contains(inCard.Name, "Cocoon") {
			inCard.Name = "Mothra's Great Cocoon"
		} else if strings.Contains(inCard.Name, "Battra") {
			inCard.Name = "Battra, Dark Destroyer"
		} else if strings.Contains(inCard.Name, "Mechagodzilla") {
			inCard.Name = "Mechagodzilla, the Weapon"
		}
	}
	// Rename reskinned dual faced cards, only keep one side and keep the
	// flavor name, to make the following lookup in AlternateProps work
	if inCard.IsSecretLair() {
		if strings.Contains(inCard.Name, "Hawkins National") {
			inCard.Name = "Hawkins National Laboratory"
		} else if strings.Contains(inCard.Name, "Plains") && strings.Contains(inCard.Name, "Battlefield Forge") {
			if inCard.Contains("Unpeeled") || inCard.Contains("669") {
				inCard.Name = "Battlefield Forge"
				inCard.Variation = "669"
			} else if inCard.Contains("Peeled") || inCard.Contains("670") {
				inCard.Name = "Plains"
				inCard.Variation = "670"
			}
		}
	}
	// Check if this card may be known as something else
	for altName, altProps := range b.AlternateProps {
		if !mtgmatcher.Equals(altName, inCard.Name) {
			continue
		}

		// Stash the number for later decoupling if available
		if altProps.OriginalNumber != "" {
			inCard.AddToVariant(altProps.OriginalNumber)
		}

		inCard.Name = altProps.OriginalName
		if altProps.IsFlavor {
			inCard.AddToVariant("Reskin")
		}

		// Adjust the token name in case it's a reskin
		if b.IsToken(inCard.Name) {
			inCard.Name += " Token"
			return
		}

		return
	}

	// Special case for Un-sets that sometimes drop the parenthesis
	if strings.Contains(inCard.Edition, "The List") ||
		strings.Contains(inCard.Edition, "Unglued") || inCard.Edition == "UGL" ||
		strings.Contains(inCard.Edition, "Unhinged") || inCard.Edition == "UNH" ||
		strings.Contains(inCard.Edition, "Unstable") || inCard.Edition == "UST" ||
		strings.Contains(inCard.Edition, "Unsanctioned") || inCard.Edition == "UND" {
		if mtgmatcher.HasPrefix(inCard.Name, "B.F.M.") || mtgmatcher.HasPrefix(inCard.Name, "BFM") {
			cardName := inCard.Name
			inCard.Name = "B.F.M. (Big Furry Monster)"
			if mtgmatcher.Contains(cardName, "Left") || mtgmatcher.Contains(inCard.Variation, "Left") {
				inCard.Variation = "28"
			} else if mtgmatcher.Contains(cardName, "Right") || mtgmatcher.Contains(inCard.Variation, "Right") {
				inCard.Variation = "29"
			}
			return
		}
		if mtgmatcher.HasPrefix(inCard.Name, "Our Market Research") {
			inCard.Name = mtgmatcher.LongestCardEver
			return
		}
		if mtgmatcher.HasPrefix(inCard.Name, "The Ultimate Nightmare") {
			inCard.Name = mtgmatcher.NightmareCard
			return
		}
		if mtgmatcher.Contains(inCard.Name, "Surgeon") && mtgmatcher.Contains(inCard.Name, "Commander") {
			inCard.Name = "Surgeon ~General~ Commander"
			return
		}
		if mtgmatcher.Contains(inCard.Name, "Who") && mtgmatcher.Contains(inCard.Name, "What") &&
			mtgmatcher.Contains(inCard.Name, "When") && mtgmatcher.Contains(inCard.Name, "Where") &&
			mtgmatcher.Contains(inCard.Name, "Why") {
			inCard.Name = "Who // What // When // Where // Why"
			return
		}

		uuids, err := b.SearchHasPrefix(inCard.Name)
		if err == nil {
			inCard.Name = b.UUIDs[uuids[0]].Name
			return
		}
	}

	// Rename a DFC with same name
	splits := strings.Split(inCard.Name, "//")
	if len(splits) == 2 && strings.TrimSpace(splits[0]) == strings.TrimSpace(splits[1]) {
		inCard.Name = strings.TrimSpace(splits[0])
		return
	}

	// Altenatively try checking across any prefix, as long as it's a double
	// sided card, for some particular cases, like meld cards, or Treasure Chest
	// Attempt first to check cards in the same edition if possible
	// Skip for tokens
	for _, set := range b.Sets {
		if mtgmatcher.Equals(set.Name, inCard.Edition) {
			for _, card := range set.Cards {
				if card.Layout != "normal" && card.Layout != "token" && mtgmatcher.HasPrefix(card.Name, inCard.Name) {
					inCard.Name = card.Name
					return
				}
			}
		}
	}
	uuids, _ := b.SearchHasPrefix(inCard.Name)
	for _, uuid := range uuids {
		co, _ := b.GetUUID(uuid)
		if co.Layout != "normal" && co.Layout != "token" {
			inCard.Name = co.Name
			return
		}
	}
}

// isBasicLand mirrors core's strict (exact-name) basic-land check for the
// moved Magic logic; the core method is removed once all callers move.
func isBasicLand(c *mtgmatcher.InputCard) bool {
	switch c.Name {
	case "Plains", "Island", "Swamp", "Mountain", "Forest", "Wastes":
		return true
	}
	return false
}
