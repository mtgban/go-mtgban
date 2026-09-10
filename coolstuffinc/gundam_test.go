package coolstuffinc

import "testing"

func TestGundamShelf(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		// The code narrows nothing the catalog knows, and leaving it on
		// leaves the shelf naming no set at all.
		{"GD01 - Newtype Rising", "Newtype Rising"},
		{"ST09 - Destiny Ignition", "Destiny Ignition"},
		{"EB01 - Eternal Nexus", "Eternal Nexus"},
		// A shelf carrying no code is the storefront's own and stays whole.
		{"Beta", "Beta"},
		{"Tokens and Misc", "Tokens and Misc"},
	} {
		if got := gundamShelf(tt.in); got != tt.want {
			t.Errorf("gundamShelf(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestGundamCard(t *testing.T) {
	for _, tt := range []struct {
		name, number   string
		wantName, want string
	}{
		// The storefront repeats the number inside the name, where it says
		// nothing the field does not.
		{"A Show of Resolve (GD01-100)", "GD01-100", "A Show of Resolve", "GD01-100"},
		// The wording behind the name picks between the printings sharing
		// the number, so it rides into the variation.
		{"Battle of Aces (GD01-111) (Alt-Art +)", "GD01-111", "Battle of Aces", "GD01-111 Alt-Art +"},
		// Only one comes off: a card whose own name ends in a parenthetical
		// wears two, and taking both asks for a card the catalog has not.
		{"Unicorn Gundam (Destroy Mode) (GD01-002) (SP)", "GD01-002", "Unicorn Gundam (Destroy Mode)", "GD01-002 SP"},
		{"GQuuuuuuX (Omega Psycommu) (GD02-038)", "GD02-038", "GQuuuuuuX (Omega Psycommu)", "GD02-038"},
		// The token shelf names the art the token wears; the number says
		// which of the two it is.
		{"Aile Strike Gundam (T-008)", "T-008", "Aile Strike Gundam", "T-008 Token"},
		// The sell listing has no number field, so the one written inside
		// the name is the only one there is - and it still has to reach the
		// token gate.
		{"A Show of Resolve (GD01-100)", "", "A Show of Resolve", "GD01-100"},
		{"Battle of Aces (GD01-111) (Alt-Art +)", "", "Battle of Aces", "GD01-111 Alt-Art +"},
		{"Aile Strike Gundam (T-008)", "", "Aile Strike Gundam", "T-008 Token"},
	} {
		gotName, gotVariation := gundamCard(tt.name, tt.number)
		if gotName != tt.wantName || gotVariation != tt.want {
			t.Errorf("gundamCard(%q, %q)\n got  %q %q\n want %q %q",
				tt.name, tt.number, gotName, gotVariation, tt.wantName, tt.want)
		}
	}
}

func TestGundamName(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		// The four names this storefront types its own way, one letter or
		// one word off the catalog.
		{"Adbul's Maganac", "Abdul's Maganac"},
		{"Tiffa Adill & Freedom", "Tiffa Adill & Freeden"},
		// A glyph the storefront names in brackets and the catalog reads
		// straight through.
		{"Xi (Symbol) Gundam", "Xi Gundam"},
		{"(Turn A Symbol) Gundam", "Turn A Gundam"},
		{"Xi (Symbol) Gundam (Flight Form)", "Xi Gundam (Flight Form)"},
		// Both halves of a card printing two, whose numbers the catalog
		// keeps behind the joined name.
		{"Guncannon (108) & Guncannon (109)", "Guncannon & Guncannon (108) (109)"},
		{"Core Booster (005) & Core Booster (006)", "Core Booster & Core Booster (005) (006)"},
		// A name carrying a parenthetical of its own is left alone.
		{"Unicorn Gundam (Destroy Mode)", "Unicorn Gundam (Destroy Mode)"},
		{"GQuuuuuuX (Omega Psycommu)", "GQuuuuuuX (Omega Psycommu)"},
	} {
		if got := gundamName(tt.in); got != tt.want {
			t.Errorf("gundamName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestGundamNumberSpelling(t *testing.T) {
	for _, tt := range []struct{ number, name, want string }{
		// Every number this game prints is a run code and three digits, so
		// a dropped digit and an extra zero are both the storefront's own
		// typing.
		{"GD02-57", "Zedas (GD02-57)", "GD02-057"},
		{"EXBP-0013", "EX Base (EXBP-013) (Promo)", "EXBP-013"},
		{"GD01-100", "A Show of Resolve (GD01-100)", "GD01-100"},
		// A parallel lettered onto the number the catalog letters nothing,
		// where the name spells the same number plain.
		{"R-008A", "Resource (R-008) (Alt-Art +)", "R-008"},
		// And a lettered number the name does not spell plain is left as
		// written rather than guessed at.
		{"R-008A", "Resource (Alt-Art +)", "R-008A"},
		{"", "", ""},
	} {
		if got := gundamNumberSpelling(tt.number, tt.name); got != tt.want {
			t.Errorf("gundamNumberSpelling(%q, %q) = %q, want %q", tt.number, tt.name, got, tt.want)
		}
	}
}

func TestGundamTokenName(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		// The catalog writes the word before the qualifier, not after it.
		{"GQuuuuuuX (Omega Psycommu)", "GQuuuuuuX Token (Omega Psycommu)"},
		// A token whose name carries no qualifier is reached by the word
		// alone, which Match adds for itself.
		{"Char's Zaku II", "Char's Zaku II"},
	} {
		if got := gundamTokenName(tt.in); got != tt.want {
			t.Errorf("gundamTokenName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestGundamTier(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		// The storefront abbreviates the suffix the catalog writes out, and
		// the suffix is what tells a parallel run from the printing it
		// parallels at the same number.
		{"CP", "C+"},
		{"LGRPP", "LR++"},
		{"U", "Uncommon"},
		// A rarity both spell alike passes through whole.
		{"Legend Rare", "Legend Rare"},
		{"", ""},
	} {
		if got := gundamTier(tt.in); got != tt.want {
			t.Errorf("gundamTier(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestGundamNumber(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		// The notes spell the number in full where the name drops a digit.
		{"GD03-072", "GD03-072"},
		{"Alt-Art + GD01-111", "GD01-111"},
		// The token shelf spends the same field on the art the token wears.
		{"Gundam Age-1 Normal", ""},
		{"", ""},
	} {
		if got := gundamNumber(tt.in); got != tt.want {
			t.Errorf("gundamNumber(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
