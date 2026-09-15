package magic

// verifiedNoUpstreamPairs is every two-sided token pairing confirmed real by
// vendor cross-verification rather than by mtgjson's own tokenProducts feed:
// both faces independently anchored to exactly one real printing each by a
// vendor's own sku/collector-number data (starcitygames's tokenPairSkuAnchors,
// cardtrader's tokenPairNumbers), verified against a real fetch of each
// vendor's live catalog, where TCGplayer's own tokenProducts feed simply never
// linked those two uuids as one product - not a matching failure, a data gap
// one level up (see deriveTokenPairs's own noUsableID exclusion). Minted the
// same way as every other derived pairing (buildDerivedCard), just with no
// usable TCGplayer id at all - see mintVerifiedPairs and
// Identifiers["vendorVerifiedPair"].
//
// A pairing here has no independent TCGplayer market price and is priced
// only from whichever vendor(s) actually sell it - an already-normal,
// unflagged shape for plenty of this datastore's own cards, not a new risk
// this table introduces.
//
// New entries are NOT added live by a vendor scraper mid-run - Backend.UUIDs
// is immutable after the datastore loads (mtgmatcher/backend.go), so a newly
// discovered pair only takes effect on the next datastore rebuild, once
// promoted into this table the same way idCanonicalKey/isMemorabiliaSet's own
// findings were: measured once against real vendor data, hard-coded here.
//
// Collected 2026-09-15 against a live fetch of Star City Games's bulk catalog
// export and Card Trader's blueprints/export + expansions endpoints (664 entries).
//
// Widening the known-pairing universe this way can *reduce* how many
// listings some other, name-based mechanism resolves - not a bug, a
// correctness improvement: a generic token name (Bird, Wizard, Zombie, ...)
// recurs across many sets' own sheets, and byFace/byBothNames (see
// tokenPairIndices) already refuse rather than guess whenever more than one
// real pairing shares a normalized name key. Before this table existed,
// some of those collisions were invisible - only one of the several real
// candidates was known at all, so nothing looked ambiguous. Measured on
// Card Trader's real catalog: adding this table's entries revealed 198
// previously-silent collisions this way (each independently confirmed to
// have more than one genuinely distinct real candidate, not a false
// positive), while adding 116 new resolutions of its own - a net drop in
// raw resolved count alongside a net gain in verified-safe ones, exactly
// the tradeoff "don't know, refuse" has always made in this file.
var verifiedNoUpstreamPairs = []struct{ a, b string }{
	{"003cfec6-d5a5-5426-8943-9991e8c8195f", "29d32a74-4bc4-58c4-8441-257804031d98"}, // Thopter // Satyr
	{"003cfec6-d5a5-5426-8943-9991e8c8195f", "edc00932-5ffc-5f52-8da6-d5298d564464"}, // Thopter // Cat
	{"00a3b3ce-e6be-5a96-becf-0357fa089f7c", "05306b3c-7628-5aa0-9206-badfea6bd69c"}, // Germ // Spirit
	{"00a3b3ce-e6be-5a96-becf-0357fa089f7c", "3594b34d-7f31-56f1-8a20-47d2a27b4e0f"}, // Germ // Thopter
	{"01a47bde-4e4b-5439-abd8-fc553f88fe66", "596bc1a1-147e-514e-ba92-0786ac7f334e"}, // Angel // Phyrexian
	{"01a47bde-4e4b-5439-abd8-fc553f88fe66", "61621bc7-618c-5ec0-94f6-2f6fb504a640"}, // Angel // Saproling
	{"01a47bde-4e4b-5439-abd8-fc553f88fe66", "e1fe63ec-ea55-5052-a9ea-5517fa3c8ef6"}, // Angel // Bird
	{"01c5a18f-07e6-593c-bc90-0683505ea03f", "763ca9d9-42ca-5695-b6f6-d39d9a1c4abc"}, // Mutagen // Rat
	{"01c5a18f-07e6-593c-bc90-0683505ea03f", "aa205eb4-f99d-56da-9136-4865d87c03ff"}, // Mutagen // Insect Warrior
	{"01c5a18f-07e6-593c-bc90-0683505ea03f", "c4d111a1-ea24-5c7c-b7cc-e0b542aa92cc"}, // Mutagen // Dinosaur Soldier
	{"01c5a18f-07e6-593c-bc90-0683505ea03f", "ce40eaba-4a56-5e24-bdfe-2c1e963a6afe"}, // Mutagen // Ninja Turtle Spirit
	{"02235d38-af30-57e9-82d6-6e335a378023", "9b02e27b-70be-56ba-a683-9e2a622ec8f5"}, // Treasure // Copy
	{"02235d38-af30-57e9-82d6-6e335a378023", "b89a4851-8648-572f-b78e-f55c00c1f6f8"}, // Treasure // Dog
	{"02f6f528-e3f3-596d-aac6-17589d10f095", "15f25eca-ba72-5b13-ab15-1e6bf04d5dfc"}, // Saproling // Zombie
	{"02f6f528-e3f3-596d-aac6-17589d10f095", "cf9ffa43-3ce7-59ac-9f09-ac718e93a505"}, // Saproling // Pirate
	{"02f6f528-e3f3-596d-aac6-17589d10f095", "d27f5b6a-7f5d-55d7-81e6-6192087ed01c"}, // Saproling // Soldier
	{"03c72458-0406-598c-953f-8f204503744c", "16b5f074-63da-51a6-bc86-5da72774d0ab"}, // Soldier // Clue
	{"03c72458-0406-598c-953f-8f204503744c", "33138258-8135-5875-ba96-57a1cebaf1f0"}, // Soldier // Clue
	{"03c72458-0406-598c-953f-8f204503744c", "3a44cd3e-fe02-5af3-b1e6-281027660e31"}, // Soldier // Treasure
	{"03c72458-0406-598c-953f-8f204503744c", "4f5f8c15-0464-5138-a9b6-07a65429f4f1"}, // Soldier // Treasure
	{"03c72458-0406-598c-953f-8f204503744c", "64f5b6ae-4946-576c-bdf9-545659758984"}, // Soldier // Alien Warrior
	{"03c72458-0406-598c-953f-8f204503744c", "915356b7-202a-5619-b003-e613d245d5bb"}, // Soldier // Clue
	{"03c72458-0406-598c-953f-8f204503744c", "a0b261b5-37b1-5fe4-85e5-c8f2bd31be21"}, // Soldier // Alien Salamander
	{"03c72458-0406-598c-953f-8f204503744c", "b9848bcd-fba3-5290-bfcc-15b590061511"}, // Soldier // Food
	{"03c72458-0406-598c-953f-8f204503744c", "e5a3f019-74a1-5996-91b2-3e8750cf0841"}, // Soldier // Food
	{"03c72458-0406-598c-953f-8f204503744c", "fbe51736-4f09-5854-ab11-d7e15b141a67"}, // Soldier // Food
	{"05306b3c-7628-5aa0-9206-badfea6bd69c", "79df8131-1c78-5b5b-94c8-11fae26589c9"}, // Spirit // Bird
	{"056111f7-430a-5c64-9d9d-51fd4c5a0957", "974e8ef7-5b3c-5f02-9887-027e2578687f"}, // Zombie Army // Zombie
	{"05d59711-8773-5466-b66a-902cf865cbf1", "596bc1a1-147e-514e-ba92-0786ac7f334e"}, // Jaya, Fiery Negotiator Emblem // Phyrexian
	{"05d59711-8773-5466-b66a-902cf865cbf1", "61621bc7-618c-5ec0-94f6-2f6fb504a640"}, // Jaya, Fiery Negotiator Emblem // Saproling
	{"05d59711-8773-5466-b66a-902cf865cbf1", "e1fe63ec-ea55-5052-a9ea-5517fa3c8ef6"}, // Jaya, Fiery Negotiator Emblem // Bird
	{"0650b215-3a3d-5d23-97d4-6e2256fd6840", "2c0b9a6a-58c1-5bd1-8540-f0fb374c2288"}, // Treasure // Clown Robot
	{"0650b215-3a3d-5d23-97d4-6e2256fd6840", "2f870f14-6517-5fcc-8656-66db0fedf25e"}, // Treasure // Squirrel
	{"0650b215-3a3d-5d23-97d4-6e2256fd6840", "6b70f468-ac73-5e14-a0ba-c5388b439dff"}, // Treasure // Cat
	{"0650b215-3a3d-5d23-97d4-6e2256fd6840", "9f7c2c23-0bc1-5b68-9daf-de6355875618"}, // Treasure // Zombie Employee
	{"0650b215-3a3d-5d23-97d4-6e2256fd6840", "a51bf167-c9c0-5a66-af04-b9bc7cc36cb5"}, // Treasure // Teddy Bear
	{"0650b215-3a3d-5d23-97d4-6e2256fd6840", "fa90f964-b0d7-5382-9199-98fa2b54c7f9"}, // Treasure // Clown Robot
	{"06558f31-bce8-5504-9058-d836968bab2b", "367b2213-a86a-5fdb-90f7-8c93a74cb3c3"}, // Knight // Faerie Rogue
	{"06558f31-bce8-5504-9058-d836968bab2b", "b04a8173-24b6-577f-a150-cd7e51791c04"}, // Knight // Saproling
	{"07163036-315e-554a-bfc0-16028503d1be", "dea574c9-6edd-54a5-a094-7a84dccbacf7"}, // Gargoyle // Elf Warrior
	{"07ba385a-72ad-50a1-935c-9e1717f565b8", "1d186e0e-3028-52e1-8537-d1c52575e57c"}, // Alien Angel // Dinosaur
	{"07ba385a-72ad-50a1-935c-9e1717f565b8", "33138258-8135-5875-ba96-57a1cebaf1f0"}, // Alien Angel // Clue
	{"07ba385a-72ad-50a1-935c-9e1717f565b8", "503c446b-a8f5-5791-a42e-4dd984bed366"}, // Alien Angel // Mark of the Rani
	{"07ba385a-72ad-50a1-935c-9e1717f565b8", "52622c86-375f-5342-963b-e5cdac14f354"}, // Alien Angel // Cyberman
	{"07ba385a-72ad-50a1-935c-9e1717f565b8", "5ba0df90-4985-53bf-9118-fbd40384d716"}, // Alien Angel // Beast
	{"07ba385a-72ad-50a1-935c-9e1717f565b8", "64f5b6ae-4946-576c-bdf9-545659758984"}, // Alien Angel // Alien Warrior
	{"07ba385a-72ad-50a1-935c-9e1717f565b8", "fb19c769-de84-5f88-ae6c-fdbb80a5cf48"}, // Alien Angel // Treasure
	{"07ba385a-72ad-50a1-935c-9e1717f565b8", "feaa3126-d549-54f5-b99c-222e4b3606f9"}, // Alien Angel // Treasure
	{"08e6cdff-da21-5a62-b87d-d80eaa6703dc", "16b5f074-63da-51a6-bc86-5da72774d0ab"}, // Human // Clue
	{"08e6cdff-da21-5a62-b87d-d80eaa6703dc", "1d186e0e-3028-52e1-8537-d1c52575e57c"}, // Human // Dinosaur
	{"08e6cdff-da21-5a62-b87d-d80eaa6703dc", "33138258-8135-5875-ba96-57a1cebaf1f0"}, // Human // Clue
	{"08e6cdff-da21-5a62-b87d-d80eaa6703dc", "52622c86-375f-5342-963b-e5cdac14f354"}, // Human // Cyberman
	{"08e6cdff-da21-5a62-b87d-d80eaa6703dc", "a0b261b5-37b1-5fe4-85e5-c8f2bd31be21"}, // Human // Alien Salamander
	{"097997f7-05b7-5199-9b99-54cca98c6d2a", "a82aa998-5fbf-5125-ac16-b9d57f944875"}, // Elemental // Goblin
	{"0abeb86c-be42-5320-8192-82721ed09ff9", "49b2a8ce-fcd7-5a00-b706-d5001f70db86"}, // Rogue // Treasure
	{"0acacbab-00bf-5a8c-9cfd-bed5da4ddeb9", "b4b23663-c2a0-555c-9c1a-a6e67b7858bf"}, // Elemental // Demon
	{"0b4fa871-1396-5317-bd87-054487b0f7d4", "84e8350e-916c-5bde-9c98-f4ec1d2b1f12"}, // Myr // Treasure
	{"0c159f1d-c5e0-5ff5-9634-772cd3526dfb", "6d455809-dce4-52fe-964c-02a3bbdf7eb1"}, // Servo // Thopter
	{"0c16da93-98d7-5037-a250-30d1e5482f15", "619dbf5b-bede-500d-b138-137233fa1b49"}, // Shark // Human Soldier
	{"0c16da93-98d7-5037-a250-30d1e5482f15", "ab6a49f3-7679-59c9-873b-d56dd6b6780a"}, // Shark // Pegasus
	{"0c16da93-98d7-5037-a250-30d1e5482f15", "d640859a-29e8-5cff-a0c0-5816ae2813b1"}, // Shark // Angel Warrior
	{"0c360f56-026d-582b-9a9f-afe878c099df", "c24f7648-d86a-5c66-93c1-104ddd0f95d3"}, // Spirit // Teferi, Who Slows the Sunset Emblem
	{"0c6204e4-e3fc-51cd-9496-4b2da24e3783", "658683d1-697c-5e43-b45c-37f98fb74631"}, // Ox // Faerie Rogue
	{"0efae50b-9876-5264-bd35-1504d8e5f345", "1eedd8ec-5ea2-52c5-887c-337040311ee2"}, // Fish // Dinosaur
	{"0efae50b-9876-5264-bd35-1504d8e5f345", "40c852ca-959e-5bc7-9e71-a4b2ee1468bd"}, // Fish // Alien Salamander
	{"0efae50b-9876-5264-bd35-1504d8e5f345", "5c61ec07-6ebc-51ec-bd6c-f909ec6ebbb5"}, // Fish // Treasure
	{"0efae50b-9876-5264-bd35-1504d8e5f345", "670f1571-ba7b-58bb-847f-525055b82c6b"}, // Fish // Alien Warrior
	{"0efae50b-9876-5264-bd35-1504d8e5f345", "733c6587-68fd-5dcb-9823-933a18a13dd5"}, // Fish // Mutant
	{"0efae50b-9876-5264-bd35-1504d8e5f345", "834e89d7-fa9d-5f99-9226-e87ff3b652fc"}, // Fish // Mark of the Rani
	{"0efae50b-9876-5264-bd35-1504d8e5f345", "91c7ddb5-749c-5fcd-acbe-aa3ab3417f0b"}, // Fish // Treasure
	{"0efae50b-9876-5264-bd35-1504d8e5f345", "94866a37-6db0-5be1-9d44-fb38145a6979"}, // Fish // Food
	{"0efae50b-9876-5264-bd35-1504d8e5f345", "e3caf8d3-4c7a-5aa7-9154-e701ded8b984"}, // Fish // Clue
	{"0efae50b-9876-5264-bd35-1504d8e5f345", "f632859f-fb8c-5d4b-a02e-14c7ce32d46c"}, // Fish // Cyberman
	{"0efae50b-9876-5264-bd35-1504d8e5f345", "ff16f71d-c50b-5929-86bc-bb7467c399ae"}, // Fish // Food
	{"0fa7ebcf-46e9-5f3c-b8a0-06203060b81e", "b59f8820-344a-593d-8800-afdc93753a29"}, // Assassin // Clue
	{"0fd11f84-7a70-5926-8f19-0fabafc4c440", "1ab286c4-c46f-57b2-9941-2613eb74cc93"}, // Saproling // Elemental
	{"11346e2d-6867-5b89-868d-baad1f9ead92", "570c3c71-b13b-509d-a61e-4d3ff221a8c4"}, // Beast // Weird
	{"11346e2d-6867-5b89-868d-baad1f9ead92", "a4dd1ba2-1b49-5196-9796-cc643e718d12"}, // Beast // Demon
	{"11535942-2557-5fbb-9d19-cc263ebb9d89", "3363ed10-f27c-5516-979e-24cb9e524dc2"}, // Plant // Servo
	{"11581c55-ec27-5d91-83fe-0b7cd7d7257a", "4e0e50a2-acf1-5745-b605-8a1800110fe0"}, // Dwarf Berserker // Kaya the Inexorable Emblem
	{"11581c55-ec27-5d91-83fe-0b7cd7d7257a", "5ee687f2-f19b-59a0-b6c9-e881606998a7"}, // Dwarf Berserker // Zombie Berserker
	{"11581c55-ec27-5d91-83fe-0b7cd7d7257a", "6e8cb570-6b9b-5a4e-b1a5-dcae91a6b8ad"}, // Dwarf Berserker // Bear
	{"11581c55-ec27-5d91-83fe-0b7cd7d7257a", "dbeac9d3-edd3-574a-9bdb-c8d419f6a373"}, // Dwarf Berserker // Troll Warrior
	{"11581c55-ec27-5d91-83fe-0b7cd7d7257a", "ff07ac7d-b9fe-5fa5-8402-45ce70e5a9e3"}, // Dwarf Berserker // Dragon
	{"11a2bb16-b17e-5f8e-aef3-5e3c2cb97169", "49b2a8ce-fcd7-5a00-b706-d5001f70db86"}, // Spirit // Treasure
	{"11a2bb16-b17e-5f8e-aef3-5e3c2cb97169", "88c22910-0015-50f7-8109-53e64e7c4784"}, // Spirit // Cat
	{"11a2bb16-b17e-5f8e-aef3-5e3c2cb97169", "923ecdbd-0175-575b-b374-fc5fd6cd1397"}, // Spirit // Treasure
	{"11eaab19-aa24-53b1-b785-d6477b7bb899", "94c06733-8abe-555b-9015-4d4ce32c40e3"}, // Wolf // Zombie
	{"12701ec9-8bdf-585d-9209-edbca76e2122", "71e8a701-0152-5d83-9323-4552f32716e7"}, // Blue Horror // Spawn
	{"128d056e-eddd-5133-9652-b9e059c22d65", "55ca5784-6689-582a-a7e8-06a663f42fdb"}, // Dalek // Treasure
	{"128d056e-eddd-5133-9652-b9e059c22d65", "5c61ec07-6ebc-51ec-bd6c-f909ec6ebbb5"}, // Dalek // Treasure
	{"128d056e-eddd-5133-9652-b9e059c22d65", "670f1571-ba7b-58bb-847f-525055b82c6b"}, // Dalek // Alien Warrior
	{"128d056e-eddd-5133-9652-b9e059c22d65", "733c6587-68fd-5dcb-9823-933a18a13dd5"}, // Dalek // Mutant
	{"128d056e-eddd-5133-9652-b9e059c22d65", "e6488883-6375-5415-8ec0-aec7307d6feb"}, // Dalek // Clue
	{"12ea9ae7-f2ee-5e08-ba1e-68890a67cd4e", "bf235272-bd80-5c56-8fa6-d355103fdf8d"}, // Zombie // Treasure
	{"134c10aa-8ee5-5bcf-a813-169abcb94318", "4109db31-6037-50fc-a8ba-07c1a21b59f4"}, // Wolf // Clue
	{"134c10aa-8ee5-5bcf-a813-169abcb94318", "4f83983f-76b0-57e8-8231-0e19f4bd4362"}, // Wolf // Soldier
	{"134c10aa-8ee5-5bcf-a813-169abcb94318", "525e7f62-6d27-5a6c-bc83-389625308040"}, // Wolf // Ashaya, the Awoken World
	{"134c10aa-8ee5-5bcf-a813-169abcb94318", "625decf2-a1bc-53b6-b601-342906413033"}, // Wolf // Vampire
	{"134c10aa-8ee5-5bcf-a813-169abcb94318", "a0aeb1e1-284f-51b4-bc3a-44cb3d26c067"}, // Wolf // Wrenn and Seven Emblem
	{"1381dd95-3d82-5215-887b-4c7178baa411", "17aa22b9-8fe3-518a-89fa-5fd2def16e30"}, // Zombie Army // Food
	{"1381dd95-3d82-5215-887b-4c7178baa411", "68325c67-cbb6-5db7-aedd-7248ad6e408e"}, // Zombie Army // Treasure
	{"1381dd95-3d82-5215-887b-4c7178baa411", "af87e130-3b6a-589f-959b-301f7269a7fa"}, // Zombie Army // Food
	{"1407fe64-0a46-5d5f-85d3-6ab80374df0d", "16b5f074-63da-51a6-bc86-5da72774d0ab"}, // Human Noble // Clue
	{"1407fe64-0a46-5d5f-85d3-6ab80374df0d", "1d186e0e-3028-52e1-8537-d1c52575e57c"}, // Human Noble // Dinosaur
	{"1407fe64-0a46-5d5f-85d3-6ab80374df0d", "33138258-8135-5875-ba96-57a1cebaf1f0"}, // Human Noble // Clue
	{"1407fe64-0a46-5d5f-85d3-6ab80374df0d", "3f5e883a-baaf-58ec-9f48-a8e8af76dbeb"}, // Human Noble // Mutant
	{"1407fe64-0a46-5d5f-85d3-6ab80374df0d", "503c446b-a8f5-5791-a42e-4dd984bed366"}, // Human Noble // Mark of the Rani
	{"1407fe64-0a46-5d5f-85d3-6ab80374df0d", "52622c86-375f-5342-963b-e5cdac14f354"}, // Human Noble // Cyberman
	{"1407fe64-0a46-5d5f-85d3-6ab80374df0d", "64f5b6ae-4946-576c-bdf9-545659758984"}, // Human Noble // Alien Warrior
	{"1407fe64-0a46-5d5f-85d3-6ab80374df0d", "8febb223-d127-5c7e-8584-2b9bef54942e"}, // Human Noble // Alien Insect
	{"1407fe64-0a46-5d5f-85d3-6ab80374df0d", "915356b7-202a-5619-b003-e613d245d5bb"}, // Human Noble // Clue
	{"1407fe64-0a46-5d5f-85d3-6ab80374df0d", "a0b261b5-37b1-5fe4-85e5-c8f2bd31be21"}, // Human Noble // Alien Salamander
	{"1407fe64-0a46-5d5f-85d3-6ab80374df0d", "b9848bcd-fba3-5290-bfcc-15b590061511"}, // Human Noble // Food
	{"1407fe64-0a46-5d5f-85d3-6ab80374df0d", "e5a3f019-74a1-5996-91b2-3e8750cf0841"}, // Human Noble // Food
	{"1407fe64-0a46-5d5f-85d3-6ab80374df0d", "fb19c769-de84-5f88-ae6c-fdbb80a5cf48"}, // Human Noble // Treasure
	{"1407fe64-0a46-5d5f-85d3-6ab80374df0d", "fbe51736-4f09-5854-ab11-d7e15b141a67"}, // Human Noble // Food
	{"1407fe64-0a46-5d5f-85d3-6ab80374df0d", "feaa3126-d549-54f5-b99c-222e4b3606f9"}, // Human Noble // Treasure
	{"1468602a-b4fb-5c4a-a498-d01e1c01feb4", "6bcccfbf-347f-58c6-aa82-fd740a61e8c3"}, // Angel // Cat
	{"1468602a-b4fb-5c4a-a498-d01e1c01feb4", "c7a731ba-0a37-517c-b02c-f57f37c9c118"}, // Angel // Construct
	{"147e652c-87f3-5002-b778-e6e9f1ae3176", "f7f1c18a-ecbe-5b99-ad80-2fcce898fc9c"}, // Thopter // Copy
	{"15cdbf33-de46-5228-bcdc-08295c9a6038", "6d9ac173-1396-5fa3-baeb-88b5214d5dfd"}, // Insect // Human
	{"15cdbf33-de46-5228-bcdc-08295c9a6038", "f0bc8e29-4b4f-5c21-8209-32a86ffdb49c"}, // Insect // Spirit
	{"15f25eca-ba72-5b13-ab15-1e6bf04d5dfc", "76ed2e95-f655-5b86-a404-257876628a67"}, // Zombie // Griffin
	{"16528ba1-8ff5-5249-9cc0-7aa74d1e08e8", "1bd4264f-22b6-5b3d-9ffe-05d33718dfe4"}, // Human Rogue // Treasure
	{"16528ba1-8ff5-5249-9cc0-7aa74d1e08e8", "1eedd8ec-5ea2-52c5-887c-337040311ee2"}, // Human Rogue // Dinosaur
	{"16528ba1-8ff5-5249-9cc0-7aa74d1e08e8", "40c852ca-959e-5bc7-9e71-a4b2ee1468bd"}, // Human Rogue // Alien Salamander
	{"16528ba1-8ff5-5249-9cc0-7aa74d1e08e8", "55ca5784-6689-582a-a7e8-06a663f42fdb"}, // Human Rogue // Treasure
	{"16528ba1-8ff5-5249-9cc0-7aa74d1e08e8", "5c61ec07-6ebc-51ec-bd6c-f909ec6ebbb5"}, // Human Rogue // Treasure
	{"16528ba1-8ff5-5249-9cc0-7aa74d1e08e8", "733c6587-68fd-5dcb-9823-933a18a13dd5"}, // Human Rogue // Mutant
	{"16528ba1-8ff5-5249-9cc0-7aa74d1e08e8", "834e89d7-fa9d-5f99-9226-e87ff3b652fc"}, // Human Rogue // Mark of the Rani
	{"16528ba1-8ff5-5249-9cc0-7aa74d1e08e8", "91c7ddb5-749c-5fcd-acbe-aa3ab3417f0b"}, // Human Rogue // Treasure
	{"16528ba1-8ff5-5249-9cc0-7aa74d1e08e8", "94866a37-6db0-5be1-9d44-fb38145a6979"}, // Human Rogue // Food
	{"16528ba1-8ff5-5249-9cc0-7aa74d1e08e8", "95cccd10-bb21-5afa-a2c3-8b6d68f48f24"}, // Human Rogue // Food
	{"16528ba1-8ff5-5249-9cc0-7aa74d1e08e8", "ab863556-b4e5-5b95-9cde-267027545314"}, // Human Rogue // Alien Insect
	{"16528ba1-8ff5-5249-9cc0-7aa74d1e08e8", "f632859f-fb8c-5d4b-a02e-14c7ce32d46c"}, // Human Rogue // Cyberman
	{"16528ba1-8ff5-5249-9cc0-7aa74d1e08e8", "ff16f71d-c50b-5929-86bc-bb7467c399ae"}, // Human Rogue // Food
	{"1657233e-c9e1-54ff-aa5a-6e2e2846be42", "1bd3ecd2-7048-548f-ae5a-6410354766b8"}, // Rock // Horror
	{"16b5f074-63da-51a6-bc86-5da72774d0ab", "1f5fc2a7-0ac2-5e82-b473-688241c0a5d3"}, // Clue // Human Rogue
	{"16b5f074-63da-51a6-bc86-5da72774d0ab", "553b5e02-de02-59fb-9901-28d367ab1c19"}, // Clue // Copy
	{"16b5f074-63da-51a6-bc86-5da72774d0ab", "bc5e5622-5b3f-5857-b5ef-204dd83e196b"}, // Clue // Alien Rhino
	{"16b5f074-63da-51a6-bc86-5da72774d0ab", "f50201d6-3e20-541a-89c9-b44ceaf4c0b7"}, // Clue // Dalek
	{"17aa22b9-8fe3-518a-89fa-5fd2def16e30", "1e44ddb7-8c59-5857-90f3-62ea867a0a9a"}, // Food // Elemental
	{"17aa22b9-8fe3-518a-89fa-5fd2def16e30", "20cf468e-53c0-55c9-81f0-7ed254feac63"}, // Food // Squirrel
	{"17aa22b9-8fe3-518a-89fa-5fd2def16e30", "3b93e4cb-767f-55f7-b59f-05f4c78db185"}, // Food // Construct
	{"17aa22b9-8fe3-518a-89fa-5fd2def16e30", "3d30fcdc-6e3d-5535-8bb3-34d175b2fb57"}, // Food // Golem
	{"17aa22b9-8fe3-518a-89fa-5fd2def16e30", "5fce24a2-3695-5bd7-b8a7-2c9fd21162d0"}, // Food // Golem
	{"17aa22b9-8fe3-518a-89fa-5fd2def16e30", "68c08e04-c7bc-5c2c-8653-80c3960b1b76"}, // Food // Zombie
	{"17aa22b9-8fe3-518a-89fa-5fd2def16e30", "7ad2c163-6575-5d2d-ac75-aeb04fa45a34"}, // Food // Insect
	{"17aa22b9-8fe3-518a-89fa-5fd2def16e30", "b053b0fa-87ef-5444-9e60-0946d493e14b"}, // Food // Goblin
	{"17aa22b9-8fe3-518a-89fa-5fd2def16e30", "d22b1f7f-bd5a-5691-ba2e-6999608aab5a"}, // Food // Crab
	{"18ab1a33-f901-5fb4-92b1-58985e370ea7", "ca1669d1-e134-5af3-acab-9d3be4fa71af"}, // Elf Warrior // Zombie
	{"18bca77f-d2a0-501b-a469-d8475badfa6c", "1bd4264f-22b6-5b3d-9ffe-05d33718dfe4"}, // Horse // Treasure
	{"18bca77f-d2a0-501b-a469-d8475badfa6c", "1eedd8ec-5ea2-52c5-887c-337040311ee2"}, // Horse // Dinosaur
	{"18bca77f-d2a0-501b-a469-d8475badfa6c", "4070e76d-b950-57a8-8b4c-0a6327b221eb"}, // Horse // Clue
	{"18bca77f-d2a0-501b-a469-d8475badfa6c", "40c852ca-959e-5bc7-9e71-a4b2ee1468bd"}, // Horse // Alien Salamander
	{"18bca77f-d2a0-501b-a469-d8475badfa6c", "55ca5784-6689-582a-a7e8-06a663f42fdb"}, // Horse // Treasure
	{"18bca77f-d2a0-501b-a469-d8475badfa6c", "670f1571-ba7b-58bb-847f-525055b82c6b"}, // Horse // Alien Warrior
	{"18bca77f-d2a0-501b-a469-d8475badfa6c", "733c6587-68fd-5dcb-9823-933a18a13dd5"}, // Horse // Mutant
	{"18bca77f-d2a0-501b-a469-d8475badfa6c", "834e89d7-fa9d-5f99-9226-e87ff3b652fc"}, // Horse // Mark of the Rani
	{"18bca77f-d2a0-501b-a469-d8475badfa6c", "8f74eb21-8ace-5fc6-9aa1-9eac3d061ab9"}, // Horse // Beast
	{"18bca77f-d2a0-501b-a469-d8475badfa6c", "91c7ddb5-749c-5fcd-acbe-aa3ab3417f0b"}, // Horse // Treasure
	{"18bca77f-d2a0-501b-a469-d8475badfa6c", "94866a37-6db0-5be1-9d44-fb38145a6979"}, // Horse // Food
	{"18bca77f-d2a0-501b-a469-d8475badfa6c", "ab863556-b4e5-5b95-9cde-267027545314"}, // Horse // Alien Insect
	{"18bca77f-d2a0-501b-a469-d8475badfa6c", "e3caf8d3-4c7a-5aa7-9154-e701ded8b984"}, // Horse // Clue
	{"18bca77f-d2a0-501b-a469-d8475badfa6c", "ff16f71d-c50b-5929-86bc-bb7467c399ae"}, // Horse // Food
	{"18eef80e-48c6-500e-98cf-3ed4c3601b55", "d9361551-ef54-562f-b97a-0d2500ac5588"}, // Bear // Soldier
	{"1930d0bd-5b6a-5783-8619-8ef1c74b3710", "796164df-aec2-5c1a-bdda-bb2482560963"}, // Spider // Saproling
	{"1a35533c-585f-57c6-a1f0-15a30c970419", "974e8ef7-5b3c-5f02-9887-027e2578687f"}, // Bat // Zombie
	{"1bd4264f-22b6-5b3d-9ffe-05d33718dfe4", "3459327e-38e0-52c6-a20d-8c866d674693"}, // Treasure // Soldier
	{"1bd4264f-22b6-5b3d-9ffe-05d33718dfe4", "8edcf37f-13f9-578d-9ccf-d137455f0d08"}, // Treasure // Alien Angel
	{"1bd4264f-22b6-5b3d-9ffe-05d33718dfe4", "ae65b9f8-8811-55d8-b44f-7eb7939047d9"}, // Treasure // Human Noble
	{"1bd4264f-22b6-5b3d-9ffe-05d33718dfe4", "b367754a-d8af-568a-8b47-3719214c2ea4"}, // Treasure // Alien Rhino
	{"1bd4264f-22b6-5b3d-9ffe-05d33718dfe4", "df497901-ee26-5687-8569-c236ec3346b3"}, // Treasure // Alien
	{"1d186e0e-3028-52e1-8537-d1c52575e57c", "553b5e02-de02-59fb-9901-28d367ab1c19"}, // Dinosaur // Copy
	{"1d186e0e-3028-52e1-8537-d1c52575e57c", "a1b995bd-aec0-5b7f-8db9-65b77d465558"}, // Dinosaur // Fish
	{"1d186e0e-3028-52e1-8537-d1c52575e57c", "bc5e5622-5b3f-5857-b5ef-204dd83e196b"}, // Dinosaur // Alien Rhino
	{"1d186e0e-3028-52e1-8537-d1c52575e57c", "ed01af5a-fc62-57b3-bf96-e73e9829883b"}, // Dinosaur // Horse
	{"1d4754fb-6636-50bc-b84b-273f1dc27277", "6f222117-73e0-5b15-8f8d-529bfa2cdbc1"}, // Shapeshifter Token // Dog
	{"1d4754fb-6636-50bc-b84b-273f1dc27277", "e20db866-4106-56e6-b6e6-898d17870875"}, // Shapeshifter Token // Cat Beast
	{"1d4754fb-6636-50bc-b84b-273f1dc27277", "f61410ad-88a7-5e6b-b4db-fcd243ae3ea7"}, // Shapeshifter Token // Samurai
	{"1dcfdfd8-836b-5bfd-9dd7-0b9f5809cbac", "40c852ca-959e-5bc7-9e71-a4b2ee1468bd"}, // Warrior // Alien Salamander
	{"1dcfdfd8-836b-5bfd-9dd7-0b9f5809cbac", "55ca5784-6689-582a-a7e8-06a663f42fdb"}, // Warrior // Treasure
	{"1dcfdfd8-836b-5bfd-9dd7-0b9f5809cbac", "670f1571-ba7b-58bb-847f-525055b82c6b"}, // Warrior // Alien Warrior
	{"1dcfdfd8-836b-5bfd-9dd7-0b9f5809cbac", "733c6587-68fd-5dcb-9823-933a18a13dd5"}, // Warrior // Mutant
	{"1dcfdfd8-836b-5bfd-9dd7-0b9f5809cbac", "91c7ddb5-749c-5fcd-acbe-aa3ab3417f0b"}, // Warrior // Treasure
	{"1dcfdfd8-836b-5bfd-9dd7-0b9f5809cbac", "95cccd10-bb21-5afa-a2c3-8b6d68f48f24"}, // Warrior // Food
	{"1dcfdfd8-836b-5bfd-9dd7-0b9f5809cbac", "ab863556-b4e5-5b95-9cde-267027545314"}, // Warrior // Alien Insect
	{"1dcfdfd8-836b-5bfd-9dd7-0b9f5809cbac", "e3caf8d3-4c7a-5aa7-9154-e701ded8b984"}, // Warrior // Clue
	{"1dcfdfd8-836b-5bfd-9dd7-0b9f5809cbac", "e6488883-6375-5415-8ec0-aec7307d6feb"}, // Warrior // Clue
	{"1dcfdfd8-836b-5bfd-9dd7-0b9f5809cbac", "f632859f-fb8c-5d4b-a02e-14c7ce32d46c"}, // Warrior // Cyberman
	{"1e23a3a4-6d12-5700-9f5e-0e1c0068d8a8", "b2a484db-a2ce-54a2-9e47-f83a057add32"}, // Horror // Hero
	{"1e44ddb7-8c59-5857-90f3-62ea867a0a9a", "20cf468e-53c0-55c9-81f0-7ed254feac63"}, // Elemental // Squirrel
	{"1e44ddb7-8c59-5857-90f3-62ea867a0a9a", "68325c67-cbb6-5db7-aedd-7248ad6e408e"}, // Elemental // Treasure
	{"1e44ddb7-8c59-5857-90f3-62ea867a0a9a", "74d281e4-6c2a-53c2-b0ea-253ef143d22c"}, // Elemental // Treasure
	{"1e44ddb7-8c59-5857-90f3-62ea867a0a9a", "7d9ff3f2-78bb-56dc-b1a3-6eac61cede64"}, // Elemental // Clue
	{"1e44ddb7-8c59-5857-90f3-62ea867a0a9a", "e7b91ed4-09bc-5999-9f8c-e60a09b645ee"}, // Elemental // Clue
	{"1e454e40-0232-5241-bafc-e32bbad90511", "65313366-37ce-5d33-a1f4-a4e4d0de2f52"}, // Bird // Phyrexian Germ
	{"1e59b4ed-5f6e-57f1-9e1c-06282af80df1", "6ce46d45-a445-5567-9852-53a3040d5769"}, // Spirit // Vivien Reid Emblem
	{"1e94733c-a29c-5667-b2b1-a74dd60f9836", "1ed3da1d-b508-579e-a17d-6c480c2584b5"}, // The Monarch // Squid
	{"1ee3ec14-2249-5992-b6bb-a0cc3c598164", "e887464c-161a-5e13-9b5b-1335a033042b"}, // Elf Warrior // Tibalt, Cosmic Impostor Emblem
	{"1eedd8ec-5ea2-52c5-887c-337040311ee2", "ae65b9f8-8811-55d8-b44f-7eb7939047d9"}, // Dinosaur // Human Noble
	{"1eedd8ec-5ea2-52c5-887c-337040311ee2", "b367754a-d8af-568a-8b47-3719214c2ea4"}, // Dinosaur // Alien Rhino
	{"1f4c2ec0-a94c-5ff4-88aa-28a349501426", "50fcb456-82d6-5452-8318-8a58a6ee5f0f"}, // Walker // Walker
	{"1f4c2ec0-a94c-5ff4-88aa-28a349501426", "537327a8-6f4c-529c-bdbc-21673381bd3d"}, // Walker // Treasure
	{"1f5fc2a7-0ac2-5e82-b473-688241c0a5d3", "3f5e883a-baaf-58ec-9f48-a8e8af76dbeb"}, // Human Rogue // Mutant
	{"1f5fc2a7-0ac2-5e82-b473-688241c0a5d3", "4f5f8c15-0464-5138-a9b6-07a65429f4f1"}, // Human Rogue // Treasure
	{"1f5fc2a7-0ac2-5e82-b473-688241c0a5d3", "503c446b-a8f5-5791-a42e-4dd984bed366"}, // Human Rogue // Mark of the Rani
	{"1f5fc2a7-0ac2-5e82-b473-688241c0a5d3", "5ba0df90-4985-53bf-9118-fbd40384d716"}, // Human Rogue // Beast
	{"1f5fc2a7-0ac2-5e82-b473-688241c0a5d3", "64f5b6ae-4946-576c-bdf9-545659758984"}, // Human Rogue // Alien Warrior
	{"1f5fc2a7-0ac2-5e82-b473-688241c0a5d3", "8febb223-d127-5c7e-8584-2b9bef54942e"}, // Human Rogue // Alien Insect
	{"1f5fc2a7-0ac2-5e82-b473-688241c0a5d3", "915356b7-202a-5619-b003-e613d245d5bb"}, // Human Rogue // Clue
	{"1f5fc2a7-0ac2-5e82-b473-688241c0a5d3", "a0b261b5-37b1-5fe4-85e5-c8f2bd31be21"}, // Human Rogue // Alien Salamander
	{"1f5fc2a7-0ac2-5e82-b473-688241c0a5d3", "b9848bcd-fba3-5290-bfcc-15b590061511"}, // Human Rogue // Food
	{"1f5fc2a7-0ac2-5e82-b473-688241c0a5d3", "e5a3f019-74a1-5996-91b2-3e8750cf0841"}, // Human Rogue // Food
	{"1f5fc2a7-0ac2-5e82-b473-688241c0a5d3", "fbe51736-4f09-5854-ab11-d7e15b141a67"}, // Human Rogue // Food
	{"1f5fc2a7-0ac2-5e82-b473-688241c0a5d3", "feaa3126-d549-54f5-b99c-222e4b3606f9"}, // Human Rogue // Treasure
	{"2066c068-0bfe-5cf9-bde5-b2726bd59211", "2c0b9a6a-58c1-5bd1-8540-f0fb374c2288"}, // Food // Clown Robot
	{"2066c068-0bfe-5cf9-bde5-b2726bd59211", "2f870f14-6517-5fcc-8656-66db0fedf25e"}, // Food // Squirrel
	{"2066c068-0bfe-5cf9-bde5-b2726bd59211", "6b70f468-ac73-5e14-a0ba-c5388b439dff"}, // Food // Cat
	{"2066c068-0bfe-5cf9-bde5-b2726bd59211", "9f7c2c23-0bc1-5b68-9daf-de6355875618"}, // Food // Zombie Employee
	{"2066c068-0bfe-5cf9-bde5-b2726bd59211", "a51bf167-c9c0-5a66-af04-b9bc7cc36cb5"}, // Food // Teddy Bear
	{"2066c068-0bfe-5cf9-bde5-b2726bd59211", "fa90f964-b0d7-5382-9199-98fa2b54c7f9"}, // Food // Clown Robot
	{"20871e74-cf51-5956-8184-09ff568097a5", "fc7beddd-3df3-5984-a6e3-c59fdcac7ace"}, // Salamander Warrior // Dragon
	{"20ac0df6-bb34-549c-8fa5-5d43da39e9de", "b04ea803-4861-5a3c-99be-ca87772c6fa0"}, // Angel // Cat
	{"20cf468e-53c0-55c9-81f0-7ed254feac63", "3d30fcdc-6e3d-5535-8bb3-34d175b2fb57"}, // Squirrel // Golem
	{"20cf468e-53c0-55c9-81f0-7ed254feac63", "68325c67-cbb6-5db7-aedd-7248ad6e408e"}, // Squirrel // Treasure
	{"20cf468e-53c0-55c9-81f0-7ed254feac63", "74d281e4-6c2a-53c2-b0ea-253ef143d22c"}, // Squirrel // Treasure
	{"20cf468e-53c0-55c9-81f0-7ed254feac63", "7d9ff3f2-78bb-56dc-b1a3-6eac61cede64"}, // Squirrel // Clue
	{"20cf468e-53c0-55c9-81f0-7ed254feac63", "af87e130-3b6a-589f-959b-301f7269a7fa"}, // Squirrel // Food
	{"20cf468e-53c0-55c9-81f0-7ed254feac63", "e7b91ed4-09bc-5999-9f8c-e60a09b645ee"}, // Squirrel // Clue
	{"20d1d391-656a-5c64-8b28-6d47937a1d1e", "c6009a9d-3584-5b6f-88e6-1678449bc0e6"}, // Bird // Wizard
	{"2394b923-52ac-5677-bdd1-777134dcbf1c", "26b25b5f-5dd1-51e7-b6c1-06a9c98ed581"}, // Zombie // Whale
	{"2394b923-52ac-5677-bdd1-777134dcbf1c", "833da3ec-b15a-5c13-b47d-d62904008524"}, // Zombie // Fish
	{"2394b923-52ac-5677-bdd1-777134dcbf1c", "917e32ad-b8da-5594-a889-31f656c60a87"}, // Zombie // Kraken
	{"23a9dbed-596e-5fa5-8871-f7d81d32cadc", "b5772afa-c000-5ab4-9eff-4bd7f18856fb"}, // Angel // Spirit
	{"23a9dbed-596e-5fa5-8871-f7d81d32cadc", "c33c9cb9-4a25-5c5b-85fe-08d32be1a86b"}, // Angel // Knight
	{"23ab24e7-c54e-5952-a3ab-eb823c8ccb4f", "70580d3e-737a-5612-9529-471b2e86d12c"}, // City's Blessing // Tiny
	{"23cf1fd4-4cb8-52d1-89ca-401b60ff3ec9", "4ad6b1d3-c030-52bd-a568-6134cd2ea3ed"}, // Spider // Angel
	{"23cf1fd4-4cb8-52d1-89ca-401b60ff3ec9", "5450889c-b58f-5974-955c-b5f0d88d1338"}, // Spider // Phyrexian Golem
	{"23cf1fd4-4cb8-52d1-89ca-401b60ff3ec9", "833fee32-d935-51e4-b378-8fa466620a49"}, // Spider // Spirit
	{"23cf1fd4-4cb8-52d1-89ca-401b60ff3ec9", "ba3bda5a-b0c8-5931-818b-0126c810dc7c"}, // Spider // Treasure
	{"24276f6c-70a9-50f5-83bc-818575dda480", "de807b5d-5a96-5bd8-ba32-ddcaf8ecac0c"}, // Vecna // Devil
	{"24dcbd3e-4399-5d76-b39e-036b82c718af", "ba1d8604-e466-5949-a156-abf0d3bbb33e"}, // Beast // Saproling
	{"25d3b098-e75c-5393-8f9b-d0e142c4c5ed", "3073a46e-c831-5525-b99e-f4edb3c3b627"}, // Dog Illusion // Lolth, Spider Queen Emblem
	{"25d3b098-e75c-5393-8f9b-d0e142c4c5ed", "a6f382cb-a984-502b-a750-6ebfd49c0dd2"}, // Dog Illusion // Faerie Dragon
	{"25d3b098-e75c-5393-8f9b-d0e142c4c5ed", "d3d27491-b15c-5475-9f2e-73ad7064b00e"}, // Dog Illusion // Angel
	{"25d3b098-e75c-5393-8f9b-d0e142c4c5ed", "f3e2ed5c-6482-53b9-bbbb-a519f69103dd"}, // Dog Illusion // Zombie
	{"26de75d4-7184-5138-b179-db16f1884a54", "f1bed91d-6add-5945-ac41-be69ba306baf"}, // Human Soldier // Dinosaur Beast
	{"27b9adf3-cfb3-5551-adad-757e7f00201c", "833fee32-d935-51e4-b378-8fa466620a49"}, // Elemental // Spirit
	{"27b9adf3-cfb3-5551-adad-757e7f00201c", "ba3bda5a-b0c8-5931-818b-0126c810dc7c"}, // Elemental // Treasure
	{"27b9adf3-cfb3-5551-adad-757e7f00201c", "e4440487-4f32-5ed9-b485-0e34a2258f79"}, // Elemental // Monk
	{"29d32a74-4bc4-58c4-8441-257804031d98", "a3030333-67fe-5f9b-a27a-139bb63040b6"}, // Satyr // Eldrazi
	{"2a43d4ad-81b6-5f95-b42f-db833ded0112", "7b90ef2c-5f10-5942-ba04-a77d160a36f2"}, // Copy // Robot
	{"2b1fa4b9-8203-524c-86b9-68b3cff120fe", "596bc1a1-147e-514e-ba92-0786ac7f334e"}, // Goblin // Phyrexian
	{"2b1fa4b9-8203-524c-86b9-68b3cff120fe", "e1fe63ec-ea55-5052-a9ea-5517fa3c8ef6"}, // Goblin // Bird
	{"2c0b9a6a-58c1-5bd1-8540-f0fb374c2288", "4fafd1b0-4281-54a5-b3c1-7811a0200625"}, // Clown Robot // Food
	{"2c0b9a6a-58c1-5bd1-8540-f0fb374c2288", "7daecc09-9012-5a81-86c2-3f6a831f41a7"}, // Clown Robot // Treasure
	{"2c0b9a6a-58c1-5bd1-8540-f0fb374c2288", "aeaa0582-78d4-5bbb-848d-472aa3bbb01c"}, // Clown Robot // Balloon
	{"2c5e04d9-23ef-503d-aa81-5678f40fc581", "be06fa30-58eb-5c66-a6a0-8ba4e57af9a3"}, // Gremlin // Phyrexian Germ
	{"2d5f69c8-8c6a-5ff3-95b2-93f039fa4a90", "5f8204d1-ab7a-5182-b905-725704116a7b"}, // Human Cleric // Food
	{"2d5f69c8-8c6a-5ff3-95b2-93f039fa4a90", "90b377a5-5f77-5f59-a5e0-3da610eef6c8"}, // Human Cleric // Food
	{"2e1d3bd1-27c1-54c5-b479-d914b4693c1e", "ef89b1c0-dd6f-59fd-9f80-d329f2fb5a2e"}, // Jace, Telepath Unbound Emblem // Pest
	{"2e2a09e2-7ee1-5fcb-af08-9501a0469cab", "3e05bc10-cba6-584a-8950-a8606a6f9961"}, // Elemental // Beast
	{"2e7a8472-b64e-5e06-9904-ff8e394c7072", "a2b7eba4-1589-5f59-8f49-c5c1efea9d3d"}, // Goat // Insect
	{"2f096756-31a7-5bd0-9bbc-39344ee5c222", "bf122205-2ce0-503c-b783-89249e2cc7c9"}, // Mutant // Food
	{"2f6c908a-f5d8-5f28-ac4b-5538518828e6", "ba1d8604-e466-5949-a156-abf0d3bbb33e"}, // Angel // Saproling
	{"2f870f14-6517-5fcc-8656-66db0fedf25e", "4fafd1b0-4281-54a5-b3c1-7811a0200625"}, // Squirrel // Food
	{"2f870f14-6517-5fcc-8656-66db0fedf25e", "7daecc09-9012-5a81-86c2-3f6a831f41a7"}, // Squirrel // Treasure
	{"2f870f14-6517-5fcc-8656-66db0fedf25e", "aeaa0582-78d4-5bbb-848d-472aa3bbb01c"}, // Squirrel // Balloon
	{"3073a46e-c831-5525-b99e-f4edb3c3b627", "de807b5d-5a96-5bd8-ba32-ddcaf8ecac0c"}, // Lolth, Spider Queen Emblem // Devil
	{"309b27b0-1526-57aa-9571-4115d6da29ca", "ac5267b4-033c-5424-bb73-5ce499ff175c"}, // Tyranid // Tyranid Warrior
	{"309b27b0-1526-57aa-9571-4115d6da29ca", "cb657067-0e45-5b4a-8cd8-c74055e757f5"}, // Tyranid // Tyranid
	{"309b27b0-1526-57aa-9571-4115d6da29ca", "d53b1c86-cdf3-5a6f-8ef5-5712493cb3e9"}, // Tyranid // Tyranid Gargoyle
	{"327bd47a-e2c3-515e-8d20-325a6a6ddfc5", "7f4dc7cc-dad3-572a-9185-406ca1c9d075"}, // Walker // Walker
	{"327bd47a-e2c3-515e-8d20-325a6a6ddfc5", "81a3c48b-c87b-51f3-9a71-d4bf0838290c"}, // Walker // Walker
	{"32e9dfb0-c0f4-5aee-87d9-aab57cecb719", "d3d27491-b15c-5475-9f2e-73ad7064b00e"}, // Spider // Angel
	{"32e9dfb0-c0f4-5aee-87d9-aab57cecb719", "e4f80f25-1d66-5ceb-9e51-c6d1bfd6e2ce"}, // Spider // Zariel, Archduke of Avernus Emblem
	{"33138258-8135-5875-ba96-57a1cebaf1f0", "a1b995bd-aec0-5b7f-8db9-65b77d465558"}, // Clue // Fish
	{"33138258-8135-5875-ba96-57a1cebaf1f0", "c0907e3c-e290-5768-b885-6bc88459ecdd"}, // Clue // Warrior
	{"33138258-8135-5875-ba96-57a1cebaf1f0", "ed01af5a-fc62-57b3-bf96-e73e9829883b"}, // Clue // Horse
	{"332b3d4e-183f-5f93-89bb-adfef34a51a4", "4b0b11fb-6382-5e5d-bbab-e77c1ed5cab0"}, // Human // Treasure
	{"3363ed10-f27c-5516-979e-24cb9e524dc2", "410b9b26-6c7e-59e7-a366-695781154d90"}, // Servo // Spirit
	{"3363ed10-f27c-5516-979e-24cb9e524dc2", "c8266738-dc3a-5b7e-8a53-2ef29c92d37c"}, // Servo // Rat
	{"3363ed10-f27c-5516-979e-24cb9e524dc2", "cb355e8f-f11b-5c4d-8e3d-8ef722795c85"}, // Servo // Soldier
	{"3363ed10-f27c-5516-979e-24cb9e524dc2", "faebb681-5591-50ad-977a-337a4ec0686e"}, // Servo // Whale
	{"3378a201-b957-5508-931e-f85e221d6801", "974e8ef7-5b3c-5f02-9887-027e2578687f"}, // Beast // Zombie
	{"341c37dc-2ba1-5179-b682-ec15a944c513", "e7e86bbb-a67b-562b-a317-bd7b64de0390"}, // Horror // Zombie
	{"3459327e-38e0-52c6-a20d-8c866d674693", "4070e76d-b950-57a8-8b4c-0a6327b221eb"}, // Soldier // Clue
	{"3459327e-38e0-52c6-a20d-8c866d674693", "40c852ca-959e-5bc7-9e71-a4b2ee1468bd"}, // Soldier // Alien Salamander
	{"3459327e-38e0-52c6-a20d-8c866d674693", "91c7ddb5-749c-5fcd-acbe-aa3ab3417f0b"}, // Soldier // Treasure
	{"3459327e-38e0-52c6-a20d-8c866d674693", "ab863556-b4e5-5b95-9cde-267027545314"}, // Soldier // Alien Insect
	{"3459327e-38e0-52c6-a20d-8c866d674693", "e3caf8d3-4c7a-5aa7-9154-e701ded8b984"}, // Soldier // Clue
	{"3459327e-38e0-52c6-a20d-8c866d674693", "e6488883-6375-5415-8ec0-aec7307d6feb"}, // Soldier // Clue
	{"3502f9f9-3b4c-5103-9968-9a91163f0558", "5e2d305b-ac8e-5694-8731-6f92fe1eff75"}, // Elemental // Drake
	{"3594b34d-7f31-56f1-8a20-47d2a27b4e0f", "6c0f04ce-10b5-5788-8169-c79b84a8dde9"}, // Thopter // Goat
	{"3594b34d-7f31-56f1-8a20-47d2a27b4e0f", "85e62787-3770-5070-aa29-28ff06cd8bf2"}, // Thopter // Horror
	{"367b2213-a86a-5fdb-90f7-8c93a74cb3c3", "5450889c-b58f-5974-955c-b5f0d88d1338"}, // Faerie Rogue // Phyrexian Golem
	{"367b2213-a86a-5fdb-90f7-8c93a74cb3c3", "833fee32-d935-51e4-b378-8fa466620a49"}, // Faerie Rogue // Spirit
	{"37221bba-4459-5a1a-9b22-382c2f961d06", "de807b5d-5a96-5bd8-ba32-ddcaf8ecac0c"}, // Boo // Devil
	{"380aee7c-7579-5bb5-a27b-1cfdbf88700c", "c98c00e2-87e0-5fa8-b255-b0064597c5d6"}, // Pegasus // Kor Soldier
	{"380b1a95-dd79-57f7-9f37-20c2d72ef53e", "a9151359-5f58-58f8-85af-53f0d6bb35b2"}, // Soldier // Copy
	{"3a44cd3e-fe02-5af3-b1e6-281027660e31", "a1b995bd-aec0-5b7f-8db9-65b77d465558"}, // Treasure // Fish
	{"3a44cd3e-fe02-5af3-b1e6-281027660e31", "b7e1f480-5560-5f11-8f61-3c9b1cc963fa"}, // Treasure // Alien
	{"3a44cd3e-fe02-5af3-b1e6-281027660e31", "bc5e5622-5b3f-5857-b5ef-204dd83e196b"}, // Treasure // Alien Rhino
	{"3a44cd3e-fe02-5af3-b1e6-281027660e31", "c0907e3c-e290-5768-b885-6bc88459ecdd"}, // Treasure // Warrior
	{"3a44cd3e-fe02-5af3-b1e6-281027660e31", "ed01af5a-fc62-57b3-bf96-e73e9829883b"}, // Treasure // Horse
	{"3a49ec7d-50c8-5459-861f-07185c1142e7", "d9327567-d717-52db-aa40-71cc2a6b962b"}, // Phyrexian Goblin // Drone
	{"3b49dcde-f310-5692-a8b0-530e880f9769", "dbd2113f-ed9f-54b3-b54a-f23fb12971fd"}, // Zombie // Vampire
	{"3b93e4cb-767f-55f7-b59f-05f4c78db185", "74d281e4-6c2a-53c2-b0ea-253ef143d22c"}, // Construct // Treasure
	{"3b93e4cb-767f-55f7-b59f-05f4c78db185", "7d9ff3f2-78bb-56dc-b1a3-6eac61cede64"}, // Construct // Clue
	{"3b93e4cb-767f-55f7-b59f-05f4c78db185", "e7b91ed4-09bc-5999-9f8c-e60a09b645ee"}, // Construct // Clue
	{"3d30fcdc-6e3d-5535-8bb3-34d175b2fb57", "68325c67-cbb6-5db7-aedd-7248ad6e408e"}, // Golem // Treasure
	{"3d30fcdc-6e3d-5535-8bb3-34d175b2fb57", "7d9ff3f2-78bb-56dc-b1a3-6eac61cede64"}, // Golem // Clue
	{"3ede2133-21e1-5a02-9e74-7921242a84a1", "e1728060-2847-5553-8cbf-f343927cffbf"}, // Frog Lizard // Ooze
	{"3efe4ef0-fcfb-543b-ae0c-63570dd400a0", "71e80bfc-877e-548a-ab87-1b6ac3eddbc8"}, // Frog Lizard // Germ
	{"3f5e883a-baaf-58ec-9f48-a8e8af76dbeb", "a1b995bd-aec0-5b7f-8db9-65b77d465558"}, // Mutant // Fish
	{"3f5e883a-baaf-58ec-9f48-a8e8af76dbeb", "b7e1f480-5560-5f11-8f61-3c9b1cc963fa"}, // Mutant // Alien
	{"3f5e883a-baaf-58ec-9f48-a8e8af76dbeb", "c0907e3c-e290-5768-b885-6bc88459ecdd"}, // Mutant // Warrior
	{"3f5e883a-baaf-58ec-9f48-a8e8af76dbeb", "ed01af5a-fc62-57b3-bf96-e73e9829883b"}, // Mutant // Horse
	{"3f5e883a-baaf-58ec-9f48-a8e8af76dbeb", "f50201d6-3e20-541a-89c9-b44ceaf4c0b7"}, // Mutant // Dalek
	{"4070e76d-b950-57a8-8b4c-0a6327b221eb", "ae65b9f8-8811-55d8-b44f-7eb7939047d9"}, // Clue // Human Noble
	{"4070e76d-b950-57a8-8b4c-0a6327b221eb", "df497901-ee26-5687-8569-c236ec3346b3"}, // Clue // Alien
	{"408570d9-1f4e-52ce-8d17-4edcbedff9fa", "79df8131-1c78-5b5b-94c8-11fae26589c9"}, // Saproling // Bird
	{"40c852ca-959e-5bc7-9e71-a4b2ee1468bd", "8244ce5a-599a-573d-a0ad-5a6d96e2ddf2"}, // Alien Salamander // Copy
	{"40c852ca-959e-5bc7-9e71-a4b2ee1468bd", "8edcf37f-13f9-578d-9ccf-d137455f0d08"}, // Alien Salamander // Alien Angel
	{"40c852ca-959e-5bc7-9e71-a4b2ee1468bd", "fc618d70-fe8c-5bf4-9437-bf88e1f700af"}, // Alien Salamander // Human
	{"4109db31-6037-50fc-a8ba-07c1a21b59f4", "974e8ef7-5b3c-5f02-9887-027e2578687f"}, // Clue // Zombie
	{"4109db31-6037-50fc-a8ba-07c1a21b59f4", "cde7b674-3d01-5d81-b2b6-3e64fa42a66c"}, // Clue // Golem
	{"4109db31-6037-50fc-a8ba-07c1a21b59f4", "dcef956d-cd2a-5cda-b6a4-90d79c7ee812"}, // Clue // Wolf
	{"4109db31-6037-50fc-a8ba-07c1a21b59f4", "e06431b6-e469-59ea-ae40-070e3c7d224d"}, // Clue // Illusion
	{"417fe9f1-c83b-5b0e-a623-3727ee01878e", "b9a92833-8175-52fb-8ed9-e5c6f3b7cce4"}, // Goblin // Goat
	{"42d30cfb-64e6-5daf-bec6-fed19aa86fb2", "aad5a6cd-c4be-5a78-8702-7fa09025f049"}, // Angel // Demon
	{"4577d236-59d6-53d7-9331-15ff1a6effc6", "8ce68783-b5c6-5d6e-a6fe-dcf41cf0ab45"}, // Elemental // Satyr
	{"4603b26d-cc27-5fee-94af-ae77d0b684b7", "596bc1a1-147e-514e-ba92-0786ac7f334e"}, // Cat Warrior // Phyrexian
	{"4603b26d-cc27-5fee-94af-ae77d0b684b7", "d3eecb4c-1ec3-5770-893f-3e89c455ebe9"}, // Cat Warrior // Soldier
	{"4603b26d-cc27-5fee-94af-ae77d0b684b7", "e1fe63ec-ea55-5052-a9ea-5517fa3c8ef6"}, // Cat Warrior // Bird
	{"4619d1ad-f2df-56e6-b96e-c8e204239231", "50da239a-24c1-58b6-a8fd-28fc4d4bce23"}, // Shapeshifter Token // Spirit
	{"4619d1ad-f2df-56e6-b96e-c8e204239231", "89925dc1-39e0-5818-bd29-1286eeb80a97"}, // Shapeshifter Token // Rat
	{"4667d9dc-90d5-5b6c-b112-fbd4d7511b96", "74d281e4-6c2a-53c2-b0ea-253ef143d22c"}, // Insect // Treasure
	{"4a62e0c4-bf54-5c81-9dcb-24ac27b2b7b2", "a3030333-67fe-5f9b-a27a-139bb63040b6"}, // Bird Illusion // Eldrazi
	{"4ad6b1d3-c030-52bd-a568-6134cd2ea3ed", "d1b963a7-1ccd-5e24-91d9-fe750f54a7fd"}, // Angel // Worm
	{"4b034b9d-3714-5949-a93f-bf983f74e09d", "95baeb63-4d61-5db9-bcce-cb44f1ca001f"}, // Pentavite // Myr
	{"4c454188-bb99-56c9-8ef8-54c6b0a13053", "9100049f-4585-5c92-8a35-98ae8a4effb8"}, // Angel // Warrior
	{"4d2ba458-569a-5a74-b079-cd2b89ec3f47", "9bf154c7-6242-5c33-9ff7-54eecc0771bb"}, // Copy // Clue
	{"4e0e50a2-acf1-5745-b605-8a1800110fe0", "ba573e3a-81c4-55a0-97c6-2aa6f0c7269e"}, // Kaya the Inexorable Emblem // Elf Warrior
	{"4f5f8c15-0464-5138-a9b6-07a65429f4f1", "553b5e02-de02-59fb-9901-28d367ab1c19"}, // Treasure // Copy
	{"4f5f8c15-0464-5138-a9b6-07a65429f4f1", "a1b995bd-aec0-5b7f-8db9-65b77d465558"}, // Treasure // Fish
	{"4f5f8c15-0464-5138-a9b6-07a65429f4f1", "bc5e5622-5b3f-5857-b5ef-204dd83e196b"}, // Treasure // Alien Rhino
	{"4f5f8c15-0464-5138-a9b6-07a65429f4f1", "c0907e3c-e290-5768-b885-6bc88459ecdd"}, // Treasure // Warrior
	{"4f5f8c15-0464-5138-a9b6-07a65429f4f1", "ed01af5a-fc62-57b3-bf96-e73e9829883b"}, // Treasure // Horse
	{"4f83983f-76b0-57e8-8231-0e19f4bd4362", "80ba4741-7769-5d02-8088-a30b8b8a6474"}, // Soldier // Angel
	{"4f83983f-76b0-57e8-8231-0e19f4bd4362", "d3d27491-b15c-5475-9f2e-73ad7064b00e"}, // Soldier // Angel
	{"4f90474a-d3c2-5f4d-95e8-94b1715d825a", "8c2f12fe-e7d0-5b3a-a644-ecaa45df5f9c"}, // Ogre // Bird
	{"4f90474a-d3c2-5f4d-95e8-94b1715d825a", "c398444a-a302-550f-8b8a-16f964fe1ca5"}, // Ogre // Beast
	{"4fafd1b0-4281-54a5-b3c1-7811a0200625", "6b70f468-ac73-5e14-a0ba-c5388b439dff"}, // Food // Cat
	{"4fafd1b0-4281-54a5-b3c1-7811a0200625", "9f7c2c23-0bc1-5b68-9daf-de6355875618"}, // Food // Zombie Employee
	{"4fafd1b0-4281-54a5-b3c1-7811a0200625", "a51bf167-c9c0-5a66-af04-b9bc7cc36cb5"}, // Food // Teddy Bear
	{"4fafd1b0-4281-54a5-b3c1-7811a0200625", "fa90f964-b0d7-5382-9199-98fa2b54c7f9"}, // Food // Clown Robot
	{"503c446b-a8f5-5791-a42e-4dd984bed366", "553b5e02-de02-59fb-9901-28d367ab1c19"}, // Mark of the Rani // Copy
	{"503c446b-a8f5-5791-a42e-4dd984bed366", "bc5e5622-5b3f-5857-b5ef-204dd83e196b"}, // Mark of the Rani // Alien Rhino
	{"503c446b-a8f5-5791-a42e-4dd984bed366", "c0907e3c-e290-5768-b885-6bc88459ecdd"}, // Mark of the Rani // Warrior
	{"503c446b-a8f5-5791-a42e-4dd984bed366", "ed01af5a-fc62-57b3-bf96-e73e9829883b"}, // Mark of the Rani // Horse
	{"50fcb456-82d6-5452-8318-8a58a6ee5f0f", "7f4dc7cc-dad3-572a-9185-406ca1c9d075"}, // Walker // Walker
	{"51112885-432d-52a4-97be-107f00bbdf70", "974e8ef7-5b3c-5f02-9887-027e2578687f"}, // Devil // Zombie
	{"51e0d2f4-5611-5adb-9795-be83c6b7d516", "6804f50f-17f5-5b4e-bac5-c5bc7fbdc337"}, // Replicated Ring // Human Warrior
	{"520736fb-7998-53cb-9bc3-40b0fd5853e7", "be177185-e59d-5a10-a0d2-11c18a1efea2"}, // Devil // Soldier
	{"52622c86-375f-5342-963b-e5cdac14f354", "ed01af5a-fc62-57b3-bf96-e73e9829883b"}, // Cyberman // Horse
	{"52626921-713d-5fbe-823b-c222119520aa", "be06fa30-58eb-5c66-a6a0-8ba4e57af9a3"}, // Insect // Phyrexian Germ
	{"52626921-713d-5fbe-823b-c222119520aa", "ddc9a92f-9fc8-5a18-ba5c-73d8e1202cbf"}, // Insect // Energy Reserve
	{"52b84e65-0f57-57be-87fa-ee254e602d1f", "65040fb5-6a51-50ea-a409-1388435e4e9f"}, // Saproling // Sliver
	{"52b84e65-0f57-57be-87fa-ee254e602d1f", "f32064f0-133f-5e83-bfb8-7f7b82c1fb0c"}, // Saproling // Ape
	{"534409fb-bbd8-5199-ba67-c6bc2d5fdc53", "a3030333-67fe-5f9b-a27a-139bb63040b6"}, // Elf Druid // Eldrazi
	{"534ff50b-4189-562a-89cb-ec8790d38620", "796164df-aec2-5c1a-bdda-bb2482560963"}, // Elephant // Saproling
	{"537327a8-6f4c-529c-bdbc-21673381bd3d", "81a3c48b-c87b-51f3-9a71-d4bf0838290c"}, // Treasure // Walker
	{"53988e4b-3d74-596f-995d-60cf887ba60b", "76ed2e95-f655-5b86-a404-257876628a67"}, // Ajani's Pridemate Token // Griffin
	{"53988e4b-3d74-596f-995d-60cf887ba60b", "a8d15e9b-675f-5c44-a604-70c49d17b8bf"}, // Ajani's Pridemate Token // Spirit
	{"53c31212-83ec-5487-b11a-e5de4593b4f6", "b99d12bc-ca42-5446-b731-e1ac531ee248"}, // Vampire // Dinosaur
	{"53c49d79-4634-5dca-80ea-af578fcdf6dc", "b4034a1c-afa9-56a2-ad50-d96ffdf03db0"}, // Copy // Land Mine
	{"53c49d79-4634-5dca-80ea-af578fcdf6dc", "f1eeabc4-d35f-513e-a146-1f76051b3ad9"}, // Copy // Fractal
	{"5450889c-b58f-5974-955c-b5f0d88d1338", "77468071-de65-572f-80bc-88e11a77392f"}, // Phyrexian Golem // Cat Dragon
	{"5450889c-b58f-5974-955c-b5f0d88d1338", "c8599875-0389-5953-be39-61a5a7b5bb6a"}, // Phyrexian Golem // Boar
	{"5450889c-b58f-5974-955c-b5f0d88d1338", "d1b963a7-1ccd-5e24-91d9-fe750f54a7fd"}, // Phyrexian Golem // Worm
	{"54dc8e6d-10b4-56f7-a712-30c9f030c149", "5f8204d1-ab7a-5182-b905-725704116a7b"}, // Human Warrior // Food
	{"54dc8e6d-10b4-56f7-a712-30c9f030c149", "8671529f-3caf-52a0-ad0c-212d999deab8"}, // Human Warrior // Food
	{"553b5e02-de02-59fb-9901-28d367ab1c19", "8febb223-d127-5c7e-8584-2b9bef54942e"}, // Copy // Alien Insect
	{"553b5e02-de02-59fb-9901-28d367ab1c19", "a0b261b5-37b1-5fe4-85e5-c8f2bd31be21"}, // Copy // Alien Salamander
	{"553b5e02-de02-59fb-9901-28d367ab1c19", "b9848bcd-fba3-5290-bfcc-15b590061511"}, // Copy // Food
	{"553b5e02-de02-59fb-9901-28d367ab1c19", "e5a3f019-74a1-5996-91b2-3e8750cf0841"}, // Copy // Food
	{"553b5e02-de02-59fb-9901-28d367ab1c19", "fb19c769-de84-5f88-ae6c-fdbb80a5cf48"}, // Copy // Treasure
	{"553b5e02-de02-59fb-9901-28d367ab1c19", "fbe51736-4f09-5854-ab11-d7e15b141a67"}, // Copy // Food
	{"553b5e02-de02-59fb-9901-28d367ab1c19", "feaa3126-d549-54f5-b99c-222e4b3606f9"}, // Copy // Treasure
	{"55ca5784-6689-582a-a7e8-06a663f42fdb", "8244ce5a-599a-573d-a0ad-5a6d96e2ddf2"}, // Treasure // Copy
	{"55ca5784-6689-582a-a7e8-06a663f42fdb", "8edcf37f-13f9-578d-9ccf-d137455f0d08"}, // Treasure // Alien Angel
	{"55ca5784-6689-582a-a7e8-06a663f42fdb", "ae65b9f8-8811-55d8-b44f-7eb7939047d9"}, // Treasure // Human Noble
	{"55ca5784-6689-582a-a7e8-06a663f42fdb", "df497901-ee26-5687-8569-c236ec3346b3"}, // Treasure // Alien
	{"560394ef-c194-52d9-9ebc-666076695acc", "e7e86bbb-a67b-562b-a317-bd7b64de0390"}, // Demon // Zombie
	{"560f5df3-44e3-5833-957b-42670f931886", "62e0a789-7895-59d2-a239-6005380c576a"}, // Elf Druid // Beast
	{"570c3c71-b13b-509d-a61e-4d3ff221a8c4", "c874bb19-716e-5350-ab05-04c3ae7ece4a"}, // Weird // Bird
	{"570c3c71-b13b-509d-a61e-4d3ff221a8c4", "d27f5b6a-7f5d-55d7-81e6-6192087ed01c"}, // Weird // Soldier
	{"57469ddb-d4b7-5e59-9df7-199251b58fc7", "cebf210a-3fd8-57f0-a1fa-f95900d45a88"}, // Blood // Goblin
	{"57469ddb-d4b7-5e59-9df7-199251b58fc7", "f0bc8e29-4b4f-5c21-8209-32a86ffdb49c"}, // Blood // Spirit
	{"57889f36-37eb-5bda-af92-8ac1cfd6a169", "596bc1a1-147e-514e-ba92-0786ac7f334e"}, // Elemental // Phyrexian
	{"57889f36-37eb-5bda-af92-8ac1cfd6a169", "e1fe63ec-ea55-5052-a9ea-5517fa3c8ef6"}, // Elemental // Bird
	{"58592a45-5fc9-5e4a-8134-4c88f4bf270a", "6e925641-f72a-56fc-a91a-af67effb8d88"}, // Dwarf Berserker // Spirit
	{"589d2a21-03bf-5371-b80d-c9b5cfe91c01", "81891071-3017-5949-9a54-97a0e80b43ef"}, // Stoneforged Blade // Germ
	{"596bc1a1-147e-514e-ba92-0786ac7f334e", "62ef67bc-de90-56b8-b9fe-8995539dd8af"}, // Phyrexian // Merfolk
	{"596bc1a1-147e-514e-ba92-0786ac7f334e", "91e766ed-eeec-5fa4-a8db-2ae498ab82c8"}, // Phyrexian // Badger
	{"596bc1a1-147e-514e-ba92-0786ac7f334e", "abbfb88b-cb12-5377-b069-0aee675281de"}, // Phyrexian // Bird
	{"596bc1a1-147e-514e-ba92-0786ac7f334e", "c2376c1a-2d69-52dc-84a0-aec8a4e9a82b"}, // Phyrexian // Knight
	{"596bc1a1-147e-514e-ba92-0786ac7f334e", "d98a1805-56ab-5ef2-b496-391468d5ff21"}, // Phyrexian // Ajani, Sleeper Agent Emblem
	{"5a1dda51-991c-5e13-b4ee-8506439adfd8", "cb812c4f-0b0c-5439-a5bf-4015822f1659"}, // Cat Dragon // Dragon
	{"5a7153f0-d17e-5c43-9180-4756c62a5fdc", "974e8ef7-5b3c-5f02-9887-027e2578687f"}, // Zombie // Zombie
	{"5ba0df90-4985-53bf-9118-fbd40384d716", "a1b995bd-aec0-5b7f-8db9-65b77d465558"}, // Beast // Fish
	{"5ba0df90-4985-53bf-9118-fbd40384d716", "b7e1f480-5560-5f11-8f61-3c9b1cc963fa"}, // Beast // Alien
	{"5ba0df90-4985-53bf-9118-fbd40384d716", "ed01af5a-fc62-57b3-bf96-e73e9829883b"}, // Beast // Horse
	{"5bb80cf6-edae-5967-8e02-a2692efe9268", "7c0e0855-217e-56a4-9a75-5409a158aec1"}, // Spirit // Drake
	{"5bb80cf6-edae-5967-8e02-a2692efe9268", "d1b963a7-1ccd-5e24-91d9-fe750f54a7fd"}, // Spirit // Worm
	{"5c61ec07-6ebc-51ec-bd6c-f909ec6ebbb5", "8244ce5a-599a-573d-a0ad-5a6d96e2ddf2"}, // Treasure // Copy
	{"5c61ec07-6ebc-51ec-bd6c-f909ec6ebbb5", "8edcf37f-13f9-578d-9ccf-d137455f0d08"}, // Treasure // Alien Angel
	{"5c61ec07-6ebc-51ec-bd6c-f909ec6ebbb5", "ae65b9f8-8811-55d8-b44f-7eb7939047d9"}, // Treasure // Human Noble
	{"5c61ec07-6ebc-51ec-bd6c-f909ec6ebbb5", "b367754a-d8af-568a-8b47-3719214c2ea4"}, // Treasure // Alien Rhino
	{"5c61ec07-6ebc-51ec-bd6c-f909ec6ebbb5", "df497901-ee26-5687-8569-c236ec3346b3"}, // Treasure // Alien
	{"5c61ec07-6ebc-51ec-bd6c-f909ec6ebbb5", "e60475cd-0dbe-5a46-b363-d2cd6b6f3f13"}, // Treasure // Human
	{"5cd0f0ce-7820-5507-b6db-8ceb32d7830e", "9b02e27b-70be-56ba-a683-9e2a622ec8f5"}, // Fish // Copy
	{"5e492088-8c67-5abf-a3f2-13d0a5871f1c", "b6b767d0-52a6-598c-997b-6876a92a5a0b"}, // Beast // Liliana, Defiant Necromancer Emblem
	{"5e95d749-81f3-5294-a405-b4b65a32eb4f", "7c0e0855-217e-56a4-9a75-5409a158aec1"}, // Vampire // Drake
	{"5e95d749-81f3-5294-a405-b4b65a32eb4f", "9db7edf6-2139-5df7-8e94-5906ee38136d"}, // Vampire // Egg
	{"5f43c683-7b4e-566b-8ded-a6f4311df06e", "c874bb19-716e-5350-ab05-04c3ae7ece4a"}, // Knight // Bird
	{"5f43c683-7b4e-566b-8ded-a6f4311df06e", "cffaaddc-a438-57e0-a721-a22f1f350763"}, // Knight // Dog
	{"5f8204d1-ab7a-5182-b905-725704116a7b", "a80b2904-822c-540c-b5f8-f1c910d05ebc"}, // Food // Human Rogue
	{"5f8204d1-ab7a-5182-b905-725704116a7b", "c60ebff2-0afe-5a7e-b011-cb3156ff8207"}, // Food // Goat
	{"600302a7-83e2-551a-81e6-1095e3c291dd", "dea574c9-6edd-54a5-a094-7a84dccbacf7"}, // Elephant // Elf Warrior
	{"60caed98-acff-565f-903e-bd854e08b427", "61621bc7-618c-5ec0-94f6-2f6fb504a640"}, // Insect // Saproling
	{"60caed98-acff-565f-903e-bd854e08b427", "e1fe63ec-ea55-5052-a9ea-5517fa3c8ef6"}, // Insect // Bird
	{"60cbcfc9-178d-510f-b449-f9533ca283f6", "796164df-aec2-5c1a-bdda-bb2482560963"}, // Snake // Saproling
	{"61621bc7-618c-5ec0-94f6-2f6fb504a640", "abbfb88b-cb12-5377-b069-0aee675281de"}, // Saproling // Bird
	{"61621bc7-618c-5ec0-94f6-2f6fb504a640", "c2376c1a-2d69-52dc-84a0-aec8a4e9a82b"}, // Saproling // Knight
	{"61621bc7-618c-5ec0-94f6-2f6fb504a640", "d98a1805-56ab-5ef2-b496-391468d5ff21"}, // Saproling // Ajani, Sleeper Agent Emblem
	{"62ef67bc-de90-56b8-b9fe-8995539dd8af", "d3eecb4c-1ec3-5770-893f-3e89c455ebe9"}, // Merfolk // Soldier
	{"62ef67bc-de90-56b8-b9fe-8995539dd8af", "e1fe63ec-ea55-5052-a9ea-5517fa3c8ef6"}, // Merfolk // Bird
	{"6382a9a1-d8c1-53e6-b1f3-e2f4f082a584", "974e8ef7-5b3c-5f02-9887-027e2578687f"}, // Ooze // Zombie
	{"63f18ec9-52eb-53f5-9a53-0e7ae8f430eb", "c33c9cb9-4a25-5c5b-85fe-08d32be1a86b"}, // Spirit // Knight
	{"63f18ec9-52eb-53f5-9a53-0e7ae8f430eb", "ed292204-266c-5e0a-b23a-0c87fef34ed6"}, // Spirit // Cat
	{"64241dfe-167b-5c4a-ba03-c0bb9a4996b6", "833fee32-d935-51e4-b378-8fa466620a49"}, // Zombie // Spirit
	{"64702a74-d644-583e-aec8-89621d98615f", "696f1119-96b4-5bad-b113-a9cf8d1146ea"}, // Plaguebearer of Nurgle // Astartes Warrior
	{"64702a74-d644-583e-aec8-89621d98615f", "71e8a701-0152-5d83-9323-4552f32716e7"}, // Plaguebearer of Nurgle // Spawn
	{"64f5b6ae-4946-576c-bdf9-545659758984", "824e4285-56ea-5d9f-9db2-48a34f3fddd6"}, // Alien Warrior // Human
	{"64f5b6ae-4946-576c-bdf9-545659758984", "a1b995bd-aec0-5b7f-8db9-65b77d465558"}, // Alien Warrior // Fish
	{"64f5b6ae-4946-576c-bdf9-545659758984", "bc5e5622-5b3f-5857-b5ef-204dd83e196b"}, // Alien Warrior // Alien Rhino
	{"64f5b6ae-4946-576c-bdf9-545659758984", "c0907e3c-e290-5768-b885-6bc88459ecdd"}, // Alien Warrior // Warrior
	{"64f5b6ae-4946-576c-bdf9-545659758984", "ed01af5a-fc62-57b3-bf96-e73e9829883b"}, // Alien Warrior // Horse
	{"658683d1-697c-5e43-b45c-37f98fb74631", "e532a629-e171-5745-a730-c6e5152d15f0"}, // Faerie Rogue // Beast
	{"658683d1-697c-5e43-b45c-37f98fb74631", "f32064f0-133f-5e83-bfb8-7f7b82c1fb0c"}, // Faerie Rogue // Ape
	{"6694c9b2-2bb0-5d4e-abab-ced0cb58da77", "78cd88b3-b790-5832-b73c-d2600bd03add"}, // Human Soldier // Spider
	{"670f1571-ba7b-58bb-847f-525055b82c6b", "8244ce5a-599a-573d-a0ad-5a6d96e2ddf2"}, // Alien Warrior // Copy
	{"670f1571-ba7b-58bb-847f-525055b82c6b", "ae65b9f8-8811-55d8-b44f-7eb7939047d9"}, // Alien Warrior // Human Noble
	{"670f1571-ba7b-58bb-847f-525055b82c6b", "df497901-ee26-5687-8569-c236ec3346b3"}, // Alien Warrior // Alien
	{"68325c67-cbb6-5db7-aedd-7248ad6e408e", "68c08e04-c7bc-5c2c-8653-80c3960b1b76"}, // Treasure // Zombie
	{"68325c67-cbb6-5db7-aedd-7248ad6e408e", "7ad2c163-6575-5d2d-ac75-aeb04fa45a34"}, // Treasure // Insect
	{"68325c67-cbb6-5db7-aedd-7248ad6e408e", "b053b0fa-87ef-5444-9e60-0946d493e14b"}, // Treasure // Goblin
	{"68325c67-cbb6-5db7-aedd-7248ad6e408e", "f4a52681-29e7-5f31-89ee-8898b454d344"}, // Treasure // Phyrexian Germ
	{"68c08e04-c7bc-5c2c-8653-80c3960b1b76", "af87e130-3b6a-589f-959b-301f7269a7fa"}, // Zombie // Food
	{"68c08e04-c7bc-5c2c-8653-80c3960b1b76", "e7b91ed4-09bc-5999-9f8c-e60a09b645ee"}, // Zombie // Clue
	{"696f1119-96b4-5bad-b113-a9cf8d1146ea", "71e8a701-0152-5d83-9323-4552f32716e7"}, // Astartes Warrior // Spawn
	{"6a613aa7-be49-5cc5-abd3-4fab87bb0065", "d3d27491-b15c-5475-9f2e-73ad7064b00e"}, // Treasure // Angel
	{"6b70f468-ac73-5e14-a0ba-c5388b439dff", "7daecc09-9012-5a81-86c2-3f6a831f41a7"}, // Cat // Treasure
	{"6b70f468-ac73-5e14-a0ba-c5388b439dff", "aeaa0582-78d4-5bbb-848d-472aa3bbb01c"}, // Cat // Balloon
	{"6b8f4014-0a30-5459-9cd7-ad8f456e4d59", "7e5dc858-2163-5de0-95cb-f0e2933a7f7f"}, // Citizen // Cat
	{"6b8f4014-0a30-5459-9cd7-ad8f456e4d59", "909fb00f-9bf6-563a-9dab-1a1ec252d97e"}, // Citizen // Cat
	{"6b8f4014-0a30-5459-9cd7-ad8f456e4d59", "fe4dffe8-dc27-5c0d-aaa3-589a15e78f55"}, // Citizen // Treasure
	{"6bcccfbf-347f-58c6-aa82-fd740a61e8c3", "cf9ffa43-3ce7-59ac-9f09-ac718e93a505"}, // Cat // Pirate
	{"6bcccfbf-347f-58c6-aa82-fd740a61e8c3", "eb1f128c-ce4e-5dff-b3d3-0df9795ffb54"}, // Cat // Shapeshifter Token
	{"6c6c515d-88ba-54d9-9bb1-54009c89eef9", "9a4cae1d-a07e-5a10-8fb3-776eefa1fc21"}, // Drake // Elemental
	{"6d9a50c3-1fbb-5ac3-9b2a-3f809e5ca705", "e9a6488c-20ca-5519-bf6e-7555fcf45db9"}, // Vraska, Golgari Queen Emblem // Human
	{"6e925641-f72a-56fc-a91a-af67effb8d88", "97cfd4de-c52d-52aa-81bb-d1586c413a54"}, // Spirit // Dragon
	{"6ffda757-579d-563a-862f-3c64eab1edb6", "ba573e3a-81c4-55a0-97c6-2aa6f0c7269e"}, // Tyvar Kell Emblem // Elf Warrior
	{"70660a05-39e9-53b5-86a5-487e15c4eca3", "a0e5dacb-c83e-5333-96bc-e1b34f710f52"}, // Insect // Necron Warrior
	{"71e80bfc-877e-548a-ab87-1b6ac3eddbc8", "94c06733-8abe-555b-9015-4d4ce32c40e3"}, // Germ // Zombie
	{"733c6587-68fd-5dcb-9823-933a18a13dd5", "ae65b9f8-8811-55d8-b44f-7eb7939047d9"}, // Mutant // Human Noble
	{"733c6587-68fd-5dcb-9823-933a18a13dd5", "df497901-ee26-5687-8569-c236ec3346b3"}, // Mutant // Alien
	{"733c6587-68fd-5dcb-9823-933a18a13dd5", "e60475cd-0dbe-5a46-b363-d2cd6b6f3f13"}, // Mutant // Human
	{"73b213cd-451a-5b18-a272-8f776c0763d6", "cf9d2b91-1654-5beb-9411-47cb73ebbb3f"}, // Angel // Golem
	{"741922f7-81d6-5ee9-b850-6d83d447552e", "f0127dde-c191-5bef-82b7-4cca8adc9585"}, // Chandra, Roaring Flame Emblem // Plant
	{"74d281e4-6c2a-53c2-b0ea-253ef143d22c", "b053b0fa-87ef-5444-9e60-0946d493e14b"}, // Treasure // Goblin
	{"74d281e4-6c2a-53c2-b0ea-253ef143d22c", "ce7c51cc-fb32-53b0-b1e7-aead756009a2"}, // Treasure // Daretti, Scrap Savant Emblem
	{"754fd11c-fb97-5a58-bf3a-fb4e7f179eb1", "cffaaddc-a438-57e0-a721-a22f1f350763"}, // Treasure // Dog
	{"754fd11c-fb97-5a58-bf3a-fb4e7f179eb1", "d27f5b6a-7f5d-55d7-81e6-6192087ed01c"}, // Treasure // Soldier
	{"76d4a480-d8bc-5786-ae16-c0c9cccc46fb", "be16489a-6054-58d9-ab8a-c7a4d7c91538"}, // Rhino Warrior // Devil
	{"76ed2e95-f655-5b86-a404-257876628a67", "9100049f-4585-5c92-8a35-98ae8a4effb8"}, // Griffin // Warrior
	{"76ed2e95-f655-5b86-a404-257876628a67", "cf9ffa43-3ce7-59ac-9f09-ac718e93a505"}, // Griffin // Pirate
	{"76ed2e95-f655-5b86-a404-257876628a67", "d27f5b6a-7f5d-55d7-81e6-6192087ed01c"}, // Griffin // Soldier
	{"796164df-aec2-5c1a-bdda-bb2482560963", "f94be7b2-a19b-55a3-8df7-093c5d384f52"}, // Saproling // Snake
	{"79df8131-1c78-5b5b-94c8-11fae26589c9", "8934c39f-95b1-56d3-8304-12d6583884e4"}, // Bird // Myr
	{"7abd7f62-5aed-5c9a-802c-47aa9d48c5a1", "b2a484db-a2ce-54a2-9e47-f83a057add32"}, // Robot Warrior // Hero
	{"7abd7f62-5aed-5c9a-802c-47aa9d48c5a1", "ea3364d5-771a-532e-8597-917f0d0ef77e"}, // Robot Warrior // Hero
	{"7ad2c163-6575-5d2d-ac75-aeb04fa45a34", "7d9ff3f2-78bb-56dc-b1a3-6eac61cede64"}, // Insect // Clue
	{"7ad2c163-6575-5d2d-ac75-aeb04fa45a34", "af87e130-3b6a-589f-959b-301f7269a7fa"}, // Insect // Food
	{"7c0e0855-217e-56a4-9a75-5409a158aec1", "833fee32-d935-51e4-b378-8fa466620a49"}, // Drake // Spirit
	{"7c0e0855-217e-56a4-9a75-5409a158aec1", "e4440487-4f32-5ed9-b485-0e34a2258f79"}, // Drake // Monk
	{"7cba0b1c-2752-57f2-bdee-e0f45123aa54", "d1b963a7-1ccd-5e24-91d9-fe750f54a7fd"}, // Eldrazi Scion // Worm
	{"7d9ff3f2-78bb-56dc-b1a3-6eac61cede64", "c4b05382-12cb-5258-8d3d-d651126026ab"}, // Clue // Bird
	{"7d9ff3f2-78bb-56dc-b1a3-6eac61cede64", "e9a6488c-20ca-5519-bf6e-7555fcf45db9"}, // Clue // Human
	{"7d9ff3f2-78bb-56dc-b1a3-6eac61cede64", "f4a52681-29e7-5f31-89ee-8898b454d344"}, // Clue // Phyrexian Germ
	{"7d9ff3f2-78bb-56dc-b1a3-6eac61cede64", "fc9da973-a626-5457-808e-f83f5ae6c6e1"}, // Clue // Treasure
	{"7daecc09-9012-5a81-86c2-3f6a831f41a7", "9f7c2c23-0bc1-5b68-9daf-de6355875618"}, // Treasure // Zombie Employee
	{"7daecc09-9012-5a81-86c2-3f6a831f41a7", "a51bf167-c9c0-5a66-af04-b9bc7cc36cb5"}, // Treasure // Teddy Bear
	{"7daecc09-9012-5a81-86c2-3f6a831f41a7", "fa90f964-b0d7-5382-9199-98fa2b54c7f9"}, // Treasure // Clown Robot
	{"7e5dc858-2163-5de0-95cb-f0e2933a7f7f", "b89a4851-8648-572f-b78e-f55c00c1f6f8"}, // Cat // Dog
	{"7fe69ac3-3746-5976-a53c-e4d06492050c", "fc9da973-a626-5457-808e-f83f5ae6c6e1"}, // Koma's Coil // Treasure
	{"802758d8-7975-5041-80eb-69f438c279dd", "ef89b1c0-dd6f-59fd-9f80-d329f2fb5a2e"}, // Lukka, Wayward Bonder Emblem // Pest
	{"80ba4741-7769-5d02-8088-a30b8b8a6474", "9a28c20d-86fe-57ac-a3c3-ab330ed62190"}, // Angel // Soldier
	{"81891071-3017-5949-9a54-97a0e80b43ef", "e7e86bbb-a67b-562b-a317-bd7b64de0390"}, // Germ // Zombie
	{"8244ce5a-599a-573d-a0ad-5a6d96e2ddf2", "ff16f71d-c50b-5929-86bc-bb7467c399ae"}, // Copy // Food
	{"824e4285-56ea-5d9f-9db2-48a34f3fddd6", "8febb223-d127-5c7e-8584-2b9bef54942e"}, // Human // Alien Insect
	{"824e4285-56ea-5d9f-9db2-48a34f3fddd6", "915356b7-202a-5619-b003-e613d245d5bb"}, // Human // Clue
	{"824e4285-56ea-5d9f-9db2-48a34f3fddd6", "a0b261b5-37b1-5fe4-85e5-c8f2bd31be21"}, // Human // Alien Salamander
	{"824e4285-56ea-5d9f-9db2-48a34f3fddd6", "fbe51736-4f09-5854-ab11-d7e15b141a67"}, // Human // Food
	{"833fee32-d935-51e4-b378-8fa466620a49", "b04a8173-24b6-577f-a150-cd7e51791c04"}, // Spirit // Saproling
	{"833fee32-d935-51e4-b378-8fa466620a49", "d1b963a7-1ccd-5e24-91d9-fe750f54a7fd"}, // Spirit // Worm
	{"834e89d7-fa9d-5f99-9226-e87ff3b652fc", "b367754a-d8af-568a-8b47-3719214c2ea4"}, // Mark of the Rani // Alien Rhino
	{"834e89d7-fa9d-5f99-9226-e87ff3b652fc", "e60475cd-0dbe-5a46-b363-d2cd6b6f3f13"}, // Mark of the Rani // Human
	{"84e8350e-916c-5bde-9c98-f4ec1d2b1f12", "bd747c18-1a47-5677-871c-9daae2341cca"}, // Treasure // Construct
	{"85006970-f095-5f8f-8888-cd32c5fa6bb0", "8ce68783-b5c6-5d6e-a6fe-dcf41cf0ab45"}, // Devil // Satyr
	{"8671529f-3caf-52a0-ad0c-212d999deab8", "a80b2904-822c-540c-b5f8-f1c910d05ebc"}, // Food // Human Rogue
	{"87d70064-55f4-5d8b-a130-f794038946b2", "fc9da973-a626-5457-808e-f83f5ae6c6e1"}, // Giant Wizard // Treasure
	{"87df5573-4b61-5eaa-94b2-efd2398be6ac", "974e8ef7-5b3c-5f02-9887-027e2578687f"}, // Zombie // Zombie
	{"8828db9d-8c5c-5d17-b2fe-a1f39510fb6c", "ca7d2b7a-6bfd-5000-b00f-b094b37fd1bc"}, // Hellion // Zombie
	{"88c22910-0015-50f7-8109-53e64e7c4784", "89925dc1-39e0-5818-bd29-1286eeb80a97"}, // Cat // Rat
	{"88c22910-0015-50f7-8109-53e64e7c4784", "9b02e27b-70be-56ba-a683-9e2a622ec8f5"}, // Cat // Copy
	{"896fde15-0956-5a5c-94a9-575552322c0b", "e1fe63ec-ea55-5052-a9ea-5517fa3c8ef6"}, // Knight // Bird
	{"89925dc1-39e0-5818-bd29-1286eeb80a97", "e20db866-4106-56e6-b6e6-898d17870875"}, // Rat // Cat Beast
	{"8ce68783-b5c6-5d6e-a6fe-dcf41cf0ab45", "c7db9aee-80eb-537f-8b1c-e643957e87b5"}, // Satyr // Goblin Construct
	{"8edcf37f-13f9-578d-9ccf-d137455f0d08", "8f74eb21-8ace-5fc6-9aa1-9eac3d061ab9"}, // Alien Angel // Beast
	{"8edcf37f-13f9-578d-9ccf-d137455f0d08", "91c7ddb5-749c-5fcd-acbe-aa3ab3417f0b"}, // Alien Angel // Treasure
	{"8edcf37f-13f9-578d-9ccf-d137455f0d08", "e6488883-6375-5415-8ec0-aec7307d6feb"}, // Alien Angel // Clue
	{"8f74eb21-8ace-5fc6-9aa1-9eac3d061ab9", "ae65b9f8-8811-55d8-b44f-7eb7939047d9"}, // Beast // Human Noble
	{"8f74eb21-8ace-5fc6-9aa1-9eac3d061ab9", "b367754a-d8af-568a-8b47-3719214c2ea4"}, // Beast // Alien Rhino
	{"8f74eb21-8ace-5fc6-9aa1-9eac3d061ab9", "df497901-ee26-5687-8569-c236ec3346b3"}, // Beast // Alien
	{"8febb223-d127-5c7e-8584-2b9bef54942e", "a1b995bd-aec0-5b7f-8db9-65b77d465558"}, // Alien Insect // Fish
	{"8febb223-d127-5c7e-8584-2b9bef54942e", "bc5e5622-5b3f-5857-b5ef-204dd83e196b"}, // Alien Insect // Alien Rhino
	{"8febb223-d127-5c7e-8584-2b9bef54942e", "c0907e3c-e290-5768-b885-6bc88459ecdd"}, // Alien Insect // Warrior
	{"8febb223-d127-5c7e-8584-2b9bef54942e", "ed01af5a-fc62-57b3-bf96-e73e9829883b"}, // Alien Insect // Horse
	{"90751f7e-123c-50a3-b7c6-f173a1fac26a", "fc9da973-a626-5457-808e-f83f5ae6c6e1"}, // Egg // Treasure
	{"909fb00f-9bf6-563a-9dab-1a1ec252d97e", "b89a4851-8648-572f-b78e-f55c00c1f6f8"}, // Cat // Dog
	{"90b377a5-5f77-5f59-a5e0-3da610eef6c8", "c60ebff2-0afe-5a7e-b011-cb3156ff8207"}, // Food // Goat
	{"9100049f-4585-5c92-8a35-98ae8a4effb8", "d8a7431e-3edd-5b15-8b6f-e62203a3ae4e"}, // Warrior // Cat Soldier
	{"9138077f-b563-55d5-a333-de099c084a2b", "95dfeb5e-37c9-5637-9711-aee284aa5702"}, // Wolf // Cat
	{"915356b7-202a-5619-b003-e613d245d5bb", "b7e1f480-5560-5f11-8f61-3c9b1cc963fa"}, // Clue // Alien
	{"915356b7-202a-5619-b003-e613d245d5bb", "c0907e3c-e290-5768-b885-6bc88459ecdd"}, // Clue // Warrior
	{"915356b7-202a-5619-b003-e613d245d5bb", "ed01af5a-fc62-57b3-bf96-e73e9829883b"}, // Clue // Horse
	{"915356b7-202a-5619-b003-e613d245d5bb", "f50201d6-3e20-541a-89c9-b44ceaf4c0b7"}, // Clue // Dalek
	{"91c7ddb5-749c-5fcd-acbe-aa3ab3417f0b", "ae65b9f8-8811-55d8-b44f-7eb7939047d9"}, // Treasure // Human Noble
	{"91c7ddb5-749c-5fcd-acbe-aa3ab3417f0b", "b367754a-d8af-568a-8b47-3719214c2ea4"}, // Treasure // Alien Rhino
	{"91e766ed-eeec-5fa4-a8db-2ae498ab82c8", "e1fe63ec-ea55-5052-a9ea-5517fa3c8ef6"}, // Badger // Bird
	{"920fc4a1-1aac-564d-bd72-e727b4586621", "d5685814-ee19-5b04-b4ef-bb83165c7c7e"}, // Spirit // Cat
	{"94866a37-6db0-5be1-9d44-fb38145a6979", "b367754a-d8af-568a-8b47-3719214c2ea4"}, // Food // Alien Rhino
	{"94866a37-6db0-5be1-9d44-fb38145a6979", "df497901-ee26-5687-8569-c236ec3346b3"}, // Food // Alien
	{"94866a37-6db0-5be1-9d44-fb38145a6979", "fc618d70-fe8c-5bf4-9437-bf88e1f700af"}, // Food // Human
	{"94c06733-8abe-555b-9015-4d4ce32c40e3", "ed292204-266c-5e0a-b23a-0c87fef34ed6"}, // Zombie // Cat
	{"95cccd10-bb21-5afa-a2c3-8b6d68f48f24", "ae65b9f8-8811-55d8-b44f-7eb7939047d9"}, // Food // Human Noble
	{"95cccd10-bb21-5afa-a2c3-8b6d68f48f24", "b367754a-d8af-568a-8b47-3719214c2ea4"}, // Food // Alien Rhino
	{"95cccd10-bb21-5afa-a2c3-8b6d68f48f24", "df497901-ee26-5687-8569-c236ec3346b3"}, // Food // Alien
	{"95cccd10-bb21-5afa-a2c3-8b6d68f48f24", "e60475cd-0dbe-5a46-b363-d2cd6b6f3f13"}, // Food // Human
	{"95dfeb5e-37c9-5637-9711-aee284aa5702", "cffaaddc-a438-57e0-a721-a22f1f350763"}, // Cat // Dog
	{"96180d9e-f0c7-5108-af2b-3c2f1e2d76eb", "be16489a-6054-58d9-ab8a-c7a4d7c91538"}, // Angel // Devil
	{"96403b1a-e970-5c47-a26f-09517ce1adbe", "af0dfc50-fb47-5ba4-a92a-6a0c4bc90cf7"}, // Soldier // Arco-Flagellant Token
	{"974e8ef7-5b3c-5f02-9887-027e2578687f", "a0aeb1e1-284f-51b4-bc3a-44cb3d26c067"}, // Zombie // Wrenn and Seven Emblem
	{"974e8ef7-5b3c-5f02-9887-027e2578687f", "b9f7553d-55f1-5a3d-a0bc-2c36967c544f"}, // Zombie // Zombie
	{"974e8ef7-5b3c-5f02-9887-027e2578687f", "c24f7648-d86a-5c66-93c1-104ddd0f95d3"}, // Zombie // Teferi, Who Slows the Sunset Emblem
	{"974e8ef7-5b3c-5f02-9887-027e2578687f", "c5487613-a4b3-5943-8405-51c497f272a3"}, // Zombie // Vampire
	{"974e8ef7-5b3c-5f02-9887-027e2578687f", "c827d437-be61-5b73-8d1a-b3576f6629a8"}, // Zombie // Elemental
	{"974e8ef7-5b3c-5f02-9887-027e2578687f", "d1a28b9a-8f82-5413-a1f1-3284bd2f3f52"}, // Zombie // Bird
	{"974e8ef7-5b3c-5f02-9887-027e2578687f", "f491d91a-8f4e-54a5-9513-5d89bad9f4ec"}, // Zombie // Treefolk
	{"974e8ef7-5b3c-5f02-9887-027e2578687f", "f553735d-b495-5c5f-b28e-464065ef8859"}, // Zombie // Insect
	{"9ab23f2e-bb74-5f2a-96d6-5fb7ce986151", "d71e6c2f-7c07-58e3-af7b-9b966f5c0403"}, // Golem // Treasure
	{"9cb97a78-9585-5641-96fd-d3e83ef9174b", "bf235272-bd80-5c56-8fa6-d355103fdf8d"}, // Myr // Treasure
	{"9f7c2c23-0bc1-5b68-9daf-de6355875618", "aeaa0582-78d4-5bbb-848d-472aa3bbb01c"}, // Zombie Employee // Balloon
	{"a031bbb1-851e-55e7-92f5-c547b6d9bc43", "de25859d-60a9-503d-b191-b72a4bd9eca8"}, // Wolf // Treefolk
	{"a0b261b5-37b1-5fe4-85e5-c8f2bd31be21", "bc5e5622-5b3f-5857-b5ef-204dd83e196b"}, // Alien Salamander // Alien Rhino
	{"a0b261b5-37b1-5fe4-85e5-c8f2bd31be21", "c0907e3c-e290-5768-b885-6bc88459ecdd"}, // Alien Salamander // Warrior
	{"a0cf69b8-6092-5687-85fa-a4402bd43e6d", "b59f8820-344a-593d-8800-afdc93753a29"}, // Kor Soldier // Clue
	{"a1b995bd-aec0-5b7f-8db9-65b77d465558", "b9848bcd-fba3-5290-bfcc-15b590061511"}, // Fish // Food
	{"a1b995bd-aec0-5b7f-8db9-65b77d465558", "fbe51736-4f09-5854-ab11-d7e15b141a67"}, // Fish // Food
	{"a1b995bd-aec0-5b7f-8db9-65b77d465558", "feaa3126-d549-54f5-b99c-222e4b3606f9"}, // Fish // Treasure
	{"a2b54b11-7486-5211-896c-48c3f0ab3c78", "e6007b70-03da-53d3-94d0-88cbb1abaacd"}, // Spirit // Soldier
	{"a32dca00-9a97-5e33-aff1-2694d04f4366", "ba1d8604-e466-5949-a156-abf0d3bbb33e"}, // Dragon // Saproling
	{"a337b75a-4f15-587a-aa05-1516d552c4d5", "c02382c4-4eab-5e53-a621-8397ac2034c0"}, // Knight // Elemental Shaman
	{"a3d5d5c7-2133-5762-a108-aa4de8c98b17", "c33c9cb9-4a25-5c5b-85fe-08d32be1a86b"}, // Gold // Knight
	{"a51bf167-c9c0-5a66-af04-b9bc7cc36cb5", "aeaa0582-78d4-5bbb-848d-472aa3bbb01c"}, // Teddy Bear // Balloon
	{"a5c96d41-cabd-57a5-b7d7-90957340d40a", "b89a4851-8648-572f-b78e-f55c00c1f6f8"}, // Treasure // Dog
	{"a8d15e9b-675f-5c44-a604-70c49d17b8bf", "d3d27491-b15c-5475-9f2e-73ad7064b00e"}, // Spirit // Angel
	{"aafbe5c9-7a49-5dff-9995-35f0d7ace2d4", "bd2ac668-91ce-54b9-bd79-d8446d40d9f1"}, // Angel // Spirit
	{"ab6a49f3-7679-59c9-873b-d56dd6b6780a", "d25ca38c-26d0-56d5-985c-2fcd0d3e3bc9"}, // Pegasus // Wall
	{"ab6a49f3-7679-59c9-873b-d56dd6b6780a", "e06431b6-e469-59ea-ae40-070e3c7d224d"}, // Pegasus // Illusion
	{"ab863556-b4e5-5b95-9cde-267027545314", "ae65b9f8-8811-55d8-b44f-7eb7939047d9"}, // Alien Insect // Human Noble
	{"ab863556-b4e5-5b95-9cde-267027545314", "b367754a-d8af-568a-8b47-3719214c2ea4"}, // Alien Insect // Alien Rhino
	{"ab863556-b4e5-5b95-9cde-267027545314", "df497901-ee26-5687-8569-c236ec3346b3"}, // Alien Insect // Alien
	{"ab863556-b4e5-5b95-9cde-267027545314", "e60475cd-0dbe-5a46-b363-d2cd6b6f3f13"}, // Alien Insect // Human
	{"abbfb88b-cb12-5377-b069-0aee675281de", "e1fe63ec-ea55-5052-a9ea-5517fa3c8ef6"}, // Bird // Bird
	{"ae65b9f8-8811-55d8-b44f-7eb7939047d9", "e3caf8d3-4c7a-5aa7-9154-e701ded8b984"}, // Human Noble // Clue
	{"ae65b9f8-8811-55d8-b44f-7eb7939047d9", "ff16f71d-c50b-5929-86bc-bb7467c399ae"}, // Human Noble // Food
	{"aeaa0582-78d4-5bbb-848d-472aa3bbb01c", "fa90f964-b0d7-5382-9199-98fa2b54c7f9"}, // Balloon // Clown Robot
	{"af87e130-3b6a-589f-959b-301f7269a7fa", "f4a52681-29e7-5f31-89ee-8898b454d344"}, // Food // Phyrexian Germ
	{"afdf1b42-a3f9-50a2-bdc1-3fdb2f1992ad", "fc7beddd-3df3-5984-a6e3-c59fdcac7ace"}, // Spirit // Dragon
	{"b04a8173-24b6-577f-a150-cd7e51791c04", "d9361551-ef54-562f-b97a-0d2500ac5588"}, // Saproling // Soldier
	{"b053b0fa-87ef-5444-9e60-0946d493e14b", "e7b91ed4-09bc-5999-9f8c-e60a09b645ee"}, // Goblin // Clue
	{"b367754a-d8af-568a-8b47-3719214c2ea4", "e3caf8d3-4c7a-5aa7-9154-e701ded8b984"}, // Alien Rhino // Clue
	{"b367754a-d8af-568a-8b47-3719214c2ea4", "ff16f71d-c50b-5929-86bc-bb7467c399ae"}, // Alien Rhino // Food
	{"b58da93a-bc8e-538a-9f15-357547dc6b53", "ba1d8604-e466-5949-a156-abf0d3bbb33e"}, // Goblin // Saproling
	{"b58da93a-bc8e-538a-9f15-357547dc6b53", "dcbed46f-3dd8-5755-b535-48f61bd5a95f"}, // Goblin // Boar
	{"b59f8820-344a-593d-8800-afdc93753a29", "f61410ad-88a7-5e6b-b4db-fcd243ae3ea7"}, // Clue // Samurai
	{"b5af056b-6809-5ca1-914d-780e9b0fbcc3", "fdca4948-3d94-581e-ac4a-665f4ffb1b00"}, // Golem // Golem
	{"b7e1f480-5560-5f11-8f61-3c9b1cc963fa", "b9848bcd-fba3-5290-bfcc-15b590061511"}, // Alien // Food
	{"b7e1f480-5560-5f11-8f61-3c9b1cc963fa", "fb19c769-de84-5f88-ae6c-fdbb80a5cf48"}, // Alien // Treasure
	{"b7e1f480-5560-5f11-8f61-3c9b1cc963fa", "fbe51736-4f09-5854-ab11-d7e15b141a67"}, // Alien // Food
	{"b880b621-4914-5e0c-a12a-c98553bc464b", "e1fe63ec-ea55-5052-a9ea-5517fa3c8ef6"}, // Dragon // Bird
	{"b8cb8d80-a458-5abb-a9c7-5a758a8d28b5", "ba1d8604-e466-5949-a156-abf0d3bbb33e"}, // Spider // Saproling
	{"b9848bcd-fba3-5290-bfcc-15b590061511", "bc5e5622-5b3f-5857-b5ef-204dd83e196b"}, // Food // Alien Rhino
	{"b9848bcd-fba3-5290-bfcc-15b590061511", "c0907e3c-e290-5768-b885-6bc88459ecdd"}, // Food // Warrior
	{"ba3bda5a-b0c8-5931-818b-0126c810dc7c", "d1b963a7-1ccd-5e24-91d9-fe750f54a7fd"}, // Treasure // Worm
	{"bc238d44-72c8-5163-9d81-f907013a613c", "e7e86bbb-a67b-562b-a317-bd7b64de0390"}, // Demon // Zombie
	{"be06fa30-58eb-5c66-a6a0-8ba4e57af9a3", "d43a8a70-5a2c-5c27-b20f-c0c8d0766750"}, // Phyrexian Germ // Insect
	{"bf122205-2ce0-503c-b783-89249e2cc7c9", "c4d111a1-ea24-5c7c-b7cc-e0b542aa92cc"}, // Food // Dinosaur Soldier
	{"bf84c66f-684d-57fe-9865-e52daa628879", "f0b41b8f-465c-581c-b8d6-f62d42417162"}, // Bird // Wizard
	{"bfdd1275-acd2-5382-b259-1e050d43015f", "ceac2db2-e42c-5a08-89a2-dfca70e74f8d"}, // Squid // Soldier
	{"c0907e3c-e290-5768-b885-6bc88459ecdd", "e5a3f019-74a1-5996-91b2-3e8750cf0841"}, // Warrior // Food
	{"c2376c1a-2d69-52dc-84a0-aec8a4e9a82b", "d3eecb4c-1ec3-5770-893f-3e89c455ebe9"}, // Knight // Soldier
	{"c260fa03-ae9a-5bda-8c06-aab126a23edd", "e9a6488c-20ca-5519-bf6e-7555fcf45db9"}, // Insect // Human
	{"c7a731ba-0a37-517c-b02c-f57f37c9c118", "c874bb19-716e-5350-ab05-04c3ae7ece4a"}, // Construct // Bird
	{"c7a731ba-0a37-517c-b02c-f57f37c9c118", "cf9ffa43-3ce7-59ac-9f09-ac718e93a505"}, // Construct // Pirate
	{"c7a731ba-0a37-517c-b02c-f57f37c9c118", "cffaaddc-a438-57e0-a721-a22f1f350763"}, // Construct // Dog
	{"ca1669d1-e134-5af3-acab-9d3be4fa71af", "e4ce562a-1e18-57a1-8778-90139091c767"}, // Zombie // Worm
	{"ca1669d1-e134-5af3-acab-9d3be4fa71af", "f331e3c2-5083-5f0a-8ba1-77718fe905a4"}, // Zombie // Goblin
	{"cb355e8f-f11b-5c4d-8e3d-8ef722795c85", "ddc9a92f-9fc8-5a18-ba5c-73d8e1202cbf"}, // Soldier // Energy Reserve
	{"cd91506e-6a64-5e5c-948e-851ed31ac543", "f7e12dcc-910d-5b82-b637-0b972542ead5"}, // Map // Treasure
	{"d1b963a7-1ccd-5e24-91d9-fe750f54a7fd", "e4440487-4f32-5ed9-b485-0e34a2258f79"}, // Worm // Monk
	{"d31558af-cbd5-5390-a950-35abcf938e4e", "f94be7b2-a19b-55a3-8df7-093c5d384f52"}, // Beast // Snake
	{"d3d27491-b15c-5475-9f2e-73ad7064b00e", "dcef956d-cd2a-5cda-b6a4-90d79c7ee812"}, // Angel // Wolf
	{"d3d27491-b15c-5475-9f2e-73ad7064b00e", "de807b5d-5a96-5bd8-ba32-ddcaf8ecac0c"}, // Angel // Devil
	{"d4928ecf-ba43-5759-b91c-d230325c6331", "f61410ad-88a7-5e6b-b4db-fcd243ae3ea7"}, // Phyrexian Mite // Samurai
	{"d98a1805-56ab-5ef2-b496-391468d5ff21", "e1fe63ec-ea55-5052-a9ea-5517fa3c8ef6"}, // Ajani, Sleeper Agent Emblem // Bird
	{"da4e6e00-e477-5fba-9f49-993015cbbdab", "eca9a6c2-1883-5ba3-9e0f-4317701db374"}, // Food // Plant
	{"dbd2113f-ed9f-54b3-b54a-f23fb12971fd", "f32064f0-133f-5e83-bfb8-7f7b82c1fb0c"}, // Vampire // Ape
	{"dcef956d-cd2a-5cda-b6a4-90d79c7ee812", "fc04e76b-ec4f-5e4f-bf47-c4e529329e32"}, // Wolf // Ellywick Tumblestrum Emblem
	{"ddc9a92f-9fc8-5a18-ba5c-73d8e1202cbf", "e9d0c3ba-e49f-509b-8227-f39558f12f3d"}, // Energy Reserve // Angel
	{"de807b5d-5a96-5bd8-ba32-ddcaf8ecac0c", "fda93a46-0633-58b7-a7e3-757852f6e47a"}, // Devil // Guenhwyvar
	{"df497901-ee26-5687-8569-c236ec3346b3", "e6488883-6375-5415-8ec0-aec7307d6feb"}, // Alien // Clue
	{"e5a3f019-74a1-5996-91b2-3e8750cf0841", "ed01af5a-fc62-57b3-bf96-e73e9829883b"}, // Food // Horse
	{"e5a3f019-74a1-5996-91b2-3e8750cf0841", "f50201d6-3e20-541a-89c9-b44ceaf4c0b7"}, // Food // Dalek
	{"e60475cd-0dbe-5a46-b363-d2cd6b6f3f13", "ff16f71d-c50b-5929-86bc-bb7467c399ae"}, // Human // Food
	{"e6488883-6375-5415-8ec0-aec7307d6feb", "fc618d70-fe8c-5bf4-9437-bf88e1f700af"}, // Clue // Human
	{"e7b91ed4-09bc-5999-9f8c-e60a09b645ee", "f4a52681-29e7-5f31-89ee-8898b454d344"}, // Clue // Phyrexian Germ
	{"ecfea11b-0a80-5178-b350-42b0bd7c175a", "ef89b1c0-dd6f-59fd-9f80-d329f2fb5a2e"}, // Food // Pest
	{"ed01af5a-fc62-57b3-bf96-e73e9829883b", "fb19c769-de84-5f88-ae6c-fdbb80a5cf48"}, // Horse // Treasure
	{"ed01af5a-fc62-57b3-bf96-e73e9829883b", "fbe51736-4f09-5854-ab11-d7e15b141a67"}, // Horse // Food
	{"ed01af5a-fc62-57b3-bf96-e73e9829883b", "feaa3126-d549-54f5-b99c-222e4b3606f9"}, // Horse // Treasure
	{"f1ced965-e127-5b18-a7ce-f929eabeccd3", "fd5fe8a4-bb00-5884-9468-86d63c58d1fc"}, // On an Adventure // Spirit
	{"f4dab64a-6da3-510f-b9d3-824abba40954", "fc7beddd-3df3-5984-a6e3-c59fdcac7ace"}, // Elf Warrior // Dragon
	{"f50201d6-3e20-541a-89c9-b44ceaf4c0b7", "fb19c769-de84-5f88-ae6c-fdbb80a5cf48"}, // Dalek // Treasure
	{"f50201d6-3e20-541a-89c9-b44ceaf4c0b7", "feaa3126-d549-54f5-b99c-222e4b3606f9"}, // Dalek // Treasure
	{"f61410ad-88a7-5e6b-b4db-fcd243ae3ea7", "f78af17a-6b08-5984-b36b-a15b4f1dccd4"}, // Samurai // Rebel
	{"fa3b70ab-ca5f-50aa-a303-fccdb78c4f1d", "ff3b889c-4b82-5527-85dd-26c35744e751"}, // Ooze // Plant
	{"fc618d70-fe8c-5bf4-9437-bf88e1f700af", "ff16f71d-c50b-5929-86bc-bb7467c399ae"}, // Human // Food
}
