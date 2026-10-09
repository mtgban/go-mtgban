package strikezone

import (
	"net/url"
	"regexp"

	"github.com/PuerkitoBio/goquery"
)

// cartCode is the cart id a listing row's link names in param, "Buy" on the
// buylist and "Add" in the store: the id Strike Zone's CSV cart import takes
// where importID can compute it, and the link's own code otherwise.
func cartCode(el *goquery.Selection, param string) string {
	link, err := url.Parse(el.Find(`a[href*="`+param+`="]`).AttrOr("href", ""))
	if err != nil {
		return ""
	}
	code := link.Query().Get(param)
	id := importID(code)
	if id == "" {
		return code
	}
	return id
}

var importCode = regexp.MustCompile(`^637-C-(\d{5,6})-(\d{3})$`)

// importShifts are added, modulo 10, to the digits of a code's item and
// variant read as one string.
var importShifts = [9]byte{8, 9, 7, 2, 9, 1, 8, 9, 7}

// importID turns the code a Strike Zone cart link carries,
// 637-C-<item>-<variant>, into the id the cart's CSV import takes and its
// export writes, USCIDU-637-F-<item>-<variant>-<check>-<check>, with the
// same id for buying and selling a card. It returns "" for a code of any
// shape but a five- or six-digit item and a three-digit variant, the only
// shapes it was measured on.
//
// The item and variant digits are shifted by importShifts. The first check
// is importCheck of the plain C-<item>-<variant> with offset 111, its letters
// then shifted by 15, 4 and 17 for a five-digit item or 4, 17 and 3 for a
// six-digit one; the second is importCheck of everything before it, with
// offset 89. Strike Zone publishes none of this: it was reconstructed from
// ids its cart exports and BidWicket's theme demo, and pinned by the ids in
// cartid_test.go.
func importID(code string) string {
	m := importCode.FindStringSubmatch(code)
	if m == nil {
		return ""
	}
	item, variant := m[1], m[2]

	digits := []byte(item + variant)
	for i, d := range digits {
		digits[i] = '0' + (d-'0'+importShifts[i])%10
	}

	shifts := [3]byte{15, 4, 17}
	if len(item) == 6 {
		shifts = [3]byte{4, 17, 3}
	}
	check := []byte(importCheck("C-"+item+"-"+variant, 111))
	for i := range check {
		check[i] = 'A' + (check[i]-'A'+shifts[i])%26
	}

	prefix := "USCIDU-637-F-" + string(digits[:len(item)]) + "-" + string(digits[len(item):]) + "-" + string(check)
	return prefix + "-" + importCheck(prefix, 89)
}

// importCheck is three letters from Java's String.hashCode of text and a
// hyphen, plus offset, made positive: the number divided by 1, 26 and 676,
// each quotient taken modulo 25 into an alphabet with Z in place of O.
func importCheck(text string, offset int32) string {
	const alphabet = "ABCDEFGHIJKLMNZPQRSTUVWXY"

	var h int32
	for _, c := range text + "-" {
		h = 31*h + c
	}
	n := int64(h + offset)
	if n < 0 {
		n = -n
	}

	out := make([]byte, 3)
	for i := range out {
		out[i] = alphabet[n%25]
		n /= 26
	}
	return string(out)
}
