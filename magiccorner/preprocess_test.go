package magiccorner

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

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

	b := &mtgmatcher.Backend{}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			_, _, got := internalPreprocess(b, tt.name, tt.edition, tt.variation, tt.extra)
			if got != tt.want {
				t.Errorf("variation = %q, want %q", got, tt.want)
			}
		})
	}
}

// Magic Corner's generic "Promo" shelf sells this card under two different
// physical listings and never tags either one with the "(V.N)" suffix its
// own DCI Promo shelf puts on the same listing, leaving the matcher a bare
// name shared by half a dozen printings - it lands on the Secret Lair Promo
// pair every time. The image name is the only place either listing still
// says which one it is.
func TestInternalPreprocessUginPromoShelf(t *testing.T) {
	tests := []struct {
		desc          string
		extra         string
		wantEdition   string
		wantVariation string
	}{
		{
			desc:          "the MagicFest 2025 foil, shared with the DCI Promo (V.2) listing",
			extra:         "ugin-the-spirit-dragon-v2_823824",
			wantEdition:   "MagicFest 2025",
			wantVariation: "6",
		},
		{
			desc:          "the Ugin's Fate nonfoil, named in Italian with no version at all",
			extra:         "ugin-lo-spirito-drago_271991",
			wantEdition:   "Ugin's Fate",
			wantVariation: "1",
		},
		{
			desc:          "no image to read leaves the shelf as ambiguous as it arrived",
			extra:         "noimage",
			wantEdition:   "Promo",
			wantVariation: "",
		},
	}

	b := &mtgmatcher.Backend{}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			_, gotEdition, gotVariation := internalPreprocess(b, "Ugin, the Spirit Dragon", "Promo", "", tt.extra)
			if gotEdition != tt.wantEdition || gotVariation != tt.wantVariation {
				t.Errorf("internalPreprocess() = edition %q, variation %q, want %q, %q",
					gotEdition, gotVariation, tt.wantEdition, tt.wantVariation)
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

// TestImageProductID pins the guards the image-carried id is resolved
// under, on rows copied verbatim from the datastore: same-name collapse,
// cross-set image reuse, a Phyrexian-language hit let through, and a
// same-set jpn twin resolving to the English printing.
func TestImageProductID(t *testing.T) {
	b := realDatastore(t)

	t.Run("the id picks the treatment an ambiguous wording could not", func(t *testing.T) {
		got := imageProductID(b, "Dominaria United", "/x/ertai-resurrected_672617.jpg", "Ertai Resurrected", "Dominaria United", "", false)
		if got == "" {
			t.Fatal("imageProductID returned no id, want DMU 298")
		}
		co, err := b.GetUUID(got)
		if err != nil || co.SetCode != "DMU" || co.Number != "298" {
			t.Errorf("imageProductID resolved to %v, want DMU 298", co)
		}
	})

	t.Run("a cross-set id is refused, deferring to the wording", func(t *testing.T) {
		got := imageProductID(b, "Secret Lair Drop Series", "/x/ertai-resurrected_672617.jpg", "Ertai Resurrected", "Secret Lair Drop Series", "", false)
		if got != "" {
			co, _ := b.GetUUID(got)
			t.Errorf("imageProductID = %v, want \"\" - the id names DMU 298, not a Secret Lair printing", co)
		}
	})

	t.Run("a Phyrexian-language hit is accepted for an English listing", func(t *testing.T) {
		got := imageProductID(b, "Phyrexia: All Will Be One: Extras",
			"/x/phyrexia-all-will-be-one-extras-jace-the-perfected-mind-v5-692910.jpg",
			"Jace, the Perfected Mind", "Phyrexia: All Will Be One: Extras", "V.5", true)
		if got == "" {
			t.Fatal("imageProductID returned no id, want ONE 429")
		}
		co, err := b.GetUUID(got)
		if err != nil || co.SetCode != "ONE" || co.Number != "429" || co.Language != "Phyrexian" {
			t.Errorf("imageProductID resolved to %v, want ONE 429 Phyrexian", co)
		}
	})

	t.Run("a same-set jpn twin resolves to the English printing", func(t *testing.T) {
		got := imageProductID(b, "Secret Lair Drop Series", "/x/the-royal-scions_791211.jpg", "The Royal Scions", "Secret Lair Drop Series", "", false)
		if got == "" {
			t.Fatal("imageProductID returned no id, want SLD 1600")
		}
		co, err := b.GetUUID(got)
		if err != nil || co.SetCode != "SLD" || co.Number != "1600" || co.Language != "English" {
			t.Errorf("imageProductID resolved to %v, want SLD 1600 English", co)
		}
	})
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
