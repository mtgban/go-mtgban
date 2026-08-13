package onepiece

import "testing"

// TestFullNumberShapes pins which collector numbers count as one, the
// shape extractNumber reads before it falls back to the first
// digit-leading word. cardtrader's collectorNumberRe holds a copy of this
// regexp to decide whether a blueprint's Version may ride behind the
// number, and cardtrader/utils_test.go asserts the same table: a shape
// gained here has to be gained there too, or the scraper starts appending
// wording to a number the matcher cannot parse.
func TestFullNumberShapes(t *testing.T) {
	tests := map[string]bool{
		"OP01-001":   true,
		"P-043":      true,
		"OP01-001a":  true,
		"OP07-047P2": false,
		"P-L":        false,
		"2024":       false,
		"":           false,
	}

	for number, want := range tests {
		if got := fullNumberRe.MatchString(number); got != want {
			t.Errorf("fullNumberRe.MatchString(%q) = %v, want %v", number, got, want)
		}
	}
}
