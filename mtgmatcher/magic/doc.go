// Package magic implements the Magic: the Gathering rules for the mtgmatcher
// card matcher — the edition/variant/promo data and the identification logic
// that core mtgmatcher dispatches through its GameRules hooks.
//
// magic imports core mtgmatcher for the Backend, InputCard, and Card types and
// the generic lookup helpers; core never imports magic. The dependency is
// one-directional, wired together at load time when the loader attaches the
// Magic GameRules to the Backend.
//
// Known limitation: the filter-callback tables (see callbacks.go) resolve a
// handful of auxiliary lookups through the package-level mtgmatcher helpers,
// which consult the global datastore — behavior inherited from the original
// in-core implementation. Magic rules therefore assume the Backend they serve
// is also installed via SetGlobalDatastore; a side Backend opened without
// being made global may answer those auxiliary lookups from the wrong data.
// Threading the Backend through the callback signatures would lift this.
package magic
