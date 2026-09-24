package mtgseattle

import "testing"

func TestBuildProductURL(t *testing.T) {
	tests := []struct {
		name    string
		product string
		want    string
		wantErr bool
	}{
		{
			name:    "plain category href gets layout=false appended",
			product: "/catalog/magic_singles-masters_sets-modern_masters_2015/1433",
			want:    "https://www.mtgseattle.com/catalog/magic_singles-masters_sets-modern_masters_2015/1433?layout=false",
		},
		{
			// The old code appended a second "?layout=false" here,
			// producing .../2271?layout=false&page=17&sort_by_price=1?layout=false.
			name:    "pagination href already carrying layout=false is merged, not duplicated",
			product: "/catalog/magic_singles-streets_of_new_capenna/2271?layout=false&page=17&sort_by_price=1",
			want:    "https://www.mtgseattle.com/catalog/magic_singles-streets_of_new_capenna/2271?layout=false&page=17&sort_by_price=1",
		},
		{
			name:    "pagination href without layout=false gets it added",
			product: "/catalog/magic_singles-kaldheim/2244?page=37&sort_by_price=1",
			want:    "https://www.mtgseattle.com/catalog/magic_singles-kaldheim/2244?layout=false&page=37&sort_by_price=1",
		},
		{
			name:    "unparseable href is an error, not a malformed request",
			product: "/catalog/foo%zz",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildProductURL(tt.product)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("buildProductURL(%q) = %q, want an error", tt.product, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("buildProductURL(%q) unexpected error: %v", tt.product, err)
			}
			if got != tt.want {
				t.Errorf("buildProductURL(%q) = %q, want %q", tt.product, got, tt.want)
			}
		})
	}
}
