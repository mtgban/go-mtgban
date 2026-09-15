package mtgban

import (
	"errors"
	"fmt"
	"os"
)

// ErrMissingSecret is what an Authenticator answers for a secret it does not
// hold, wrapped with the name of the one that is missing.
var ErrMissingSecret = errors.New("missing secret")

// Authenticator hands a scraper the secrets it asks for by name: an API key,
// an app token, a signature. A scraper package names the secrets it needs as
// constants, spelled the way the environment variables that have always
// carried them are, so EnvAuthenticator reads them unchanged and a test
// hands a MapAuthenticator the same names.
type Authenticator interface {
	// Secret returns the named secret. One the authenticator does not hold
	// is an error wrapping ErrMissingSecret, and the scraper asking decides
	// whether it can do without: see Options.OptionalSecret.
	Secret(name string) (string, error)
}

// EnvAuthenticator reads each secret from the environment variable of the
// same name. An unset or empty variable is a missing secret.
type EnvAuthenticator struct{}

// Secret implements Authenticator.
func (EnvAuthenticator) Secret(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("%w: %s env var", ErrMissingSecret, name)
	}
	return value, nil
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

// OptionalSecret asks for a secret the scraper works without: a missing one
// comes back empty with no error, and any other failure as it was.
func (o Options) OptionalSecret(name string) (string, error) {
	value, err := o.Secret(name)
	if errors.Is(err, ErrMissingSecret) {
		return "", nil
	}
	return value, err
}
