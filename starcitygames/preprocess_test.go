package starcitygames

import (
	"log"
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func TestMain(m *testing.M) {
	allprintingsPath := os.Getenv("ALLPRINTINGS5_PATH")
	if allprintingsPath == "" {
		log.Fatalln("Need ALLPRINTINGS5_PATH variable set to run tests")
	}

	allPrintingsReader, err := os.Open(allprintingsPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer allPrintingsReader.Close()

	err = mtgmatcher.LoadDatastore(allPrintingsReader)
	if err != nil {
		log.Fatalln(err)
	}

	mtgmatcher.SetGlobalLogger(log.New(os.Stderr, "", 0))

	os.Exit(m.Run())
}

type SKUTest struct {
	Name string
	In   string
	Out  string
	Err  string
}

var SKUTests = []SKUTest{
	{
		In:  "SGL-MTG-FDN2-428-ENF1",
		Out: "f91b4613-1633-5428-90ca-420174ceb533",
	},
	{
		In:  "SGL-MTG-PWSB-ELD2_291-ENN1",
		Out: "728fa13b-e7e3-5193-a7a3-d18439bb7ee5",
	},
	{
		In:  "SGL-MTG-LTR4-741-ENF1",
		Out: "9d1b6dca-03ce-5c5c-b0d4-ee294bb89421",
	},
	{
		In:  "SGL-MTG-MPS3-001-ENF1",
		Out: "cb4a260d-8f98-5e81-b881-6727d5323917",
	},
	{
		Name: "Cavern of Souls",
		In:   "SGL-MTG-UMA2-32-ENF",
		Out:  "2b0cfd28-e73e-5519-8aea-608854b0ef43",
	},
	{
		In:  "SGL-MTG-WCHP-97SG_VIS_123s-ENN1",
		Out: "a6d53759-0830-5380-b7fe-a0d51858ec1b",
	},
	{
		Name: "Ponder",
		In:   "SGL-MTG-PRM-FEST_2025_002-ENF1",
		Out:  "0d239a5a-c6ad-54b5-b899-50ca15fdc6c9",
	},
	{
		In:  "SGL-MTG-PWSB-PRM_DRFT_UST_108-ENF1",
		Out: "e95a606c-0ac2-5a4f-a9e1-d3a29e518bb2",
	},
	{
		Name: "Counterspell",
		In:   "SGL-MTG-PRM-NYCC24_002-ENF1",
		Out:  "406b39be-799b-5da8-bc24-e1a8fa056680",
	},
	{
		Name: "Cloud, Midgar Mercenary",
		In:   "SGL-MTG-PRM-PT_2025_001-ENF1",
		Out:  "1830749f-55f4-52ba-8b3c-3e8d1a3805e6",
	},
	{
		Name: "Rograkh, Son of Rohgahh",
		In:   "SGL-MTG-PRM-CF_2025_002-ENF1",
		Out:  "7618d60e-3790-5e8f-bd44-3154001bf975",
	},
	{
		Name: "Ponder",
		In:   "SGL-MTG-PRM-SECRET_SLD_019-ENF1",
		Out:  "d25135dd-3a0b-5b7f-b23b-79b483324759",
	},
	{
		Name: "Ephemerate",
		In:   "SGL-MTG-PRM-SECRET_SLD_1990-ENF1",
		Out:  "5c67ed18-02b5-597a-a78f-fe7cb154b819_f",
	},
	{
		Name: "Agonizing Remorse",
		In:   "SGL-MTG-STA2-087-JAF1",
		Out:  "d50b8669-352c-58d9-8cb4-6352e1f0a5ee_e",
	},
	{
		In:  "SGL-MTG-PRM-CD_Q06_006-ENN1",
		Out: "f7bd93a8-58b7-58d0-849f-0c25dc2e56fb",
	},
	{
		Name: "Stupor",
		In:   "SGL-MTG-PRM-ARENA_6ED_158-ENF1",
		Out:  "bbe150f9-9c53-53ed-be36-ca06eb3b8975",
	},
	{
		Name: "Dragon's Rage Channeler",
		In:   "SGL-MTG-PRM-EWK_2023_002-ENF1",
		Out:  "e7fae764-7c36-5e47-8666-dec1729024b1",
	},
	{
		Name: "Maelstrom of the Spirit Dragon",
		In:   "SGL-MTG-PRM-PRE_TDM_260-ENF1",
		Out:  "017b0743-989e-51b5-99a4-3cdb49be35a2",
	},
	{
		Name: "Wicked Pact",
		In:   "SGL-MTG-PRM-MB2_ME4_102-ENN1",
		Out:  "fc0fe777-fc82-5c9b-a0e0-a6ee061dc721",
	},
	{
		Name: "Blood Frenzy",
		In:   "SGL-MTG-PRM-MB2_TMP_164-ENN1",
		Out:  "5909496f-bd57-5ab3-a340-4534e1148731",
	},
	{
		// Source set carries a letter in the PLST number (TD0-A80).
		Name: "Moment's Peace",
		In:   "SGL-MTG-PRM-MB2_TD0_080-ENN1",
		Out:  "7c714d73-1d87-5833-ab0f-903993eb5dc0",
	},
	{
		// Play-promo family: SKU year matches PW year.
		Name: "Dragonspeaker Shaman",
		In:   "SGL-MTG-PRM-CP_2025_001b-ENN1",
		Out:  "e15e8463-a8cb-59b0-a293-e46b39499730",
	},
	{
		// Play-promo family: maps to PLG, not PW, resolved by lookup.
		Name: "Wastes",
		In:   "SGL-MTG-PRM-MA_2025_001-ENN1",
		Out:  "943c8e65-ccaa-5723-9d8a-31c8ae1fa039",
	},
	{
		// Play-promo family: SKU year (2022) differs from the PW year (24).
		Name: "Chaos Warp",
		In:   "SGL-MTG-PRM-DRFT_2022_001-ENF1",
		Out:  "b2e0860b-d3fa-54c9-ad81-8f16ddb91b6c",
	},
	{
		// Play-promo family: maps to PSPL.
		Name: "Elektra, Daughter of the Hand",
		In:   "SGL-MTG-PRM-PLAY_MSH_004-ENF1",
		Out:  "8d8e876b-bba2-57a2-892b-0d06f3cd7d53",
	},
	{
		// Foreign Black Border: 4BB matches mtgjson directly.
		Name: "Mishra's Factory",
		In:   "SGL-MTG-4BB-361-KON1",
		Out:  "b1a68c47-9f23-558d-868a-74d6f4294cd5",
	},
	{
		// SCG's 3BB is mtgjson's FBB (Revised Foreign Black Border).
		Name: "Vesuvan Doppelganger",
		In:   "SGL-MTG-3BB-88-ITN1",
		Out:  "9ff79323-63f9-5e16-a95b-738f93ed16ca",
	},
	{
		// 15th Anniversary special card.
		Name: "Kamahl, Pit Fighter",
		In:   "SGL-MTG-PRM-15A_10E_214-ENF1",
		Out:  "b9b70a07-dcb1-540b-8361-1a6626b71d06",
	},
	{
		// DD3 is the third Duel Deck on The List.
		Name: "Bad Moon",
		In:   "SGL-MTG-PWSB-DD3_048-ENN1",
		Out:  "a1e38495-64f1-5612-801d-1cae15307db9",
	},
	{
		// Prerelease reprint carried in the main set (LCI) rather than PLCI.
		Name: "Growing Rites of Itlimoc // Itlimoc, Cradle of the Sun",
		In:   "SGL-MTG-PRM-PRE_LCI_188-ENF1",
		Out:  "9487b866-aa46-5666-b5b8-981dcedf8901",
	},
	{
		// Release promo referencing a real set directly.
		Name: "Deadeye Navigator",
		In:   "SGL-MTG-PRM-RLS_INR_492-ENF1",
		Out:  "f301811c-eb3d-5116-8603-738984fa4cfb",
	},
	{
		// Bundle promo.
		Name: "Chandra's Regulator",
		In:   "SGL-MTG-PRM-BUN_M20_131-ENF1",
		Out:  "d0d49b2e-886f-5bf3-9907-e044e0250bc8",
	},
	{
		// Lunar New Year -> Play-promo family.
		Name: "Emiel the Blessed",
		In:   "SGL-MTG-PRM-LNY_2026_001b-ENF1",
		Out:  "b6e747cf-59c0-5abc-8c63-0984e38bb445",
	},
	{
		// PRES -> Play-promo family (unique PW26 printing).
		Name: "Gilded Lotus",
		In:   "SGL-MTG-PRM-PRES_2026_001-ENF1",
		Out:  "c51d9a99-8884-50b9-88f0-97dfc0caaefe",
	},
	{
		// MagicFest 2019 -> PF19 on The List.
		Name: "Lightning Bolt",
		In:   "SGL-MTG-PWSB-PRM_MF_2019_001-ENN1",
		Out:  "1b602847-c240-594e-9c48-b939f5ebdadb",
	},
	{
		// Card in several Play sets -> prefer the yearly PW<year> printing.
		Name: "Serra Angel",
		In:   "SGL-MTG-PRM-WPNP_2023_001-ENF1",
		Out:  "5458f1a8-9b7a-5a48-a594-48be1eedb510",
	},
	{
		// Arena basic land: the matcher's arena handling picks the year from
		// the base set (Urza's Saga -> Arena League 1999).
		Name: "Island",
		In:   "SGL-MTG-PRM-ARENA_USG_338-ENF1",
		Out:  "206ed424-ffd7-596f-864b-487f775cc0d1",
	},
}

func TestSCGSKU(t *testing.T) {
	for _, probe := range SKUTests {
		test := probe
		t.Run(test.In, func(t *testing.T) {
			t.Parallel()
			out, err := ProcessSKU(test.Name, test.In)
			if err == nil && test.Err != "" {
				t.Errorf("FAIL: Expected error: %s", test.Err)
				return
			}
			if err != nil {
				if test.Err == "" {
					t.Errorf("FAIL: Unexpected error: %s", err.Error())
					return
				}
				if test.Err != err.Error() {
					t.Errorf("FAIL: Mismatched error: expected '%s', got '%s'", test.Err, err.Error())
					return
				}
			}
			if out.Id != test.Out {
				co, _ := mtgmatcher.GetUUID(out.Id)
				t.Errorf("FAIL %s: Expected '%s' got '%s' (%s)", test.In, test.Out, out.Id, co)
				return
			}
			t.Log("PASS:", test.In)
		})
	}
}
