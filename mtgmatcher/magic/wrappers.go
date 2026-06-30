package magic

import (
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// The Has*Printing helpers answer whether any printing of a name carries a
// given Magic treatment, delegating to the core generic with the Magic
// vocabulary; they live here because the treatments do. The finish-based
// HasNonfoil/HasFoil/HasEtchedPrinting stay in core, since every game has
// finishes.

func HasExtendedArtPrinting(name string, editions ...string) bool {
	return mtgmatcher.HasPrinting(name, "frame_effect", FrameEffectExtendedArt, editions...)
}

func HasBorderlessPrinting(name string, editions ...string) bool {
	return mtgmatcher.HasPrinting(name, "border_color", BorderColorBorderless, editions...)
}

func HasShowcasePrinting(name string, editions ...string) bool {
	return mtgmatcher.HasPrinting(name, "frame_effect", FrameEffectShowcase, editions...)
}

func HasPromoPackPrinting(name string, editions ...string) bool {
	return mtgmatcher.HasPrinting(name, "promo_type", PromoTypePromoPack, editions...)
}

func HasSerializedPrinting(name string, editions ...string) bool {
	return mtgmatcher.HasPrinting(name, "promo_type", PromoTypeSerialized, editions...)
}

func HasRetroFramePrinting(name string, editions ...string) bool {
	return mtgmatcher.HasPrinting(name, "frame_version", "1997", editions...)
}
