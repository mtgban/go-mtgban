package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestTreatmentLabel pins the priority a printing wearing more than one
// treatment resolves to, and that a printing wearing none of them gets no
// label at all.
func TestTreatmentLabel(t *testing.T) {
	for _, tt := range []struct {
		desc string
		co   mtgmatcher.CardObject
		want string
	}{
		{
			desc: "a plain printing carries no label",
			co:   mtgmatcher.CardObject{},
			want: "",
		},
		{
			desc: "showcase",
			co:   mtgmatcher.CardObject{Card: mtgmatcher.Card{FrameEffects: []string{FrameEffectShowcase}}},
			want: "Showcase",
		},
		{
			desc: "extended art",
			co:   mtgmatcher.CardObject{Card: mtgmatcher.Card{FrameEffects: []string{FrameEffectExtendedArt}}},
			want: "Extended Art",
		},
		{
			desc: "borderless",
			co:   mtgmatcher.CardObject{Card: mtgmatcher.Card{BorderColor: BorderColorBorderless}},
			want: "Borderless",
		},
		{
			desc: "retro frame",
			co:   mtgmatcher.CardObject{Card: mtgmatcher.Card{FrameVersion: "1997"}},
			want: "Retro Frame",
		},
		{
			desc: "showcase outranks a borderless printing that is also extended art",
			co: mtgmatcher.CardObject{Card: mtgmatcher.Card{
				FrameEffects: []string{FrameEffectShowcase, FrameEffectExtendedArt},
				BorderColor:  BorderColorBorderless,
			}},
			want: "Showcase",
		},
		{
			desc: "extended art outranks borderless",
			co: mtgmatcher.CardObject{Card: mtgmatcher.Card{
				FrameEffects: []string{FrameEffectExtendedArt},
				BorderColor:  BorderColorBorderless,
			}},
			want: "Extended Art",
		},
		{
			desc: "borderless outranks retro frame",
			co: mtgmatcher.CardObject{Card: mtgmatcher.Card{
				BorderColor:  BorderColorBorderless,
				FrameVersion: "1997",
			}},
			want: "Borderless",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := TreatmentLabel(&tt.co); got != tt.want {
				t.Errorf("TreatmentLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}
