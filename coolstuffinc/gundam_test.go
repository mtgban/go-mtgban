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
	} {
		gotName, gotVariation := gundamCard(tt.name, tt.number)
		if gotName != tt.wantName || gotVariation != tt.want {
			t.Errorf("gundamCard(%q, %q)\n got  %q %q\n want %q %q",
				tt.name, tt.number, gotName, gotVariation, tt.wantName, tt.want)
		}
	}
}
