package cardmarket

import "testing"

func TestContentRangeCovered(t *testing.T) {
	tests := []struct {
		name         string
		pagesFetched int
		pageSize     int
		total        int
		capped       bool
		want         bool
	}{
		{"a page short of the total is not covered", 1, 10, 15, false, false},
		{"exactly the total is covered", 2, 10, 20, false, true},
		{"past the total is covered", 2, 10, 15, false, true},
		{"an unknown total (0, header missing or unparseable) is never covered", 5, 10, 0, false, false},
		{"a capped total is never treated as covered, however many pages", 100, 10, 1000, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contentRangeCovered(tt.pagesFetched, tt.pageSize, tt.total, tt.capped)
			if got != tt.want {
				t.Errorf("contentRangeCovered(%d, %d, %d, %v) = %v, want %v",
					tt.pagesFetched, tt.pageSize, tt.total, tt.capped, got, tt.want)
			}
		})
	}
}
