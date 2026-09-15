package mtgban

import (
	"errors"
	"fmt"
)

// ErrMissingSecret is what an Authenticator answers for a secret it does not
// hold, wrapped with the name of the one that is missing.
var ErrMissingSecret = errors.New("missing secret")

// Authenticator hands a scraper the secrets it asks for by name: an API key,
// an app token, a signature. A scraper package names the secrets it needs as
// constants, spelled the way the environment variables that have always
// carried them are, so a caller reading the environment passes them through
// unchanged and a test hands a MapAuthenticator the same names. Where the
// secrets come from is the caller's business: this package reads no
// environment of its own.
type Authenticator interface {
	// Secret returns the named secret. One the authenticator does not hold
	// is an error wrapping ErrMissingSecret, and the scraper asking decides
	// whether it can do without it.
	Secret(name string) (string, error)
}

// MapAuthenticator holds secrets by name, for a test or for a caller that
// read them from a store of its own. An absent or empty entry is missing.
type MapAuthenticator map[string]string

// Secret implements Authenticator.
func (m MapAuthenticator) Secret(name string) (string, error) {
	value := m[name]
	if value == "" {
		return "", fmt.Errorf("%w: %s", ErrMissingSecret, name)
	}
	return value, nil
}

// Secret asks the options' authenticator for a secret the scraper cannot do
// without. Options carrying no authenticator hold nothing, and answer the
// way one lacking the secret does, so a scraper fails the same way whether
// its caller gave no authenticator or one without the secret.
func (o Options) Secret(name string) (string, error) {
	if o.Authenticator == nil {
		return "", fmt.Errorf("%w: %s", ErrMissingSecret, name)
	}
	return o.Authenticator.Secret(name)
}
