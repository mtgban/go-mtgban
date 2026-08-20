package magic

import (
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// The Has*Printing helpers answer whether any printing of a name carries a
// given Magic treatment, delegating to the core generic with the Magic
// vocabulary; they live here because the treatments do. The finish-based
// HasNonfoil/HasFoil/HasEtchedPrinting stay in core, since every game has
// finishes.

// HasExtendedArtPrinting reports whether the card was ever printed with
// extended art, narrowed to the given editions when any are named.
func HasExtendedArtPrinting(name string, editions ...string) bool {
	return mtgmatcher.HasPrinting(name, "frame_effect", FrameEffectExtendedArt, editions...)
}

// HasBorderlessPrinting reports whether the card was ever printed borderless,
// narrowed to the given editions when any are named.
func HasBorderlessPrinting(name string, editions ...string) bool {
	return mtgmatcher.HasPrinting(name, "border_color", BorderColorBorderless, editions...)
}

// HasShowcasePrinting reports whether the card was ever printed in a showcase
// frame, narrowed to the given editions when any are named.
func HasShowcasePrinting(name string, editions ...string) bool {
	return mtgmatcher.HasPrinting(name, "frame_effect", FrameEffectShowcase, editions...)
}

// HasPromoPackPrinting reports whether the card ever had a promo pack
// printing, narrowed to the given editions when any are named.
func HasPromoPackPrinting(name string, editions ...string) bool {
	return mtgmatcher.HasPrinting(name, "promo_type", PromoTypePromoPack, editions...)
}

// HasSerializedPrinting reports whether the card ever had a serialized
// printing, narrowed to the given editions when any are named.
func HasSerializedPrinting(name string, editions ...string) bool {
	return mtgmatcher.HasPrinting(name, "promo_type", PromoTypeSerialized, editions...)
}

// HasRetroFramePrinting reports whether the card was ever printed in the retro
// frame, narrowed to the given editions when any are named.
func HasRetroFramePrinting(name string, editions ...string) bool {
	return mtgmatcher.HasPrinting(name, "frame_version", "1997", editions...)
}
