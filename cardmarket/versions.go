package cardmarket

import (
	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// versionPrintings names the printing a Magic product sells where the id map
// cannot tell: a "(V.N)" version the map links to no printing of its shelf,
// to a sibling's, or to several. Keyed by Cardmarket's product id; the tag
// after each name says what placed the row, and
// docs/agents/cardmarket-census/versions.md what each tag means.
var versionPrintings = map[int]struct{ set, number string }{
	// Arabian Nights
	6794: {"ARN", "33"}, // Stone-Throwing Devils (V.1), map lists two

	// Pro Tour 1996: Bertrand Lestree
	22465: {"PTC", "bl17sb"}, // Circle of Protection: Red (V.1), map lists two
	22471: {"PTC", "bl16b"},  // Order of Leitbur (V.1), map lists two
	22496: {"PTC", "bl366"},  // Plains (V.3), map lists two

	// Pro Tour 1996: Eric Tam
	22658: {"PTC", "et83sb"}, // Autumn Willow (V.2), map lists two
	22683: {"PTC", "et374"},  // Mountain (V.1), map lists two
	22690: {"PTC", "et365"},  // Plains (V.3), map lists two
	22697: {"PTC", "et52sb"}, // Swords to Plowshares (V.2), map lists two

	// Pro Tour 1996: George Baxter
	22709: {"PTC", "gb38c"}, // Hymn to Tourach (V.3), map lists two
	22728: {"PTC", "gb371"}, // Swamp (V.1), map lists two

	// Pro Tour 1996: Leon Lindback
	22750: {"PTC", "ll54sb"}, // Jalum Tome (V.2), map lists two
	22757: {"PTC", "ll42c"},  // Order of the Ebon Hand (V.2), map lists two
	22774: {"PTC", "ll38b"},  // Hymn to Tourach (V.4), map lists two

	// Pro Tour 1996: Mark Justice
	22781: {"PTC", "mj184sb"}, // Detonate (V.2), map lists two
	22804: {"PTC", "mj365"},   // Plains (V.3), map lists two

	// Pro Tour 1996: Michael Locanto
	22841: {"PTC", "ml365"}, // Plains (V.2), map lists two

	// Pro Tour 1996: Preston Poulter
	22872: {"PTC", "pp377"}, // Forest (V.2), map lists two
	22883: {"PTC", "pp366"}, // Plains (V.1), map lists two

	// Pro Tour 1996: Shawn Regnier
	22908: {"PTC", "shr368"}, // Island (V.1), map lists two
	22919: {"PTC", "shr365"}, // Plains (V.3), map lists two

	// WCD 1997: Jakub Slemr
	242161: {"WC97", "js440"}, // Swamp (V.3), image
	242162: {"WC97", "js438"}, // Swamp (V.4), swap

	// WCD 1997: Janosch Kuhn
	242108: {"WC97", "jk26"},   // Disenchant (V.1), sideboard
	242127: {"WC97", "jk26sb"}, // Disenchant (V.2), sideboard
	242131: {"WC97", "jk445"},  // Mountain (V.2), image
	242133: {"WC97", "jk442"},  // Mountain (V.4), swap
	242135: {"WC97", "jk433"},  // Plains (V.3), CardTrader

	// WCD 1997: Paul McCabe
	250101: {"WC97", "pm78sb"}, // Pyrokinesis (V.2), CardTrader
	250104: {"WC97", "pm434"},  // Island (V.1), swap
	250106: {"WC97", "pm436"},  // Island (V.3), image
	250108: {"WC97", "pm442"},  // Mountain (V.1), swap
	250110: {"WC97", "pm444"},  // Mountain (V.3), image
	250111: {"WC97", "pm445"},  // Mountain (V.4), CardTrader

	// WCD 1997: Svend Geertsen
	250070: {"WC97", "sg341"},   // Whirling Dervish (V.1), sideboard
	250071: {"WC97", "sg341sb"}, // Whirling Dervish (V.2), sideboard
	250073: {"WC97", "sg123sb"}, // Uktabi Orangutan (V.2), CardTrader
	250078: {"WC97", "sg85sb"},  // Bounty of the Hunt (V.2), CardTrader
	250080: {"WC97", "sg446"},   // Forest (V.1), swap
	250082: {"WC97", "sg448"},   // Forest (V.3), image

	// WCD 1998: Randy Buehler
	249897: {"WC98", "rb335"}, // Island (V.1), swap
	249900: {"WC98", "rb338"}, // Island (V.4), image

	// WCD 1998: Brian Hacker
	249872: {"WC98", "bh7bsb"}, // Aura of Silence (V.2), CardTrader
	249876: {"WC98", "bh16sb"}, // Disenchant (V.2), CardTrader
	249878: {"WC98", "bh331"},  // Plains (V.1), swap
	249881: {"WC98", "bh334"},  // Plains (V.4), image

	// WCD 1998: Ben Rubin
	249851: {"WC98", "br343"}, // Mountain (V.1), image
	249852: {"WC98", "br344"}, // Mountain (V.2), swap
	249854: {"WC98", "br346"}, // Mountain (V.4), CardTrader

	// WCD 1998: Brian Selden
	249818: {"WC98", "bs347"}, // Forest (V.1), swap
	249820: {"WC98", "bs349"}, // Forest (V.3), image

	// WCD 1999: Jakub Šlemr
	249777: {"WC99", "js67"},   // Rapid Decay (V.1), sideboard
	249778: {"WC99", "js67sb"}, // Rapid Decay (V.2), sideboard
	249786: {"WC99", "js340b"}, // Swamp (V.3), left over

	// WCD 1999: Matt Linde
	249756: {"WC99", "ml347b"}, // Forest (V.3), left over

	// WCD 1999: Mark Le Pine
	249720: {"WC99", "mlp173"},   // Fireslinger (V.1), sideboard
	249721: {"WC99", "mlp173sb"}, // Fireslinger (V.2), sideboard
	249734: {"WC99", "mlp344"},   // Mountain (V.2), image
	249735: {"WC99", "mlp346"},   // Mountain (V.3), left over

	// WCD 1999: Kai Budde
	249700: {"WC99", "kb302"},   // Mishra's Helix (V.1), sideboard
	249701: {"WC99", "kb302sb"}, // Mishra's Helix (V.2), sideboard

	// WCD 2000: Tom Van de Logt
	249640: {"WC00", "tvdl18sb"}, // Seal of Cleansing (V.2), CardTrader
	249648: {"WC00", "tvdl54sb"}, // Wrath of God (V.2), CardTrader

	// WCD 2000: Janosch Kühn
	249607: {"WC00", "jk134"},   // Masticore (V.1), sideboard
	249608: {"WC00", "jk134sb"}, // Masticore (V.2), sideboard
	249611: {"WC00", "jk220"},   // Creeping Mold (V.1), sideboard
	249612: {"WC00", "jk220sb"}, // Creeping Mold (V.2), sideboard
	249617: {"WC00", "jk306sb"}, // Phyrexian Processor (V.2), CardTrader

	// WCD 2000: Jon Finkel
	249585: {"WC00", "jf302sb"}, // Mishra's Helix (V.2), CardTrader

	// WCD 2001: Jan Tomcani
	249555: {"WC01", "jt191sb"}, // Kavu Chameleon (V.1), CardTrader
	249561: {"WC01", "jt348"},   // Forest (V.1), image
	249562: {"WC01", "jt328"},   // Forest (V.2), swap
	249565: {"WC01", "jt343a"},  // Mountain (V.1), image
	249566: {"WC01", "jt337"},   // Mountain (V.2), swap
	250142: {"WC01", "jt349a"},  // Forest (V.6), CardTrader

	// WCD 2001: Antoine Ruel
	249523: {"WC01", "ar67"},    // Counterspell (V.1), swap
	249533: {"WC01", "ar131sb"}, // Duress (V.2), CardTrader
	250137: {"WC01", "ar338"},   // Island (V.5), image
	250140: {"WC01", "ar334"},   // Island (V.8), left over
	316302: {"WC01", "ar69"},    // Counterspell (V.2), image

	// WCD 2001: Alex Borteh
	249506: {"WC01", "ab69"},   // Counterspell (V.1), CardTrader
	249512: {"WC01", "ab338"},  // Island (V.4), CardTrader
	250127: {"WC01", "ab336"},  // Island (V.6), image
	250129: {"WC01", "ab338a"}, // Island (V.8), CardTrader
	250130: {"WC01", "ab332"},  // Island (V.9), swap

	// WCD 2001: Tom van de Logt
	249474: {"WC01", "tvdl97"},   // Crypt Angel (V.1), sideboard
	249475: {"WC01", "tvdl97sb"}, // Crypt Angel (V.2), sideboard
	249482: {"WC01", "tvdl339"},  // Swamp (V.1), swap
	249487: {"WC01", "tvdl343b"}, // Mountain (V.2), image
	249488: {"WC01", "tvdl337"},  // Mountain (V.3), swap
	250135: {"WC01", "tvdl347"},  // Swamp (V.5), image

	// WCD 2002: Sim Han How
	249453: {"WC02", "shh347"},  // Forest (V.1), image
	249454: {"WC02", "shh348"},  // Forest (V.2), CardTrader
	249455: {"WC02", "shh349"},  // Forest (V.3), CardTrader
	249457: {"WC02", "shh335"},  // Island (V.1), CardTrader
	249458: {"WC02", "shh336a"}, // Island (V.2), CardTrader
	249459: {"WC02", "shh337"},  // Island (V.3), CardTrader
	249460: {"WC02", "shh338"},  // Island (V.4), image
	250041: {"WC02", "shh328"},  // Forest (V.5), swap
	250042: {"WC02", "shh329"},  // Forest (V.6), CardTrader
	250043: {"WC02", "shh330"},  // Forest (V.7), CardTrader
	250044: {"WC02", "shh331"},  // Forest (V.8), CardTrader
	250045: {"WC02", "shh334"},  // Island (V.5), swap

	// WCD 2002: Raphael Levy
	249423: {"WC02", "rl30sb"}, // Rushing River (V.2), CardTrader
	249427: {"WC02", "rl347"},  // Forest (V.1), CardTrader
	249428: {"WC02", "rl348"},  // Forest (V.2), CardTrader
	249429: {"WC02", "rl349"},  // Forest (V.3), CardTrader
	249430: {"WC02", "rl350"},  // Forest (V.4), CardTrader
	250047: {"WC02", "rl329"},  // Forest (V.6), CardTrader
	250048: {"WC02", "rl330"},  // Forest (V.7), CardTrader
	250049: {"WC02", "rl331"},  // Forest (V.8), CardTrader
	250052: {"WC02", "rl332"},  // Island (V.7), swap
	250053: {"WC02", "rl333"},  // Island (V.8), image
	250054: {"WC02", "rl334"},  // Island (V.9), CardTrader

	// WCD 2002: Brian Kibler
	249382: {"WC02", "bk11sb"},  // Glory (V.2), CardTrader
	249390: {"WC02", "bk347"},   // Forest (V.1), image
	249393: {"WC02", "bk328"},   // Forest (V.4), swap
	249396: {"WC02", "bk327sb"}, // City of Brass (V.2), CardTrader
	249402: {"WC02", "bk331a"},  // Plains (V.1), swap
	249403: {"WC02", "bk333"},   // Plains (V.2), image

	// WCD 2002: Carlos Romao
	249351: {"WC02", "cr57sb"}, // Fact or Fiction (V.2), CardTrader
	249360: {"WC02", "cr336"},  // Island (V.4), left over
	249364: {"WC02", "cr340"},  // Swamp (V.1), swap
	249365: {"WC02", "cr341"},  // Swamp (V.2), image
	250057: {"WC02", "cr332"},  // Island (V.7), swap
	250060: {"WC02", "cr335b"}, // Island (V.10), image

	// WCD 2003: Peer Kröger
	249313: {"WC03", "pk118"},   // Buried Alive (V.1), sideboard
	249314: {"WC03", "pk118sb"}, // Buried Alive (V.2), sideboard
	249317: {"WC03", "pk62sb"},  // Cabal Therapy (V.2), CardTrader
	249320: {"WC03", "pk216"},   // Recoup (V.1), sideboard
	249321: {"WC03", "pk216sb"}, // Recoup (V.2), sideboard
	249323: {"WC03", "pk72sb"},  // Stitch Together (V.2), CardTrader
	249327: {"WC03", "pk344"},   // Mountain (V.1), swap
	249329: {"WC03", "pk346"},   // Mountain (V.3), image
	249332: {"WC03", "pk339"},   // Swamp (V.1), swap
	249333: {"WC03", "pk340"},   // Swamp (V.2), image

	// WCD 2003: Wolfgang Eder
	249286: {"WC03", "we170sb"}, // Smother (V.2), CardTrader
	249295: {"WC03", "we340"},   // Swamp (V.1), image
	249296: {"WC03", "we339"},   // Swamp (V.2), swap

	// WCD 2003: Dave Humpherys
	249247: {"WC03", "dh54sb"},  // Wonder (V.2), CardTrader
	249249: {"WC03", "dh122"},   // Krosan Reclamation (V.1), sideboard
	249250: {"WC03", "dh122sb"}, // Krosan Reclamation (V.2), sideboard
	249251: {"WC03", "dh20"},    // Ray of Revelation (V.1), sideboard
	249252: {"WC03", "dh20sb"},  // Ray of Revelation (V.2), sideboard
	249254: {"WC03", "dh112sb"}, // Unsummon (V.2), CardTrader
	249257: {"WC03", "dh36sb"},  // Deep Analysis (V.2), CardTrader
	249262: {"WC03", "dh348"},   // Forest (V.1), swap
	249263: {"WC03", "dh349"},   // Forest (V.2), image
	249266: {"WC03", "dh336"},   // Island (V.1), swap
	249267: {"WC03", "dh337"},   // Island (V.2), image

	// WCD 2003: Daniel Zink
	249208: {"WC03", "dz33"},   // Circular Logic (V.1), sideboard
	249209: {"WC03", "dz33sb"}, // Circular Logic (V.2), sideboard
	249214: {"WC03", "dz50sb"}, // Renewed Faith (V.2), CardTrader
	249215: {"WC03", "dz21"},   // Vengeful Dreams (V.1), sideboard
	249216: {"WC03", "dz21sb"}, // Vengeful Dreams (V.2), sideboard
	249222: {"WC03", "dz347"},  // Forest (V.1), swap
	249223: {"WC03", "dz348"},  // Forest (V.2), image
	249226: {"WC03", "dz335"},  // Island (V.1), swap
	249228: {"WC03", "dz337"},  // Island (V.3), image
	249231: {"WC03", "dz331"},  // Plains (V.1), swap
	249232: {"WC03", "dz332"},  // Plains (V.2), image

	// WCD 2004: Gabriel Nassif
	249195: {"WC04", "gn331"}, // Plains (V.1), swap
	249196: {"WC04", "gn332"}, // Plains (V.2), image

	// WCD 2004: Julien Nuijten
	249116: {"WC04", "jn272"},   // Plow Under (V.1), sideboard
	249117: {"WC04", "jn272sb"}, // Plow Under (V.2), sideboard
	249126: {"WC04", "jn333"},   // Plains (V.3), CardTrader

	// Chronicles: Japanese
	272488: {"BCHR", "114b"}, // Urza's Mine (V.2), map lists four
	272502: {"BCHR", "115b"}, // Urza's Power Plant (V.2), map lists four
	272552: {"BCHR", "116a"}, // Urza's Tower (V.1), map lists four
}

// versionPrinting answers the printing versionPrintings names for a Magic
// product, or "" where it names none the datastore carries.
func versionPrinting(b *mtgmatcher.Backend, product *cm.Product) string {
	printing, found := versionPrintings[product.IDProduct]
	if !found {
		return ""
	}
	cards := b.MatchInSetNumber(mtgmatcher.SplitVariants(product.Name)[0], printing.set, printing.number)
	if len(cards) != 1 {
		return ""
	}
	return cards[0].UUID
}
