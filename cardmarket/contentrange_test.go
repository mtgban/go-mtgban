package cardmarket

import "testing"

func TestContentRangeCovered(t *testing.T) {
	tests := []struct {
		name         string
		pagesFetched int
		pageSize     int
		total        int
		want         bool
	}{
		{"a page short of the total is not covered", 1, 10, 15, false},
		{"exactly the total is covered", 2, 10, 20, true},
		{"past the total is covered", 2, 10, 15, true},
		{"an unknown total (0, header missing or unparseable) is never covered", 5, 10, 0, false},
		{"the 1000-result API ceiling is not covered after nine pages", 9, 100, 1000, false},
		{"the 1000-result API ceiling is covered after ten pages", 10, 100, 1000, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contentRangeCovered(tt.pagesFetched, tt.pageSize, tt.total)
			if got != tt.want {
				t.Errorf("contentRangeCovered(%d, %d, %d) = %v, want %v",
					tt.pagesFetched, tt.pageSize, tt.total, got, tt.want)
			}
		})
	}
}
