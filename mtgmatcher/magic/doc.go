// Package magic implements the Magic: the Gathering rules for the mtgmatcher
// card matcher — the edition/variant/promo data and the identification logic
// that core mtgmatcher dispatches through its GameRules hooks.
//
// magic imports core mtgmatcher for the Backend, InputCard, and Card types and
// the generic lookup helpers; core never imports magic. The dependency is
// one-directional, wired together at load time when the loader attaches the
// Magic GameRules to the Backend.
package magic
