// Package magic implements the Magic: the Gathering rules for the mtgmatcher
// card matcher — the edition/variant/promo data and the identification logic
// that core mtgmatcher dispatches through its GameRules hooks.
//
// magic imports core mtgmatcher for the Backend, InputCard, and Card types and
// the generic lookup helpers; core never imports magic. The dependency is
// one-directional, wired together at load time when the loader attaches the
// Magic GameRules to the Backend.
//
// The identification path takes the Backend it serves and reads nothing
// else: the filter callbacks and the promo tag functions are handed it
// along with the card, so a Backend opened on the side is matched against
// its own data whether or not it was installed via SetGlobalDatastore. The
// package's exported Has*Printing helpers (wrappers.go) and its
// token-pairing helpers (tokenpairs.go) take a Backend for the same reason -
// they exist for the scrapers, which hand over the one they were built on -
// and the pairing index the latter read is that Backend's own, built by the
// loader that minted the pairings.
package magic
