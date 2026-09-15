package mtgban

import "fmt"

// Options are the settings NewScraper collected from its Option arguments
// and hands to a scraper's constructor. A constructor applies the ones its
// scraper has a use for and ignores the rest: a store with no affiliate
// program has nowhere to put an affiliate code.
type Options struct {
	// Authenticator answers the secrets a scraper asks for, read through
	// Secret and OptionalSecret. Most scrapers ask for none and a caller
	// building one of those sets no authenticator.
	Authenticator Authenticator

	// LogCallback receives the scraper's progress messages.
	LogCallback LogCallbackFunc

	// MaxConcurrency caps the requests in flight where the scraper makes
	// several at once. Zero keeps the scraper's own default.
	MaxConcurrency int

	// Affiliate is the partner code the scraper stamps on the links it
	// publishes. BuylistAffiliate is the one for its buylist links, for the
	// scraper whose buylist links go to a different store.
	Affiliate        string
	BuylistAffiliate string

	// TargetEdition restricts the scraper to one edition, for a run that
	// wants a single set priced quickly.
	TargetEdition string

	// DisableRetail and DisableBuylist ask for one half of a store that
	// publishes both. NewScraper refuses both at once, and refuses either
	// for a scraper that cannot honour it.
	DisableRetail  bool
	DisableBuylist bool

	resources map[string]any
}

// Option configures a scraper NewScraper builds. The options here are the
// ones every scraper understands; a scraper package wraps WithResource in a
// typed option for each input only it reads.
type Option interface {
	apply(*Options)
}

type optionFunc func(*Options)

func (f optionFunc) apply(o *Options) {
	f(o)
}

// WithAuthenticator hands over the secrets the scraper asks for by name. A
// scraper that asks for none needs no authenticator.
func WithAuthenticator(auth Authenticator) Option {
	return optionFunc(func(o *Options) {
		o.Authenticator = auth
	})
}

// WithLogCallback sends the scraper's progress messages to fn.
func WithLogCallback(fn LogCallbackFunc) Option {
	return optionFunc(func(o *Options) {
		o.LogCallback = fn
	})
}

// WithMaxConcurrency caps the requests the scraper keeps in flight.
func WithMaxConcurrency(n int) Option {
	return optionFunc(func(o *Options) {
		o.MaxConcurrency = n
	})
}

// WithAffiliate stamps the partner code on the links the scraper publishes.
func WithAffiliate(code string) Option {
	return optionFunc(func(o *Options) {
		o.Affiliate = code
	})
}

// WithBuylistAffiliate stamps the partner code on the scraper's buylist
// links, where those go to a store of their own.
func WithBuylistAffiliate(code string) Option {
	return optionFunc(func(o *Options) {
		o.BuylistAffiliate = code
	})
}

// WithTargetEdition restricts the scraper to one edition.
func WithTargetEdition(edition string) Option {
	return optionFunc(func(o *Options) {
		o.TargetEdition = edition
	})
}

// WithRetailOnly asks for the store's retail listings and not its buylist.
func WithRetailOnly() Option {
	return optionFunc(func(o *Options) {
		o.DisableBuylist = true
	})
}

// WithBuylistOnly asks for the store's buylist and not its retail listings.
func WithBuylistOnly() Option {
	return optionFunc(func(o *Options) {
		o.DisableRetail = true
	})
}

// WithResource hands the scraper an input it prices from beyond the
// datastore - a published catalog, a SKU list, an id bridge between two
// marketplaces - under the name the scraper documents. The scraper packages
// wrap this in a typed option per input, which is what a caller should
// reach for; the constructor reads it back with Resource and refuses a
// value of the wrong type.
func WithResource(name string, value any) Option {
	return optionFunc(func(o *Options) {
		if o.resources == nil {
			o.resources = map[string]any{}
		}
		o.resources[name] = value
	})
}

// Resource reads the named resource out of the options as a T: the zero T
// where none was given, and an error naming both types where one was given
// of another type.
func Resource[T any](o Options, name string) (T, error) {
	var zero T
	value, found := o.resources[name]
	if !found {
		return zero, nil
	}
	typed, ok := value.(T)
	if !ok {
		return zero, fmt.Errorf("resource %s is a %T, not a %T", name, value, zero)
	}
	return typed, nil
}
