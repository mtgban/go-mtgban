package coolstuffinc

import "testing"

// TestNameQualifiers pins what a One Piece buylist name carries behind the
// card's own name. That feed spends the bracket on the qualifier telling one
// printing from another, and repeats the collector number in a bracket of its
// own, which says nothing the number field has not.
func TestNameQualifiers(t *testing.T) {
	for _, tt := range []struct {
		desc, name, want string
	}{
		{"the qualifier the feed spends the bracket on",
			"Kouzuki Hiyori (Parallel)", "Parallel"},
		{"a repeated collector number is left behind",
			"Roronoa Zoro (025) (Parallel)", "Parallel"},
		{"and a bare number on its own",
			"Charlotte Katakuri (123)", ""},
		{"two qualifiers both travel",
			"Charlotte Katakuri (123) (Alternate Art) (PRB)", "Alternate Art PRB"},
		{"a name with no bracket says nothing",
			"Monkey.D.Luffy", ""},
		{"an empty bracket says nothing either",
			"Monkey.D.Luffy ()", ""},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := nameQualifiers(tt.name); got != tt.want {
				t.Errorf("nameQualifiers(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}
