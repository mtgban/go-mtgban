package magiccorner

import "testing"

// The older sets' reprints are told apart in the variants table by the
// store's own image name, and the store's "(Version N)" tag names none of
// them. These pin that the tag steps aside there and nowhere else.
func TestInternalPreprocessVersionTag(t *testing.T) {
	tests := []struct {
		desc      string
		name      string
		edition   string
		variation string
		extra     string
		want      string
	}{
		{
			desc:      "version tag yields to the image name",
			name:      "Abbey Matron",
			edition:   "Homelands",
			variation: "Version 1",
			extra:     "OR003",
			want:      "OR003",
		},
		{
			desc:      "the other art reaches its own image name",
			name:      "Abbey Matron",
			edition:   "Homelands",
			variation: "Version 2",
			extra:     "AbbeyMatron",
			want:      "AbbeyMatron",
		},
		{
			desc:      "an empty variation still reaches the image name",
			name:      "Aesthir Glider",
			edition:   "Alliances",
			variation: "",
			extra:     "AZ002",
			want:      "AZ002",
		},
		{
			desc:      "a variation that names something keeps it",
			name:      "Aesthir Glider",
			edition:   "Alliances",
			variation: "Foreign White Border",
			extra:     "AZ002",
			want:      "Foreign White Border",
		},
		{
			desc:      "the tag survives where no table keys on the image",
			name:      "Agent of Treachery",
			edition:   "Core Set 2020 Promos",
			variation: "Version 1",
			extra:     "M20001",
			want:      "Version 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			_, _, got := internalPreprocess(tt.name, tt.edition, tt.variation, tt.extra)
			if got != tt.want {
				t.Errorf("variation = %q, want %q", got, tt.want)
			}
		})
	}
}
