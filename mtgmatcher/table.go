package mtgmatcher

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

var LanguageTag2LanguageCode = map[string]string{
	"":                    "en",
	"English":             "en",
	"French":              "fr",
	"German":              "de",
	"Italian":             "it",
	"Japanese":            "ja",
	"Korean":              "ko",
	"Russian":             "ru",
	"Spanish":             "es",
	"Portuguese":          "pt",
	"Chinese Simplified":  "zhs",
	"Chinese Traditional": "zht",
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

	// Global series
	"GS1",

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
	"CLB",
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
	"Fast", "Furious", "Fast // Furious",
}

// List of numbers in SLD that need to be decoupled
var sldJPNLangDupes = []string{
	// Special Guests Yoji Shinkawa
	"1110", "1111", "1112", "1113",
	// Special Guests Junji Ito
	"1114", "1115", "1116", "1117",
	// Miku Sakura Superstar
	"1587", "1594", "1596", "1597", "805", "808",
	"1587★", "1594★", "1596★", "1597★",
	// Miku Digital Sensation
	"1592", "1595", "1599", "1603", "1604", "1607", "806",
	// Miku Electric Entourage
	"1585", "1590", "1593", "1598", "1600", "807",
	// Miku Winter Diva
	"1586", "1588", "1589", "1591", "1601", "1606", "804",
	// Final Fantasy Game Over
	"1858", "1859", "1860", "1861", "1862",
	// Final Fantasy Weapons
	"1863", "1864", "1865", "1866", "1867",
	// Final Fantasy Grimoire
	"1868", "1869", "1870", "1871", "1872",
	// Summer Superdrop 2025 promo
	"909",
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
		"9990", "9991", "9992", "9993", "9994",

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
		"2005", "2006", "2007", "2008",
		"2071", "2072", "2073", "2074", "2075", "7028",
		"2010", "2011", "2012", "2013", // Food tokens
		"2076", "2077", "2078", "2079", "2080",

		"1980", "1981", "1982", "1983", "1984",
		"2024", "2025", "2026", "2027", "2029", "2030", "2031", "2049", "2057", "2058", "2059", "2060",
		"7001", "7002", "7003", "7029",

		"2057", "2058", "2059", "2060", "2061", "2062", "2063", "2064", "2065", "7027",
		"2009", "2028", "2037", "2038", "2039", "2040", "2041", "2047", "2048", "2050", "2051",

		"2081", "2082", "2083", "2084", "2085", "2086", "2087", "2088", "2089", "2090",
		"2091", "2092", "2093", "2094", "2095", "2096", "2097", "2098", "2099", "2100", "2101",

		"2102", "2103", "2104", "2105", "2106",
		"2107", "2108", "2109", "2110", "2111",
	},
	"M3C": {
		"32", "33", "34", "35", "36", "37", "38", "39", "40", "41", "42", "43",
		"44", "45", "46", "47", "48", "49", "50", "51", "52", "53", "54", "55",
		"56", "57", "58", "59", "60", "61", "62", "63", "64", "65", "66", "67",
		"68", "69", "70", "71", "72", "73", "74", "75", "76", "77", "78", "79",
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
		"400", "401", "402", "403", "404", "405", "406", "407", "408", "409",
		"410", "411",
	},
	"FIC": {
		"9",
		"10", "11", "12", "13", "14", "15", "16", "17", "18", "19", "20", "21",
		"22", "23", "24", "25", "26", "27", "28", "29", "30", "31", "32", "33",
		"34", "35", "36", "37", "38", "39", "40", "41", "42", "43", "44", "45",
		"46", "47", "48", "49", "50", "51", "52", "53", "54", "55", "56", "57",
		"58", "59", "60", "61", "62", "63", "64", "65", "66", "67", "68", "69",
		"70", "71", "72", "73", "74", "75", "76", "77", "78", "79", "80", "81",
		"82", "83", "84", "85", "86", "87", "88", "89", "90", "91", "92", "93",
		"94", "95", "96", "97", "98", "99", "100",
		"229", "230", "231", "232", "233", "234", "235", "236", "237", "238",
		"239", "240", "241", "242", "243", "244", "245", "246", "247", "248",
		"249", "250", "251", "252", "253", "254", "255", "256", "257", "258",
		"259", "260", "261", "262", "263", "264", "265", "266", "267", "268",
		"269", "270", "271", "272", "273", "274", "275", "276", "277", "278",
		"279", "280", "281", "282", "283", "284", "285", "286", "287", "288",
		"289", "290", "291", "292", "293", "294", "295", "296", "297", "298",
		"299", "300", "301", "302", "303", "304", "305", "306", "307", "308",
		"309", "310", "311", "312", "313", "314", "315", "316", "317", "318",
		"319", "320", "321", "322", "323", "324", "325", "326", "327", "328",
		"329", "330", "331", "332", "333", "334", "335", "336", "337", "338",
		"339", "340", "341", "342", "343", "344", "345", "346", "347", "348",
		"349", "350", "351", "352", "353", "354", "355", "356", "357", "358",
		"359", "360", "361", "362", "363", "364", "365", "366", "367", "368",
		"369", "370", "371", "372", "373", "374", "375", "376", "377", "378",
		"379", "380", "381", "382", "383", "384", "385", "386", "387", "388",
		"389", "390", "391", "392", "393", "394", "395", "396", "397", "398",
		"399", "400", "401", "402", "403", "404", "405", "406", "407", "408",
		"409", "410", "411", "412", "413", "414", "415", "416", "417", "418",
		"419", "420", "421", "422", "423", "424", "425", "426", "427", "428",
		"429", "430", "431", "432", "433", "434", "435", "436", "437", "438",
		"439", "440", "441", "484", "485", "486",
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
		"549480",

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

		"632096", "632099", "632102", "632105", "632137", "632141", "632148",
		"632155", "632158", "632107", "632109", "632120", "632124", "632127",
		"629929", "629934", "629939", "629941",
		"629904", "629906", "629908", "629910", "629912", "632760",
		"626615", "626621", "626624", "626625", // Food tokens
		"629914", "629916", "629919", "629921", "629924",

		"637704", "637708", "637710", "637702", "637706",
		"637685", "637686", "637689", "637691", "637698", "637696", "637700", "629976", "636334", "636336", "636338", "636340",
		"638080", "638078", "638073", "632759",

		"636334", "636336", "636338", "636340", "636342", "636344", "636346", "636348", "636350", "639447",
		"629943", "637694", "629986", "629989", "629992", "629995", "629998", "629967", "629970", "629980", "629983",

		"643699", "643702", "643705", "643708", "643709", "643713", "643714", "643623", "643681", "643683",
		"643687", "643691", "643696", "643698", "643601", "643604", "643606", "643608", "643614", "643618", "643620",

		"646789", "646792", "646795", "646804", "646805",
		"646814", "646820", "646822", "646824", "646826",
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
	"FIC": {
		"636747", "636764", "636788", "636791", "636793", "636798", "636799", "636816", "636818",
		"636831", "636859", "636861", "636871", "636872", "636901", "636935", "636980", "636990",
		"636991", "636995", "636996", "637016", "637021", "637034", "636748", "636778", "636779",
		"636874", "636880", "636902", "636919", "636920", "636947", "636994", "636834", "636841",
		"637022", "636842", "636887", "636942", "636944", "636964", "636968", "637032", "636767",
		"636785", "636860", "636865", "636955", "636974", "636986", "636989", "636992", "637035",
		"637036", "637042", "637051", "636784", "636897", "636905", "636932", "636982", "636993",
		"636997", "637033", "637052", "637053", "636746", "636761", "636770", "636771", "636830",
		"636832", "636835", "636878", "636891", "636893", "636894", "636900", "636904", "636910",
		"636923", "636943", "636961", "636963", "636966", "636969", "637039", "637046", "636808",
		"637023", "637050", "636751", "636758", "636765", "636766", "636773", "636782", "636795",
		"636796", "636800", "636815", "636817", "636824", "636826", "636843", "636848", "636866",
		"636899", "636903", "636922", "636926", "636933", "636937", "636946", "636958", "636960",
		"636998", "637004", "637008", "637027", "637030", "637038", "637041", "636760", "636790",
		"636825", "636881", "636884", "636888", "636750", "636934", "636936", "636949", "636987",
		"637029", "636759", "636772", "636813", "636840", "636896", "636911", "636914", "636917",
		"636927", "636941", "636948", "636962", "636975", "636984", "636985", "637009", "636752",
		"636777", "636789", "636802", "636850", "636856", "636873", "636931", "636953", "637040",
		"636769", "636814", "636829", "636844", "636847", "636854", "636862", "636868", "636869",
		"636870", "636883", "636885", "636915", "636925", "636939", "636940", "637025", "637026",
		"636749", "636768", "636775", "636776", "636781", "636812", "636821", "636833", "636845",
		"636895", "636912", "636930", "636954", "637043", "637045", "636754", "636755", "636756",
		"636757", "636762", "636787", "636801", "636807", "636809", "636811", "636819", "636836",
		"636839", "636875", "636876", "636886", "636892", "636898", "636906", "636907", "636908",
		"636909", "636945", "636970", "636976", "636977", "636978", "636979", "636981", "637006",
		"637007", "637010", "637011", "637012", "637013", "637014", "637024", "637028", "637031",
		"637047", "637048", "636753", "636763", "636774", "636780", "636783", "636786", "636792",
		"636794", "636797", "636803", "636810", "636820", "636822", "636823", "636827", "636828",
		"636837", "636838", "636846", "636849", "636851", "636852", "636853", "636855", "636857",
		"636858", "636863", "636864", "636867", "636877", "636879", "636882", "636889", "636890",
		"636913", "636916", "636918", "636921", "636924", "636928", "636929", "636938", "636950",
		"636951", "636952", "636956", "636957", "636959", "636965", "636967", "636971", "636972",
		"636973", "636983", "636988", "636999", "637000", "637001", "637002", "637003", "637005",
		"637015", "637017", "637018", "637019", "637020", "637037", "637044", "637049", "636804",
		"636805", "636806",
	},
}
