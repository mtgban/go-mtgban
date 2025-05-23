package mtgmatcher

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgmatcher/mtgjson"
	"golang.org/x/exp/slices"
)

type DataStore interface {
	Load() cardBackend
}

type cardinfo struct {
	Name      string
	Printings []string
	Layout    string
}

// CardObject is an extension of mtgjson.Card, containing fields that cannot
// be easily represented in the original object.
type CardObject struct {
	Card
	Edition string
	Foil    bool
	Etched  bool
	Sealed  bool
}

// Card implements the Stringer interface
func (co CardObject) String() string {
	if co.Sealed {
		return co.Card.String()
	}
	finish := "nonfoil"
	if co.Etched {
		finish = "etched"
	} else if co.Foil {
		finish = "foil"
	}
	return fmt.Sprintf("%s|%s", co.Card, finish)
}

type alternateProps struct {
	OriginalName   string
	OriginalNumber string
	IsFlavor       bool
}

var backend cardBackend

type cardBackend struct {
	// Slice of all set codes loaded
	AllSets []string

	// Map of set code : Set
	Sets map[string]*Set

	// Map of normalized name : cardinfo
	// Only the main canonical name is stored here
	CardInfo map[string]cardinfo

	// Map of uuid : CardObject
	UUIDs map[string]CardObject

	// Slice with token names (not normalized and without any "Token" tags)
	Tokens []string

	// Slice with every uniquely normalized name
	AllNames []string
	// Slice with every unique name, as it would appear on a card
	AllCanonicalNames []string
	// Slice with every unique name, lower case
	AllLowerNames []string

	// Slice with every uniquely normalized product name
	AllSealed []string
	// Slice with every unique product name, as defined by mtgjson
	AllCanonicalSealed []string
	// Slice with every unique product name, lower case
	AllLowerSealed []string

	// Map of all normalized names to slice of uuids
	Hashes map[string][]string

	// Map of face/flavor names to set of canonical properties, such as original
	// name, and number, as well as a way to determine FlavorNames
	// Neither key nor values are normalized
	AlternateProps map[string]alternateProps

	// Slice with every uniquely normalized alternative name
	AlternateNames []string

	// Slice with every possible non-sealed uuid
	AllUUIDs []string
	// Slice with every possible sealed uuid
	AllSealedUUIDs []string

	// Scryfall UUID to MTGJSON UUID
	Scryfall map[string]string
	// TCG player Product ID to MTGJSON UUID
	Tcgplayer map[string]string

	// A list of keywords mapped to the full Commander set name
	CommanderKeywordMap map[string]string

	// A list of promo types as exported by mtgjson
	AllPromoTypes []string

	// A list of deck names of Secret Lair Commander cards
	SLDDeckNames []string
}

var logger = log.New(io.Discard, "", log.LstdFlags)

const (
	suffixFoil   = "_f"
	suffixEtched = "_e"
)

var languageCode2LanguageTag = map[string]string{
	"en":    "",
	"fr":    "French",
	"de":    "German",
	"it":    "Italian",
	"ja":    "Japanese",
	"jp":    "Japanese",
	"ko":    "Korean",
	"ru":    "Russian",
	"es":    "Spanish",
	"pt":    "Portuguese",
	"pt-bz": "Portuguese",
	"cs":    "Chinese Simplified",
	"ct":    "Chinese Traditional",
	"zs":    "Chinese Simplified",
	"zt":    "Chinese Traditional",
	"zhs":   "Chinese Simplified",
	"zht":   "Chinese Traditional",
}

var allLanguageTags = []string{
	"French",
	"German",
	"Italian",
	"Japanese",
	"Korean",
	"Russian",
	"Spanish",

	// Not languages but unique tags found in the language field
	"Brazil",
	"Simplified",
	"Traditional",

	// Languages affected by the tags above
	"Chinese",
	"Portuguese",
}

// Editions with interesting tokens
var setAllowedForTokens = []string{
	// League Tokens
	"L12",
	"L13",
	"L14",
	"L15",
	"L16",
	"L17",

	// Magic Player Rewards
	"MPR",
	"PR2",
	"P03",
	"P04",

	// FNM
	"F12",
	"F17",
	"F18",

	// FtV: Lore
	"V16",

	// Holiday
	"H17",

	// Secret lair
	"SLD",

	// Guild kits
	"GK1",
	"GK2",

	// Token sets
	"PHEL",
	"PL21",
	"PLNY",
	"WDMU",

	"10E",
	"30A",
	"A25",
	"AFR",
	"ALA",
	"ARB",
	"BFZ",
	"BNG",
	"DKA",
	"DMU",
	"DOM",
	"FRF",
	"ISD",
	"JOU",
	"M15",
	"MBS",
	"NPH",
	"NEO",
	"NEC",
	"RTR",
	"SOM",
	"SHM",
	"WAR",
	"ZEN",

	// Theros token sets
	"TBTH",
	"TDAG",
	"TFTH",

	// Funny token sets
	"SUNF",
	"UGL",
	"UNF",
	"UST",
}

var missingPELPtags = map[string]string{
	"1":  "Schwarzwald, Germany",
	"2":  "Danish Island, Scandinavia",
	"3":  "Vesuvio, Italy",
	"4":  "Scottish Highlands, United Kingdom, U.K.",
	"5":  "Ardennes Fagnes, Belgium",
	"6":  "Brocéliande, France",
	"7":  "Venezia, Italy",
	"8":  "Pyrenees, Spain",
	"9":  "Lowlands, Netherlands",
	"10": "Lake District National Park, United Kingdom, U.K.",
	"11": "Nottingham Forest, United Kingdom, U.K.",
	"12": "White Cliffs of Dover, United Kingdom, U.K.",
	"13": "Mont Blanc, France",
	"14": "Steppe Tundra, Russia",
	"15": "Camargue, France",
}

var missingPALPtags = map[string]string{
	"1":  "Japan",
	"2":  "Hong Kong",
	"3":  "Banaue Rice Terraces, Philippines",
	"4":  "Japan",
	"5":  "New Zealand",
	"6":  "China",
	"7":  "Meoto Iwa, Japan",
	"8":  "Taiwan",
	"9":  "Uluru, Australia",
	"10": "Japan",
	"11": "Korea",
	"12": "Singapore",
	"13": "Mount Fuji, Japan",
	"14": "Great Wall of China",
	"15": "Indonesia",
}

// List of playtest or unknown cards which have a similar name to actual cards
// in various editions. We change their name appending "Playtest" to treat them
// differently and tell them apart their main counterpart
var duplicatedCardNames = []string{
	"Clear, the Mind",
	"Glimpse, the Unthinkable",
	"Pick Your Poison",
	"Red Herring",
	"______",
}

// List of numbers in SLD that need to be decoupled
var sldJPNLangDupes = []string{
	// Specila Guests
	"1110", "1111", "1112", "1113", "1114", "1115", "1116", "1117",
	// Miku
	"1587", "1592", "1594", "1595", "1596", "1597", "1599", "1602", "1603", "1604", "1605", "1607",
	// Miku 2
	"1585", "1590", "1593", "1598", "1600", "807",
}

// List of numbers that need to have their number/uuid revisioned due
// to having foil and nonfoil merged in the same card object
var foilDupes = map[string][]string{
	"SLD": {
		"800", "801", "802", "804", "805", "806", "807", "808", "810",
		"828", "871", "872", "873", "886",

		"1316", "1317", "1318", "1319", "1320", "1321", "1322", "1323", "1324",
		"1478", "1479", "1480", "1481", "1482",
		"1550", "1551", "1552", "1553",

		// The first miku drop was split upstream, but like all miku drops numbers have gaps
		"1557", "1558", "1559", "1560", "1561", "1570", "1571", "1572", "1573", "1574",
		"1585", "1586", "1588", "1589", "1590", "1591", "1592", "1593", "1595",
		"1598", "1599", "1600", "1601", "1603", "1604", "1606", "1607",

		"1614", "1615", "1616", "1617", "1618",
		"1619", "1620", "1621", "1622", "1623", "1624", "1625", "1626",

		"1647", "1648", "1649", "1650", "1651", "1652", "1653", "1654", "1655", "1656",
		"1657", "1658", "1659", "1660", "1661", "1662", "1663", "1664", "1665", "1666",
		"1667", "1668", "1669", "1670", "1671",
		"1691", "1692", "1693", "1694", "1695", "1696", "1697", "1698", "1699", "1700", "1701", "1702",
		"1703", "1704", "1705", "1706", "1707",
		"1708", "1709", "1710",
		"1711", "1712", "1713", "1714", "1715", "1716", "1717", "1718", "1719", "1720",
		"1721", "1722", "1723", "1724", "1725", "1726", "1727", "1728", "1729", "1730",
		"1731", "1732", "1733", "1734", "1735", "1736", "1737", "1738", "1739", "1740",
		"1741", "1742", "1743", "1744", "1745", "1746", "1747", "1748", "1749", "1750",
		"1751", "1752",
		"1758", "1759", "1760", "1761", "1762", "1763", "1764", "1765", "1766", "1767",
		"1768", "1769", "1770", "1771", "1772", "1773", "1774", "1775", "1776", "1777",
		"1778", "1779", "1780", "1781", "1782", "1783", "1784", "1785", "1786", "1787",
		"1788", "1789", "1790", "1791", "1792", "1793", "1794", "1795", "1796", "1797",
		"1798", "1799", "1800", "1801", "1802", "1803", "1804", "1805", "1806",
		"1821", "1822", "1823", "1824",
		"1877", "1878", "1879", "1880",
		"1911", "1912", "1913", "1914", "1915",
		"1955", "1956", "1957", "1958", "1959",
		"9990", "9991", "9992", "9993",

		"895", "896",
		"1428", "1429", "1430", "1431", "1432",
		"1562", "1563", "1564", "1565",
		"1753", "1754", "1755", "1756", "1757",
		"1816", "1817", "1818", "1819", "1820",
		"1873", "1874", "1875", "1876",
		"1916", "1917", "1918", "1919", "1920", "1921", "1922", "1923", "1924", "1925",
		"1926", "1927", "1928", "1929", "1930", "1931", "1932", "1933", "1934", "1935",
		"1936", "1937", "1938", "1939", "1940", "1941", "1942", "1943",
		"1970", "1971", "1972", "1973", "1974",
		"2014", "2015", "2016", "2017", "2018",
		"1894", "1893", "1892", "1895",

		"1859", "1860", "1861", "1862", "1863", "1864", "1865",
		"1866", "1867", "1868", "1869", "1870", "1871", "1872",
		"2005", "2006", "2007", "2008", "2043", "2044", "2046",
		"2052", "2054", "2056", "2071", "2072", "2074", "2075",
	},
	"M3C": {
		"32", "33", "34", "35", "36", "37", "38", "39", "40", "41", "42", "43", "44", "45", "46", "47",
		"48", "49", "50", "51", "52", "53", "54", "55", "56", "57", "58", "59", "60", "61", "62", "63",
		"64", "65", "66", "67", "68", "69", "70", "71", "72", "73", "74", "75", "76", "77", "78", "79",
		"80", "81", "82", "83",
		"152", "153", "154", "155", "156", "157", "158", "159",
		"160", "161", "162", "163", "164", "165", "166", "167", "168", "169",
		"170", "171", "172", "173", "174", "175", "176", "177", "178", "179",
		"180", "181", "182", "183", "184", "185", "186", "187", "188", "189",
		"190", "191", "192", "193", "194", "195", "196", "197", "198", "199",
		"200", "201", "202", "203", "204", "205", "206", "207", "208", "209",
		"210", "211", "212", "213", "214", "215", "216", "217", "218", "219",
		"220", "221", "222", "223", "224", "225", "226", "227", "228", "229",
		"230", "231", "232", "233", "234", "235", "236", "237", "238", "239",
		"240", "241", "242", "243", "244", "245", "246", "247", "248", "249",
		"250", "251", "252", "253", "254", "255", "256", "257", "258", "259",
		"260", "261", "262", "263", "264", "265", "266", "267", "268", "269",
		"270", "271", "272", "273", "274", "275", "276", "277", "278", "279",
		"280", "281", "282", "283", "284", "285", "286", "287", "288", "289",
		"290", "291", "292", "293", "294", "295", "296", "297", "298", "299",
		"300", "301", "302", "303", "304", "305", "306", "307", "308", "309",
		"310", "311", "312", "313", "314", "315", "316", "317", "318", "319",
		"320", "321", "322", "323", "324", "325", "326", "327", "328", "329",
		"330", "331", "332", "333", "334", "335", "336", "337", "338", "339",
		"340", "341", "342", "343", "344", "345", "346", "347", "348", "349",
		"350", "351", "352", "353", "354", "355", "356", "357", "358", "359",
		"360", "361", "362", "363", "364", "365", "366", "367", "368", "369",
		"370", "371", "372", "373", "374", "375", "376", "377", "378", "379",
		"380", "381", "382", "383", "384", "385", "386", "387", "388", "389",
		"390", "391", "392", "393", "394", "395", "396", "397", "398", "399",
		"400", "401", "402", "403", "404", "405", "406", "407", "408", "409", "410", "411",
	},
}

var tcgIds = map[string][]string{
	"SLD": {
		"619189", "554682", "560087", "619186", "554585", "561737", "585049", "557904", "554606",
		"557959", "586048", "586050", "570915", "587713", "560455", "560458", "560461", "560463",
		"618007", "618017", "618021", "618027", "618033", "614245", "614249", "614256", "614265",
		"614275", "617958", "617961", "617965", "617967", "555820", "555824", "555827", "555831",
		"555834", "555801", "555805", "555810", "555814", "555816", "583908", "617854", "617856",
		"617860", "583910", "617866", "555440", "583912", "555442", "583914", "555444", "583916",
		"617870", "555446", "555449", "617872", "555452", "550807", "550809", "550811", "550813",
		"550815", "550756", "550758", "550760", "550763", "550769", "550773", "550775", "550777",
		"560418", "560424", "560428", "560430", "560432", "560437", "560440", "561291", "561514",
		"561517", "560266", "560283", "560298", "560301", "560303", "560305", "560308", "560357",
		"560412", "560415", "583890", "583892", "583894", "583896", "583898", "541370", "541374",
		"541379", "541380", "541383", "541385", "541387", "560442", "560444", "560446", "560448",
		"560450", "550742", "550746", "550747", "550749", "550752", "555460", "555462", "555458",
		"555464", "555466", "555682", "555686", "555680", "555688", "555684", "583900", "583902",
		"583904", "583906", "617936", "617942", "617945", "617950", "587991", "587994", "587997",
		"588001", "588003", "588007", "588010", "588013", "588015", "588020", "588025", "588053",
		"588058", "588061", "588063", "588065", "588034", "588037", "588040", "588044", "588047",
		"587977", "587979", "587981", "587983", "587985", "587987", "583879", "583880", "583881",
		"583882", "583883", "583884", "583885", "583886", "583887", "583888", "583869", "583870",
		"583871", "583872", "583873", "583874", "583875", "583876", "583877", "583878", "565154",
		"565156", "565158", "565160", "565162", "565164", "565166", "565168", "565170", "565172",
		"565174", "565176", "565178", "565180", "565182", "565184", "565186", "565189", "565191",
		"565193", "565195", "565197", "565199", "565201", "565203", "565205", "565207", "565209",
		"565211",
		"587182", "587184", "587186", "587188", // for "1821", "1822", "1823", "1824"
		"617989", "617995", "618000", "618003", "617969", "617971", "617978", "617982", "617984",
		"617874", "617876", "617878", "617880", "617882", "549465", "549473", "549476", "549478",

		"625635", "625637",
		"625061", "625063", "625065", "625067", "625068",
		"625053", "625055", "625057", "625059",
		"625926", "625930", "625941", "625943", "625945",
		"625040", "625042", "625044", "625046", "625048",
		"625084", "625086", "625088", "625090",
		"624509", "624511", "624513", "624515", "624517", "624483", "624489", "624493", "624497",
		"624502", "624693", "624692", "624691", "624689", "624690", "624688", "624687", "624694",
		"624695", "624696", "624697", "624698", "624699", "624700", "624701", "624702", "624703",
		"624704", "625071", "625073", "625075", "625077", "625079", "625092", "625094", "625096",
		"625098", "625100",
		"626516", "629657", "629658", "629656",

		"632095", "632097", "632100", "632104", "632135", "632139", "632145",
		"632152", "632157", "632106", "632108", "632118", "632123", "632126",
		"629928", "629933", "629938", "629940", "629990", "629993", "629999",
		"629968", "629978", "629984", "629903", "629905", "629909", "629911",
	},
	"M3C": {
		"553399", "553438", "553815", "553424", "553433", "553816", "553870", "553379", "553015",
		"553017", "553019", "553020", "553021", "553027", "553435", "553490", "552501", "553484",
		"553666", "553493", "552503", "553420", "552504", "553036", "553039", "552505", "553043",
		"553814", "553810", "552507", "553812", "553427", "553403", "553702", "553651", "553414",
		"553711", "553415", "552509", "553784", "553782", "553779", "553776", "553047", "553054",
		"553061", "553564", "553571", "552518", "553591", "553592", "553597",
		"553393", "555124", "553445", "553871", "553370", "553872", "552521", "553461", "553402",
		"553462", "553465", "553003", "553004", "553005", "553006", "553013", "552499", "553374",
		"553014", "553016", "553022", "553024", "553025", "553026", "553491", "553028", "553029",
		"553425", "553394", "553488", "552500", "553030", "553487", "553431", "553486", "553032",
		"553363", "553034", "553485", "553353", "553437", "553418", "553035", "553443", "553395",
		"552502", "553670", "553672", "553676", "553679", "553683", "553460", "553689", "553384",
		"553691", "553693", "553701", "553706", "553037", "553712", "553040", "553041", "553385",
		"553042", "553044", "553726", "553456", "553390", "552508", "553447", "553409", "553873",
		"553694", "553874", "553440", "553696", "553875", "553423", "553697", "553369", "553876",
		"553365", "553698", "553700", "553391", "553656", "553407", "553654", "553704", "553650",
		"553411", "553707", "553389", "553649", "553709", "553417", "553648", "553646", "553428",
		"553645", "553383", "553392", "553714", "553804", "553801", "553046", "553797", "553795",
		"553877", "553794", "553793", "553792", "553791", "553455", "553362", "553790", "553789",
		"553788", "553458", "553787", "553879", "553410", "553786", "553454", "553381", "553880",
		"553048", "553785", "553049", "553050", "553770", "553051", "553052", "553367", "553053",
		"553055", "553056", "553478", "553477", "553406", "553413", "553419", "553057", "553476",
		"553481", "553480", "553768", "553771", "553372", "553058", "553397", "553377", "553426",
		"553059", "553060", "553062", "553063", "553416", "553479", "553444", "553769", "553064",
		"553475", "553065", "553066", "553067", "553543", "553068", "553361", "553069", "553359",
		"553546", "553547", "553767", "553378", "553070", "553457", "553766", "553551", "553071",
		"553398", "553765", "553421", "553072", "553554", "553557", "553432", "553408", "553764",
		"553354", "553453", "553763", "553073", "553074", "553561", "553451", "553463", "553366",
		"553084", "553567", "553762", "553422", "553578", "553405", "553401", "553761", "553758",
		"553087", "553089", "553449", "553581", "553584", "553093", "553096", "553376", "553756",
		"553434", "553753", "553439", "553752", "553448", "553751", "553749", "553098", "553387",
		"553588", "553441", "553747", "553590", "553380", "553741", "553745", "553396", "553740",
		"553100", "553103", "553450", "553739", "553593", "553594", "553105", "553436", "553737",
		"553595", "553442", "553596", "553736", "553598", "553599", "553356", "553452", "553600",
		"553601", "553602", "553735", "553603", "553605", "553446", "553607", "553400",
	},
}

func okForTokens(set *Set) bool {
	return slices.Contains(setAllowedForTokens, set.Code) ||
		strings.Contains(set.Name, "Duel Deck")
}

func skipSet(set *Set) bool {
	// Skip unsupported sets
	switch set.Code {
	case "PRED", // a single foreign card
		"PSAL", "PS11", "PHUK", // salvat05, salvat11, hachette
		"OLGC", "OLEP", "OVNT", "O90P": // oversize
		return true
	}
	// Skip online sets, and any token-based sets
	if set.IsOnlineOnly ||
		(set.Type == "token" && !okForTokens(set)) ||
		strings.HasSuffix(set.Name, "Art Series") ||
		strings.HasSuffix(set.Name, "Minigames") ||
		strings.HasSuffix(set.Name, "Front Cards") ||
		strings.Contains(set.Name, "Heroes of the Realm") {
		return true
	}
	// In case there is nothing interesting in the set
	if len(set.Cards)+len(set.Tokens)+len(set.SealedProduct) == 0 {
		return true
	}
	return false
}

func sortPrintings(ap AllPrintings, printings []string) {
	sort.Slice(printings, func(i, j int) bool {
		setDateI, errI := time.Parse("2006-01-02", ap.Data[printings[i]].ReleaseDate)
		setDateJ, errJ := time.Parse("2006-01-02", ap.Data[printings[j]].ReleaseDate)
		if errI != nil || errJ != nil {
			return false
		}

		if setDateI.Equal(setDateJ) {
			return ap.Data[printings[i]].Name < ap.Data[printings[j]].Name
		}

		return setDateI.After(setDateJ)
	})
}

func generateImageURL(card Card, version string) string {
	_, found := card.Identifiers["scryfallId"]
	if !found {
		tcgId, found := card.Identifiers["tcgplayerProductId"]
		if !found {
			return ""
		}
		if version == "small" {
			// This size is the default "small" format
			tcgId = "fit-in/146x204/" + tcgId
		}
		return "https://product-images.tcgplayer.com/" + tcgId + ".jpg"
	}

	number := card.Number

	// Retrieve the original number if present
	dupe, found := card.Identifiers["originalScryfallNumber"]
	if found {
		number = dupe
	}

	// Support BAN's custom sets
	code := strings.ToLower(card.SetCode)
	if strings.HasSuffix(code, "ita") {
		code = strings.TrimSuffix(code, "ita")
		number += "/it"
	} else if strings.HasSuffix(code, "jpn") {
		code = strings.TrimSuffix(code, "jpn")
		number += "/ja"
	}
	code = strings.TrimSuffix(code, "alt")

	return fmt.Sprintf("https://api.scryfall.com/cards/%s/%s?format=image&version=%s", code, number, version)
}

func (ap AllPrintings) Load() cardBackend {
	uuids := map[string]CardObject{}
	cardInfo := map[string]cardinfo{}
	scryfall := map[string]string{}
	tcgplayer := map[string]string{}
	alternates := map[string]alternateProps{}
	commanderKeywordMap := map[string]string{}
	var promoTypes []string
	var allCardNames []string
	var tokens []string
	var allSets []string

	for code, set := range ap.Data {
		// Filer out unneeded data
		if skipSet(set) {
			delete(ap.Data, code)
			continue
		}

		// Load all possible card names
		for _, card := range set.Cards {
			if !slices.Contains(allCardNames, card.Name) {
				allCardNames = append(allCardNames, card.Name)
			}
		}

		// Load token names (that don't have the same name of a real card)
		for _, token := range set.Tokens {
			if !slices.Contains(tokens, token.Name) && !slices.Contains(allCardNames, token.Name) {
				tokens = append(tokens, token.Name)
			}
		}
	}

	for code, set := range ap.Data {
		var filteredCards []Card
		var rarities, colors []string

		allSets = append(allSets, code)

		allCards := set.Cards

		if okForTokens(set) {
			// Append tokens to the list of considered cards
			// if they are not named in the same way of a real card
			for _, token := range set.Tokens {
				if !slices.Contains(allCardNames, token.Name) {
					allCards = append(allCards, token)
				}
			}
		} else {
			// Clean a bit of memory
			set.Tokens = nil
		}

		switch set.Code {
		// Remove reference to an online-only set
		case "PMIC":
			set.ParentCode = ""
		}

		for _, card := range allCards {
			// Skip anything non-paper
			if card.IsOnlineOnly {
				continue
			}

			card.Images = map[string]string{}
			card.Images["full"] = generateImageURL(card, "normal")
			card.Images["thumbnail"] = generateImageURL(card, "small")
			card.Images["crop"] = generateImageURL(card, "art_crop")

			// Custom modifications or skips
			switch set.Code {
			// Override non-English Language
			case "FBB":
				card.Language = "Italian"
			case "4BB":
				card.Language = "Japanese"
			// Missing variant tags
			case "PALP":
				card.FlavorText = missingPALPtags[card.Number]
			case "PELP":
				card.FlavorText = missingPELPtags[card.Number]
			// Remove frame effects and borders where they don't belong
			case "STA", "PLST":
				card.PromoTypes = nil
				card.FrameEffects = nil
				card.BorderColor = "black"
			case "SLD":
				switch card.Number {
				// One of the tokens is a DFC but burns a card number, skip it
				case "28":
					continue
				// Source is "technically correct" but it gets too messy to track
				case "589":
					card.Finishes = []string{"nonfoil", "etched"}
				default:
					num, _ := strconv.Atoi(card.Number)
					// Override the frame type for the Braindead drops
					if (num >= 821 && num <= 824) || (num >= 1652 && num <= 1666) {
						card.FrameVersion = "2015"
					}
				}
			// Only keep dungeons, and fix their layout to make sure they are tokens
			case "AFR":
				if card.SetCode == "TAFR" {
					switch card.Number {
					case "20", "21", "22":
						card.Layout = "token"
					default:
						continue
					}
				}
			// Override all to tokens so that duplicates get named differently
			case "TFTH", "TBTH", "TDAG":
				card.Layout = "token"
			// Modify the Normalize string replacer to ignore replacing card names with commas
			// that conflict with another card name
			case "MB2", "DA1", "UNK":
				if strings.Contains(card.Name, ",") && slices.Contains(allCardNames, strings.Replace(card.Name, ",", "", 1)) {
					lower := strings.ToLower(card.Name)
					replacerStrings = append([]string{lower, lower}, replacerStrings...)
					replacer = strings.NewReplacer(replacerStrings...)
				}
			}

			// Override any "double_faced_token" entries and emblems
			if strings.Contains(card.Layout, "token") || card.Layout == "emblem" {
				card.Layout = "token"
			}

			// Make sure this property is correctly initialized
			if strings.HasSuffix(card.Number, "p") && !slices.Contains(card.PromoTypes, mtgjson.PromoTypePromoPack) {
				card.PromoTypes = append(card.PromoTypes, mtgjson.PromoTypePromoPack)
			}

			// Rename DFCs into a single name
			dfcSameName := card.IsDFCSameName()
			if dfcSameName {
				card.Name = strings.Split(card.Name, " // ")[0]
			}

			for i, name := range []string{card.FaceName, card.FlavorName, card.FaceFlavorName} {
				// Skip empty entries
				if name == "" {
					continue
				}
				// Skip FaceName entries that could be aliased
				// ie 'Start' could be Start//Finish and Start//Fire
				switch name {
				case "Bind",
					"Fire",
					"Smelt",
					"Start":
					continue
				}
				// Skip faces of DFCs with same names that aren't reskin version of other cars
				if dfcSameName && card.FlavorName == "" {
					continue
				}
				// Rename the sub-name of a DFC card
				if dfcSameName {
					name = strings.Split(name, " // ")[0]
				}

				// If the name is unique, keep track of the numbers so that they
				// can be decoupled later for reprints of the main card.
				// If the name is not unique, we might overwrite data and lose
				// track of the main version
				props := alternateProps{
					OriginalName:   card.Name,
					OriginalNumber: card.Number,
					IsFlavor:       i > 0,
				}
				_, found := alternates[name]
				if found {
					props.OriginalNumber = ""
				}
				alternates[name] = props
			}

			// MTGJSON v5 contains duplicated card info for each face, and we do
			// not need that level of detail, so just skip any extra side.
			if card.Side != "" && card.Side != "a" {
				continue
			}

			// Filter out unneeded printings
			var printings []string
			for i := range card.Printings {
				subset, found := ap.Data[card.Printings[i]]
				// If not found it means the set was already deleted above
				if !found || skipSet(subset) {
					continue
				}
				printings = append(printings, card.Printings[i])
			}
			// Sort printings by most recent sets first
			sortPrintings(ap, printings)

			card.Printings = printings

			// Tokens do not come with a printing array, add it
			// It'll be updated later with the sets discovered so far
			if card.Layout == "token" {
				card.Printings = []string{set.Code}
			}

			// Now assign the card to the list of cards to be saved
			filteredCards = append(filteredCards, card)

			// Quick dictionary of valid card names and their printings
			name := card.Name

			// Due to several cards having the same name of a token we hardcode
			// this value to tell them apart in the future -- checks and names
			// are still using the official Scryfall name (without the extra Token)
			if card.Layout == "token" {
				name += " Token"
			}

			// Deduplicate clashing names
			if slices.Contains(duplicatedCardNames, name) &&
				(strings.Contains(set.Name, "Playtest") || strings.Contains(set.Name, "Unknown")) {
				name += " Playtest"
			}

			norm := Normalize(name)
			_, found := cardInfo[norm]
			if !found {
				cardInfo[norm] = cardinfo{
					Name:      card.Name,
					Printings: card.Printings,
					Layout:    card.Layout,
				}
			} else if card.Layout == "token" {
				// If already present, check if this set is already contained
				// in the current array, otherwise add it
				// Note the setCode will be from the parent
				if !slices.Contains(cardInfo[norm].Printings, code) {
					printings := append(cardInfo[norm].Printings, set.Code)
					sortPrintings(ap, printings)

					ci := cardinfo{
						Name:      card.Name,
						Printings: printings,
						Layout:    card.Layout,
					}
					cardInfo[norm] = ci
				}
			}

			// Custom properties for tokens
			if card.Layout == "token" {
				card.Printings = cardInfo[Normalize(card.Name+" Token")].Printings
				card.Rarity = "token"
			}
			if card.IsOversized {
				card.Rarity = "oversize"
			}

			// Initialize custom lookup tables
			scryfallId, found := card.Identifiers["scryfallId"]
			if found {
				scryfall[scryfallId] = card.UUID
			}
			for _, tag := range []string{"tcgplayerProductId", "tcgplayerEtchedProductId"} {
				tcgplayerId, found := card.Identifiers[tag]
				if found {
					tcgplayer[tcgplayerId] = card.UUID
				}
			}

			// Shared card object
			co := CardObject{
				Card:    card,
				Edition: set.Name,
			}

			// Save the original uuid
			co.Identifiers["mtgjsonId"] = card.UUID

			// Append "_f" and "_e" to uuids, unless etched is the only printing.
			// If it's not etched, append "_f", unless foil is the only printing.
			// Leave uuids unchanged, if there is a single printing of any kind.
			if card.HasFinish(mtgjson.FinishEtched) {
				uuid := card.UUID

				// Etched + Nonfoil [+ Foil]
				if card.HasFinish(mtgjson.FinishNonfoil) {
					// Save the card object
					uuids[uuid] = co
				}

				// Etched + Foil
				if card.HasFinish(mtgjson.FinishFoil) {
					// Set the main property
					co.Foil = true
					// Make sure "_f" is appended if a different version exists
					if card.HasFinish(mtgjson.FinishNonfoil) {
						uuid = card.UUID + suffixFoil
						co.UUID = uuid
					}
					// Save the card object
					uuids[uuid] = co
				}

				// Etched
				// Set the main properties
				co.Foil = false
				co.Etched = true
				// If there are alternative finishes, always append the suffix
				if card.HasFinish(mtgjson.FinishNonfoil) || card.HasFinish(mtgjson.FinishFoil) {
					uuid = card.UUID + suffixEtched
					co.UUID = uuid
				}
				// Save the card object
				uuids[uuid] = co
			} else if card.HasFinish(mtgjson.FinishFoil) {
				uuid := card.UUID

				// Foil [+ Nonfoil]
				if card.HasFinish(mtgjson.FinishNonfoil) {
					// Save the card object
					uuids[uuid] = co

					// Update the uuid for the *next* finish type
					uuid = card.UUID + suffixFoil
					co.UUID = uuid
				}

				// Foil
				co.Foil = true
				// Save the card object
				uuids[uuid] = co
			} else {
				// Single printing, use as-is
				uuids[card.UUID] = co
			}

			// Add to the ever growing list of promo types
			for _, promoType := range card.PromoTypes {
				if !slices.Contains(promoTypes, promoType) {
					promoTypes = append(promoTypes, promoType)
				}
			}

			// Add possible rarities and colors
			if !slices.Contains(rarities, card.Rarity) {
				rarities = append(rarities, card.Rarity)
			}
			for _, color := range card.Colors {
				if !slices.Contains(colors, mtgColorNameMap[color]) {
					colors = append(colors, mtgColorNameMap[color])
				}
			}
			if len(card.Colors) == 0 && !slices.Contains(colors, "colorless") {
				colors = append(colors, "colorless")
			}
			if len(card.Colors) > 1 && !slices.Contains(colors, "multicolor") {
				colors = append(colors, "multicolor")
			}

		}

		// Replace the original array with the filtered one
		set.Cards = filteredCards

		// Assign the rarities and colors present in the set
		sort.Slice(rarities, func(i, j int) bool {
			return mtgRarityMap[rarities[i]] > mtgRarityMap[rarities[j]]
		})
		set.Rarities = rarities
		sort.Slice(colors, func(i, j int) bool {
			return mtgColorMap[colors[i]] > mtgColorMap[colors[j]]
		})
		set.Colors = colors

		// Adjust the setBaseSize to take into account the cards with
		// the same name in the same set (also make sure that it is
		// correctly initialized)
		setDate, err := time.Parse("2006-01-02", set.ReleaseDate)
		if err != nil {
			continue
		}
		if setDate.After(PromosForEverybodyYay) {
			for _, card := range set.Cards {
				if card.HasPromoType(mtgjson.PromoTypeBoosterfun) {
					// Usually boosterfun cards have real numbers
					cn, err := strconv.Atoi(card.Number)
					if err == nil {
						set.BaseSetSize = cn - 1
					}
					break
				}
			}
		}

		// Retrieve the best describing word for a commander set and save it for later reuse
		if strings.HasSuffix(set.Name, "Commander") && !strings.Contains(set.Name, "Display") {
			keyword := longestWordInEditionName(strings.TrimSuffix(set.Name, "Commander"))
			commanderKeywordMap[keyword] = set.Name
		}

		for _, product := range set.SealedProduct {
			if product.Identifiers == nil {
				product.Identifiers = map[string]string{}
			}
			product.Identifiers["mtgjsonId"] = product.UUID

			card := Card{
				UUID:        product.UUID,
				Name:        product.Name,
				SetCode:     code,
				Identifiers: product.Identifiers,
				Rarity:      "product",
				Layout:      product.Category,
				Side:        product.Subtype,
				// Will be filled later
				SourceProducts: map[string][]string{},
				Images:         map[string]string{},
			}

			card.Images["full"] = generateImageURL(card, "normal")
			card.Images["thumbnail"] = generateImageURL(card, "small")
			card.Images["crop"] = generateImageURL(card, "normal")

			uuids[product.UUID] = CardObject{
				Card:    card,
				Sealed:  true,
				Edition: set.Name,
			}
		}
	}

	duplicate(ap.Data, cardInfo, uuids, "Legends Italian", "LEG", "ITA", "1995-09-01")
	duplicate(ap.Data, cardInfo, uuids, "The Dark Italian", "DRK", "ITA", "1995-08-01")
	duplicate(ap.Data, cardInfo, uuids, "Alternate Fourth Edition", "4ED", "ALT", "1995-04-01")
	allSets = append(allSets, "LEGITA", "DRKITA", "4EDALT")

	duplicateCards(ap.Data, uuids, "SLD", "JPN", sldJPNLangDupes)
	duplicateCards(ap.Data, uuids, "PURL", "JPN", []string{"1"})

	for setCode, numbers := range foilDupes {
		spinoffFoils(ap.Data, uuids, setCode, numbers, tcgIds[setCode])
	}

	// Add all names and associated uuids to the global names and hashes arrays
	hashes := map[string][]string{}
	var names, fullNames, lowerNames []string
	var sealed, fullSealed, lowerSealed []string
	for uuid, card := range uuids {
		norm := Normalize(card.Name)
		_, found := hashes[norm]
		if !found {
			if card.Sealed {
				sealed = append(sealed, norm)
				fullSealed = append(fullSealed, card.Name)
				lowerSealed = append(lowerSealed, strings.ToLower(card.Name))
			} else {
				names = append(names, norm)
				fullNames = append(fullNames, card.Name)
				lowerNames = append(lowerNames, strings.ToLower(card.Name))
			}
		}
		hashes[norm] = append(hashes[norm], uuid)
	}
	// Add all alternative names too
	var altNames []string
	for altName, altProps := range alternates {
		altNorm := Normalize(altName)
		_, found := hashes[altNorm]
		if !found {
			altNames = append(altNames, altNorm)
			fullNames = append(fullNames, altName)
			lowerNames = append(lowerNames, strings.ToLower(altName))
		}
		if altProps.IsFlavor {
			// Retrieve all the uuids with a FlavorName attached
			allAltUUIDs := hashes[Normalize(altProps.OriginalName)]
			for _, uuid := range allAltUUIDs {
				if uuids[uuid].FlavorName != "" {
					hashes[altNorm] = append(hashes[altNorm], uuid)
				}
			}
		} else {
			// Copy the original uuids, avoiding duplicates for the cards already added
			for _, hash := range hashes[Normalize(altProps.OriginalName)] {
				if slices.Contains(hashes[altNorm], hash) {
					continue
				}
				hashes[altNorm] = append(hashes[altNorm], hash)
			}
		}
	}

	// Finally save all the  uuids generated
	var allUUIDs []string
	var allSealedUUIDs []string
	for uuid, co := range uuids {
		if co.Sealed {
			allSealedUUIDs = append(allSealedUUIDs, uuid)
			continue
		}
		allUUIDs = append(allUUIDs, uuid)
	}

	// Remove promo tags that apply to a single finish only
	for uuid, card := range uuids {
		if !card.Foil && !card.Etched {
			for _, promoType := range []string{
				mtgjson.PromoTypeDoubleExposure,
				mtgjson.PromoTypeSilverFoil,
				mtgjson.PromoTypeRainbowFoil,
				mtgjson.PromoTypeRippleFoil,
			} {
				if card.HasPromoType(promoType) {
					var filtered []string
					for _, pt := range card.PromoTypes {
						if pt != promoType {
							filtered = append(filtered, pt)
						}
					}
					card.PromoTypes = filtered
					uuids[uuid] = card
				}
			}
		}
	}

	sort.Strings(promoTypes)
	sort.Strings(allSets)

	sort.Strings(names)
	sort.Strings(fullNames)
	sort.Strings(lowerNames)
	sort.Strings(sealed)
	sort.Strings(fullSealed)
	sort.Strings(lowerSealed)

	fillinSealedContents(ap.Data, uuids)

	var backend cardBackend

	backend.Hashes = hashes
	backend.AllSets = allSets
	backend.AllUUIDs = allUUIDs
	backend.AllSealedUUIDs = allSealedUUIDs

	backend.AllNames = names
	backend.AllCanonicalNames = fullNames
	backend.AllLowerNames = lowerNames

	backend.AllSealed = sealed
	backend.AllCanonicalSealed = fullSealed
	backend.AllLowerSealed = lowerSealed

	backend.Sets = ap.Data
	backend.CardInfo = cardInfo
	backend.Tokens = tokens
	backend.UUIDs = uuids
	backend.Scryfall = scryfall
	backend.Tcgplayer = tcgplayer
	backend.AlternateProps = alternates
	backend.AlternateNames = altNames
	backend.AllPromoTypes = promoTypes

	backend.CommanderKeywordMap = commanderKeywordMap
	backend.SLDDeckNames = fillinSLDdecks(ap.Data["SLD"])

	return backend
}

var mtgRarityMap = map[string]int{
	"token":    1,
	"common":   2,
	"uncommon": 3,
	"rare":     4,
	"mythic":   5,
	"special":  6,
	"oversize": 7,
}

var mtgColorNameMap = map[string]string{
	"W": "white",
	"U": "blue",
	"B": "black",
	"R": "red",
	"G": "green",
}

var mtgColorMap = map[string]int{
	"white":      7,
	"blue":       6,
	"black":      5,
	"red":        4,
	"green":      3,
	"colorless":  2,
	"multicolor": 1,
}

func fillinSLDdecks(set *Set) []string {
	var output []string
	for _, product := range set.SealedProduct {
		if strings.HasPrefix(product.Name, "Secret Lair Commander") {
			name := strings.TrimPrefix(product.Name, "Secret Lair Commander Deck ")
			if !slices.Contains(output, name) {
				output = append(output, name)
			}
		}
	}
	return output
}

// Add a map of which kind of products sealed contains
func fillinSealedContents(sets map[string]*Set, uuids map[string]CardObject) {
	result := map[string][]string{}
	tmp := map[string][]string{}

	for _, set := range sets {
		for _, product := range set.SealedProduct {
			dedup := map[string]int{}
			list := SealedWithinSealed(set.Code, product.UUID)
			for _, item := range list {
				dedup[item]++
			}
			for uuid := range dedup {
				tmp[product.UUID] = append(tmp[product.UUID], uuid)
			}
		}
	}

	// Reverse to be compatible with SourceProducts model
	for _, list := range tmp {
		for _, item := range list {
			for key, sublist := range tmp {
				// Add if item is in the sublist, and the key was not already added
				if slices.Contains(sublist, item) && !slices.Contains(result[item], key) {
					result[item] = append(result[item], key)
				}
			}
		}
	}

	for uuid, co := range uuids {
		if !co.Sealed {
			continue
		}

		res, found := result[uuid]
		if !found {
			continue
		}

		uuids[uuid].SourceProducts["sealed"] = res
	}
}

// Match the name of the deck with the product UUID(s)
func findDeck(setCode, deckName string) []string {
	var list []string

	set, found := backend.Sets[setCode]
	if !found {
		return nil
	}

	for _, deck := range set.Decks {
		if deck.Name != deckName {
			continue
		}
		list = append(list, deck.SealedProductUUIDs...)
	}

	return list
}

// Return a list of sealed products contained by the input product
// Decks and Packs and Card cannot contain other sealed product, so they are ignored here
func SealedWithinSealed(setCode, sealedUUID string) []string {
	var list []string

	set, found := backend.Sets[setCode]
	if !found {
		return nil
	}

	for _, product := range set.SealedProduct {
		if sealedUUID != product.UUID {
			continue
		}

		for key, contents := range product.Contents {
			for _, content := range contents {
				switch key {
				case "sealed":
					list = append(list, content.UUID)

				case "variable":
					for _, config := range content.Configs {
						for _, sealed := range config["sealed"] {
							list = append(list, sealed.UUID)
						}
						for _, deck := range config["deck"] {
							decklist := findDeck(deck.Set, deck.Name)
							list = append(list, decklist...)
						}
					}
				}
			}
		}
	}

	return list
}

var langs = map[string]string{
	"JPN": "Japanese",
	"ITA": "Italian",
	"ALT": "English",
}

func duplicate(sets map[string]*Set, cardInfo map[string]cardinfo, uuids map[string]CardObject, name, code, tag, date string) {
	// Copy base set information
	dup := *sets[code]

	// Update with new info
	dup.Name = name
	dup.Code = code + tag
	dup.ParentCode = code
	dup.ReleaseDate = date

	// Copy card information
	dup.Cards = make([]Card, len(sets[code].Cards))
	for i := range sets[code].Cards {
		// Skip misprints from main sets
		if strings.HasSuffix(sets[code].Cards[i].Number, mtgjson.SuffixVariant) {
			continue
		}

		// Update printings for the original set
		printings := append(sets[code].Cards[i].Printings, dup.Code)
		sets[code].Cards[i].Printings = printings

		// Loop through all other sets mentioned
		for _, setCode := range printings {
			// Skip the set being added, there might be cards containing
			// the set code being processed due to variants
			if setCode == dup.Code {
				continue
			}
			_, found := sets[setCode]
			if !found {
				continue
			}
			if skipSet(sets[setCode]) {
				continue
			}

			for j := range sets[setCode].Cards {
				// Name match, can't break after the first because there could
				// be other variants
				if sets[setCode].Cards[j].Name == sets[code].Cards[i].Name {
					sets[setCode].Cards[j].Printings = printings
				}
			}
		}

		// Update with new info
		dup.Cards[i] = sets[code].Cards[i]
		dup.Cards[i].UUID += "_" + strings.ToLower(tag)
		dup.Cards[i].SetCode = dup.Code
		dup.Cards[i].Language = langs[tag]

		// Update images
		dup.Cards[i].Images = map[string]string{}
		dup.Cards[i].Images["full"] = generateImageURL(dup.Cards[i], "normal")
		dup.Cards[i].Images["thumbnail"] = generateImageURL(dup.Cards[i], "small")
		dup.Cards[i].Images["crop"] = generateImageURL(dup.Cards[i], "art_crop")

		// Update printings for the CardInfo map
		ci := cardInfo[Normalize(dup.Cards[i].Name)]
		ci.Printings = printings
		cardInfo[Normalize(dup.Cards[i].Name)] = ci

		// Remove store references to avoid duplicates
		altIdentifiers := map[string]string{}
		for k, v := range dup.Cards[i].Identifiers {
			altIdentifiers[k] = v
		}
		delete(altIdentifiers, "tcgplayerProductId")
		dup.Cards[i].Identifiers = altIdentifiers

		// Add the new uuid to the UUID map
		uuids[dup.Cards[i].UUID] = CardObject{
			Card:    dup.Cards[i],
			Edition: name,
		}
	}

	sets[dup.Code] = &dup
}

func duplicateCards(sets map[string]*Set, uuids map[string]CardObject, code, tag string, numbers []string) {
	var duplicates []Card

	for i := range sets[code].Cards {
		// Skip unneeded
		if !slices.Contains(numbers, sets[code].Cards[i].Number) {
			continue
		}

		mainUUID := sets[code].Cards[i].UUID

		// Update with new info
		dupeCard := sets[code].Cards[i]
		dupeCard.UUID = mainUUID + "_" + strings.ToLower(tag)
		dupeCard.Language = langs[tag]
		dupeCard.Identifiers["originalScryfallNumber"] = dupeCard.Number
		dupeCard.Number += strings.ToLower(tag)

		// Update images
		dupeCard.Images = map[string]string{}
		dupeCard.Images["full"] = generateImageURL(dupeCard, "normal")
		dupeCard.Images["thumbnail"] = generateImageURL(dupeCard, "small")
		dupeCard.Images["crop"] = generateImageURL(dupeCard, "art_crop")

		duplicates = append(duplicates, dupeCard)

		// Add the new uuid to the UUID map
		for _, suffixTag := range []string{suffixEtched, suffixFoil, ""} {
			uuid := mainUUID + suffixTag
			co, found := uuids[uuid]
			if !found {
				continue
			}

			dupeCard.UUID = mainUUID + "_" + strings.ToLower(tag) + suffixTag
			uuids[dupeCard.UUID] = CardObject{
				Card:    dupeCard,
				Edition: sets[code].Name,
				Etched:  co.Etched,
				Foil:    co.Foil,
			}
		}
	}

	sets[code].Cards = append(sets[code].Cards, duplicates...)
}

func spinoffFoils(sets map[string]*Set, uuids map[string]CardObject, code string, numbers []string, tcgIds []string) {
	if tcgIds != nil && len(numbers) != len(tcgIds) {
		panic("different length of duped numbers and duped ids")
	}

	var newCardsArray []Card

	for i := range sets[code].Cards {
		dupeCard := sets[code].Cards[i]
		ogVariations := dupeCard.Variations
		ogUUID := dupeCard.UUID

		// Skip unneeded (just preserve the card as-is)
		if !slices.Contains(numbers, sets[code].Cards[i].Number) {
			newCardsArray = append(newCardsArray, dupeCard)
			continue
		}

		// Retrieve the main card object
		co, found := uuids[dupeCard.UUID]
		if !found {
			continue
		}

		// Change properties
		dupeCard.Finishes = []string{"nonfoil"}
		dupeCard.Variations = append(ogVariations, ogUUID+suffixFoil)

		// Propagate changes across the board
		co.Card = dupeCard
		uuids[dupeCard.UUID] = co
		newCardsArray = append(newCardsArray, dupeCard)

		// Move to the foil version
		co, found = uuids[dupeCard.UUID+suffixFoil]
		if !found {
			continue
		}

		// Change properties
		dupeCard.UUID += suffixFoil
		if tcgIds != nil {
			// Clone the map and replace it, overriding the id
			newIdentifiers := map[string]string{}
			for k, v := range dupeCard.Identifiers {
				newIdentifiers[k] = v
			}

			dupeCard.Identifiers = newIdentifiers
			dupeCard.Identifiers["tcgplayerProductId"] = tcgIds[slices.Index(numbers, sets[code].Cards[i].Number)]
			// Signal that the TCG SKUs from MTGJSON are not reliable
			dupeCard.Identifiers["needsNewTCGSKUs"] = "true"
		}
		dupeCard.Identifiers["originalScryfallNumber"] = dupeCard.Number
		dupeCard.Number += mtgjson.SuffixSpecial
		dupeCard.Finishes = []string{"foil"}
		dupeCard.Variations = append(ogVariations, ogUUID)

		// Update images
		dupeCard.Images = map[string]string{}
		dupeCard.Images["full"] = generateImageURL(dupeCard, "normal")
		dupeCard.Images["thumbnail"] = generateImageURL(dupeCard, "small")
		dupeCard.Images["crop"] = generateImageURL(dupeCard, "art_crop")

		// Update or create the new card object, add the new card to the list
		co.Card = dupeCard
		uuids[dupeCard.UUID] = co
		newCardsArray = append(newCardsArray, dupeCard)
	}

	sets[code].Cards = newCardsArray
}

func SetGlobalDatastore(datastore cardBackend) {
	backend = datastore
}

func LoadDatastore(reader io.Reader) error {
	var buf bytes.Buffer
	tee := io.TeeReader(reader, &buf)

	datastore, err := LoadAllPrintings(tee)
	if err != nil {
		datastore, err = LoadLorcana(&buf)
		if err != nil {
			return err
		}
	}

	backend = datastore.Load()
	return nil
}

func LoadDatastoreFile(filename string) error {
	reader, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer reader.Close()
	return LoadDatastore(reader)
}

func SetGlobalLogger(userLogger *log.Logger) {
	logger = userLogger
}
