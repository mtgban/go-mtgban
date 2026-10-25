package gundam

import "testing"

// TestWordsDescribeWholeWords pins that a label is named by its words and
// not by a run of letters inside another: "Special" does not name the sp
// printing, and a note saying "Ahead" says nothing about a head one.
func TestWordsDescribeWholeWords(t *testing.T) {
	for _, tt := range []struct {
		wording string
		labels  []string
		want    bool
	}{
		{"ST01-001 SP", []string{"sp"}, true},
		{"ST01-001 Special", []string{"sp"}, false},
		{"Ahead of the curve", []string{"head"}, false},
		{"Newtype Challenge 2025", []string{"newtypechallenge"}, true},
		{"Newtype", []string{"newtypechallenge"}, false},
		{"Store Tournament Winner", []string{"storetournament", "winner"}, true},
		{"Store Tournament Winners", []string{"storetournament", "winner"}, false},
		{"anything", nil, false},
	} {
		if got := wordsDescribe(tt.wording, tt.labels); got != tt.want {
			t.Errorf("wordsDescribe(%q, %v) = %v, want %v", tt.wording, tt.labels, got, tt.want)
		}
	}
}
