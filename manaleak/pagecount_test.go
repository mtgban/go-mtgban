package manaleak

import "testing"

// TestPageCount pins that the fan-out is sized off a count that was read:
// a listing whose count is missing says zero, and a first page holding
// products beside a count of zero is a page the count was not read off.
func TestPageCount(t *testing.T) {
	for _, tt := range []struct {
		total, onFirstPage, want int
		err                      bool
	}{
		{0, 0, 0, false},
		{pageLimit, pageLimit, 1, false},
		{pageLimit + 1, pageLimit, 2, false},
		{0, 12, 0, true},
		{5, 12, 0, true},
	} {
		got, err := pageCount(tt.total, tt.onFirstPage)
		if (err != nil) != tt.err || got != tt.want {
			t.Errorf("pageCount(%d, %d) = %d, %v; want %d, error %v", tt.total, tt.onFirstPage, got, err, tt.want, tt.err)
		}
	}
}
