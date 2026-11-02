package matcher

import "testing"

// A hook rewrites Edition and Variation between two asks, and the answers
// have to follow the field as it stands, not the value first normalized.
func TestInputCardFollowsRewrittenFields(t *testing.T) {
	var card InputCard

	card.Edition = "Foo Promos"
	if !card.Contains("promos") {
		t.Errorf("edition %q does not contain promos", card.Edition)
	}
	card.Edition = "Bar"
	if card.Contains("promos") {
		t.Errorf("edition %q still contains promos", card.Edition)
	}
	if !card.Equals("bar") {
		t.Errorf("edition %q does not equal bar", card.Edition)
	}

	card.Variation = "Foo Promos"
	if !card.Contains("promos") {
		t.Errorf("variation %q does not contain promos", card.Variation)
	}
	card.Variation = "Baz"
	if card.Contains("promos") {
		t.Errorf("variation %q still contains promos", card.Variation)
	}
	if !card.Equals("baz") {
		t.Errorf("variation %q does not equal baz", card.Variation)
	}
}
