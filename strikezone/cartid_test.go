package strikezone

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

// Ids exported from Strike Zone carts on 2026-10-09, after adding each code
func TestImportID(t *testing.T) {
	for _, tc := range []struct {
		code string
		want string
	}{
		// Every grade of one card
		{"637-C-30199-106", "USCIDU-637-F-19818-285-CFS-JKR"},
		{"637-C-30199-108", "USCIDU-637-F-19818-287-QCR-WZD"},
		{"637-C-30199-109", "USCIDU-637-F-19818-288-JAR-FMN"},
		{"637-C-30199-110", "USCIDU-637-F-19818-299-CZP-WIK"},
		{"637-C-30571-106", "USCIDU-637-F-19290-285-OVN-RMS"},
		// The shifted item keeps its leading zero
		{"637-C-29931-106", "USCIDU-637-F-08650-285-LKO-GXZ"},
		{"637-C-29932-106", "USCIDU-637-F-08651-285-KON-XMA"},
		// Six-digit items, foil and not
		{"637-C-181059-106", "USCIDU-637-F-978240-993-XAK-QHC"},
		{"637-C-181138-105", "USCIDU-637-F-978329-992-HUO-NYQ"},
		{"637-C-180944-105", "USCIDU-637-F-977135-992-VIA-YUS"},
		{"637-C-181286-105", "USCIDU-637-F-978477-992-DWK-XJM"},
		{"637-C-181287-105", "USCIDU-637-F-978478-992-TSL-LJL"},
		{"637-C-181288-105", "USCIDU-637-F-978479-992-UNM-XJX"},
		{"637-C-181290-105", "USCIDU-637-F-978481-992-DBG-ZKX"},
		{"637-C-181291-105", "USCIDU-637-F-978482-992-TXH-NUJ"},
		{"637-C-181292-105", "USCIDU-637-F-978483-992-JNZ-VCE"},
		// Shapes it was never measured on
		{"637-C-1234-106", ""},
		{"637-C-1234567-106", ""},
		{"637-C-30571-10", ""},
		{"637-X-30571-106", ""},
		{"638-C-30571-106", ""},
		{"", ""},
	} {
		got := importID(tc.code)
		if got != tc.want {
			t.Errorf("importID(%q) = %q, want %q", tc.code, got, tc.want)
		}
	}
}

func TestCartCode(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<table>
<tr id="buy"><td><a href="/Item/Unlimited/Ankh_of_Mishra_F19290285.html">Ankh of Mishra</a>
<td><a href="/TUser?MC=CUVC&amp;Buy=637-C-30571-106&amp;MF=B&amp;BUID=637">Sell to Us</a></tr>
<tr id="add"><td><a href="/Item/Unlimited/Ankh_of_Mishra_F19290285.html">Ankh of Mishra</a>
<td><a href="/TUser?MC=CUVC&amp;Add=637-C-30571-106&amp;MF=B&amp;BUID=637">Add to Cart</a></tr>
<tr id="odd"><td><a href="/TUser?MC=CUVC&amp;Buy=637-C-1234-106&amp;MF=B&amp;BUID=637">Sell to Us</a></tr>
<tr id="none"><td>Ankh of Mishra</tr>
</table>`))
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		row   string
		param string
		want  string
	}{
		{"buy", "Buy", "USCIDU-637-F-19290-285-OVN-RMS"},
		{"add", "Add", "USCIDU-637-F-19290-285-OVN-RMS"},
		// A code importID cannot compute stays the link's own
		{"odd", "Buy", "637-C-1234-106"},
		{"none", "Buy", ""},
	} {
		got := cartCode(doc.Find("#"+tc.row), tc.param)
		if got != tc.want {
			t.Errorf("cartCode(%s, %s) = %q, want %q", tc.row, tc.param, got, tc.want)
		}
	}
}
