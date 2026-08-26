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

// The Ravnica guildgates are told apart by the number in their image name,
// and the tag steps aside only where a number is actually there to read.
func TestNamesTheArt(t *testing.T) {
	tests := []struct {
		desc    string
		edition string
		extra   string
		want    bool
	}{
		{
			desc:    "a numbered image says which art",
			edition: "Guilds of Ravnica",
			extra:   "GRN243",
			want:    true,
		},
		{
			desc:    "a slug in the same edition says nothing",
			edition: "Guilds of Ravnica",
			extra:   "guilds-of-ravnica-boros-guildgate-260044",
			want:    false,
		},
		{
			desc:    "the older sets are keyed on the image itself",
			edition: "Homelands",
			extra:   "AbbeyMatron",
			want:    true,
		},
		{
			desc:    "everywhere else the wording is what speaks",
			edition: "Core Set 2020 Promos",
			extra:   "M20001",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := namesTheArt(tt.edition, tt.extra)
			if got != tt.want {
				t.Errorf("namesTheArt(%q, %q) = %v, want %v",
					tt.edition, tt.extra, got, tt.want)
			}
		})
	}
}

// The image name carries the English card name for the listings whose own
// name field does not, along with the art index when it has one.
func TestImageName(t *testing.T) {
	tests := []struct {
		desc          string
		extra         string
		edition       string
		wantName      string
		wantVariation string
	}{
		{
			desc:     "the id comes off the end",
			extra:    "basal-sliver_695839",
			edition:  "Secret Lair Drop Series",
			wantName: "basal sliver",
		},
		{
			desc:     "so does a hexadecimal one",
			extra:    "ember-island-production_0f79a7fc",
			edition:  "Avatar: The Last Airbender",
			wantName: "ember island production",
		},
		{
			desc:     "the edition comes off the front",
			extra:    "the-brothers-war-curate-683384",
			edition:  "The Brothers' War",
			wantName: "curate",
		},
		{
			desc:          "the art index is kept as the variation",
			extra:         "the-brothers-war-island-v3-683345",
			edition:       "The Brothers' War",
			wantName:      "island",
			wantVariation: "V.3",
		},
		{
			desc:    "a collector number names no card",
			extra:   "GRN243",
			edition: "Guilds of Ravnica",
		},
		{
			desc:    "neither does an image with no id",
			extra:   "catToken",
			edition: "Scars of Mirrodin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			name, variation := imageName(tt.extra, tt.edition)
			if name != tt.wantName || variation != tt.wantVariation {
				t.Errorf("imageName(%q, %q) = %q, %q, want %q, %q",
					tt.extra, tt.edition, name, variation,
					tt.wantName, tt.wantVariation)
			}
		})
	}
}

// A name holding an apostrophe sometimes arrives quoted the way a database
// quotes it.
func TestUnquote(t *testing.T) {
	tests := []struct {
		desc string
		name string
		want string
	}{
		{
			desc: "the quoting comes off",
			name: "'Skyseer''s Chariot (V.1)'",
			want: "Skyseer's Chariot (V.1)",
		},
		{
			desc: "a plain name is left alone",
			name: "Skyseer's Chariot",
			want: "Skyseer's Chariot",
		},
		{
			desc: "so is a name that only starts with one",
			name: "'Ol' Buzzbark",
			want: "'Ol' Buzzbark",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := unquote(tt.name)
			if got != tt.want {
				t.Errorf("unquote(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}
