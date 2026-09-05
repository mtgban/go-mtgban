package magic

import "testing"

// TestPlainNumberKeepsTheListNumbers pins the numbers The List is filed
// under. It names a card by the set it was drawn from, "ARB-1", and those end
// in a digit, so the letters trimmed off a number's tail never reach them:
// 5,582 of its 5,584 numbers keep the plain form they had. The two this does
// reach are here as well, so what the change touches is written down rather
// than assumed.
func TestPlainNumberKeepsTheListNumbers(t *testing.T) {
	for _, tt := range []struct {
		number, want string
	}{
		{"ARB-1", "ARB-1"},
		{"BBD-1", "BBD-1"},
		{"CSP-1", "CSP-1"},
		{"MID-123", "MID-123"},
		{"M19-185", "M19-185"},
		{"POR-57", "POR-57"},
		// The two The List numbers a tail reaches, each a treatment of the
		// number standing beside it rather than a number of its own.
		{"POR-57s", "POR-57"},
		{"M19-185j", "M19-185"},
		// And a mark, which the older rule already reached.
		{"JUD-78†", "JUD-78"},
	} {
		t.Run(tt.number, func(t *testing.T) {
			got := plainNumber(tt.number)
			if got != tt.want {
				t.Errorf("plainNumber(%q) = %q, want %q", tt.number, got, tt.want)
			}
		})
	}
}
