package matcher

import (
	"strings"
	"testing"
)

// setByCode is a ToUpper lookup that does not allocate for ascii; the
// answer has to match the plain one for every shape of input.
func TestSetByCodeMatchesToUpper(t *testing.T) {
	b := &Backend{Sets: map[string]*Set{
		"M10": {Code: "M10"}, "ÆTHER": {Code: "ÆTHER"}, "A-B": {Code: "A-B"},
		strings.Repeat("Z", 200): {Code: "long"},
	}}
	for _, code := range []string{
		"m10", "M10", "m1O", "æther", "ÆTHER", "a-b", "A-b", "", "Magic 2010",
		strings.Repeat("z", 200), strings.Repeat("z", 201), strings.Repeat("Z", 128),
	} {
		set, found := b.setByCode(code)
		want, wantFound := b.Sets[strings.ToUpper(code)]
		if found != wantFound || set != want {
			t.Errorf("setByCode(%q) = %v,%v; ToUpper lookup gives %v,%v", code, set, found, want, wantFound)
		}
	}
	allocs := testing.AllocsPerRun(100, func() {
		b.setByCode("Magic 2010")
	})
	if allocs != 0 {
		t.Errorf("setByCode allocates %v times for an ascii name", allocs)
	}
}
