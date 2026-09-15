package mtgban

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// registryScraper is the scraper the registry tests build: it records what
// its constructor was handed and can be asked for one half.
type registryScraper struct {
	backend *mtgmatcher.Backend
	secret  string
	opts    Options
	config  ScraperOptions
}

func (registryScraper) Load(context.Context) error { return nil }
func (registryScraper) Info() ScraperInfo          { return ScraperInfo{Name: "Registry Test"} }

func (s *registryScraper) SetConfig(config ScraperOptions) {
	s.config = config
}

// registryPlain is a scraper that cannot be asked for one half.
type registryPlain struct{}

func (registryPlain) Load(context.Context) error { return nil }
func (registryPlain) Info() ScraperInfo          { return ScraperInfo{Name: "Plain"} }

func init() {
	Register("registry_test", []Game{GameMagic, GamePokemon},
		func(b *mtgmatcher.Backend, auth Authenticator, opts Options) (Scraper, error) {
			secret, err := auth.Secret("REGISTRY_TEST_SECRET")
			if err != nil {
				return nil, err
			}
			return &registryScraper{backend: b, secret: secret, opts: opts}, nil
		})
	Register("registry_plain", []Game{GameMagic},
		func(*mtgmatcher.Backend, Authenticator, Options) (Scraper, error) {
			return registryPlain{}, nil
		})
}

func TestNewScraperBuildsTheRegisteredScraper(t *testing.T) {
	b := &mtgmatcher.Backend{Game: "magic"}
	auth := MapAuthenticator{"REGISTRY_TEST_SECRET": "hunter2"}
	logged := false
	scraper, err := NewScraper(b, "registry_test", auth,
		WithLogCallback(func(string, ...any) { logged = true }),
		WithMaxConcurrency(3),
		WithAffiliate("aff"),
		WithBuylistAffiliate("bl"),
		WithTargetEdition("LEA"),
		WithResource("bridge", map[int]int{1: 2}),
	)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := scraper.(*registryScraper)
	if !ok {
		t.Fatalf("got %T", scraper)
	}
	if got.backend != b {
		t.Error("the constructor was not handed the datastore")
	}
	if got.secret != "hunter2" {
		t.Errorf("secret = %q", got.secret)
	}
	got.opts.LogCallback("")
	if !logged {
		t.Error("LogCallback was not passed through")
	}
	if got.opts.MaxConcurrency != 3 || got.opts.Affiliate != "aff" ||
		got.opts.BuylistAffiliate != "bl" || got.opts.TargetEdition != "LEA" {
		t.Errorf("options = %+v", got.opts)
	}
	bridge, err := Resource[map[int]int](got.opts, "bridge")
	if err != nil || bridge[1] != 2 {
		t.Errorf("bridge = %v, %v", bridge, err)
	}
	if got.config != (ScraperOptions{}) {
		t.Errorf("SetConfig was called with %+v for a scraper asked for both halves", got.config)
	}
}

func TestNewScraperRefusesWhatItCannotBuild(t *testing.T) {
	magic := &mtgmatcher.Backend{Game: "magic"}
	auth := MapAuthenticator{"REGISTRY_TEST_SECRET": "x"}
	cases := []struct {
		name    string
		backend *mtgmatcher.Backend
		scraper string
		auth    Authenticator
		opts    []Option
		want    string
	}{
		{"nil datastore", nil, "registry_test", auth, nil, "needs a datastore"},
		{"unnamed game", &mtgmatcher.Backend{}, "registry_test", auth, nil, "names no game"},
		{"unknown game", &mtgmatcher.Backend{Game: "chess"}, "registry_test", auth, nil, `"chess" is not one`},
		{"unknown scraper", magic, "nope", auth, nil, `no scraper "nope" for Magic (registered: registry_plain, registry_test)`},
		{"wrong game", &mtgmatcher.Backend{Game: "lorcana"}, "registry_test", auth, nil, `no scraper "registry_test" for Lorcana`},
		{"missing secret", magic, "registry_test", nil, nil, "registry_test: missing secret: REGISTRY_TEST_SECRET"},
		{"both halves", magic, "registry_test", auth, []Option{WithRetailOnly(), WithBuylistOnly()}, "leaves nothing to publish"},
		{"half of a plain scraper", magic, "registry_plain", nil, []Option{WithRetailOnly()}, "asked for its retail alone, but does not implement"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewScraper(tc.backend, tc.scraper, tc.auth, tc.opts...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestNewScraperConfiguresOneHalf(t *testing.T) {
	b := &mtgmatcher.Backend{Game: "pokemon"}
	auth := MapAuthenticator{"REGISTRY_TEST_SECRET": "x"}
	cases := []struct {
		name string
		half Option
		want ScraperOptions
	}{
		{"buylist only", WithBuylistOnly(), ScraperOptions{DisableRetail: true}},
		{"retail only", WithRetailOnly(), ScraperOptions{DisableBuylist: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scraper, err := NewScraper(b, "registry_test", auth, tc.half)
			if err != nil {
				t.Fatal(err)
			}
			got, ok := scraper.(*registryScraper)
			if !ok {
				t.Fatalf("got %T", scraper)
			}
			if got.config != tc.want {
				t.Errorf("config = %+v, want %+v", got.config, tc.want)
			}
		})
	}
}

func TestRegisteredListsAGameSortedAndRegisterRefusesTwice(t *testing.T) {
	names := Registered(GameMagic)
	if !strings.Contains(strings.Join(names, ","), "registry_plain,registry_test") {
		t.Errorf("Registered(Magic) = %v", names)
	}
	for _, name := range Registered(GamePokemon) {
		if name == "registry_plain" {
			t.Error("registry_plain was registered for Magic alone")
		}
	}
	defer func() {
		if recover() == nil {
			t.Error("registering a name twice for one game did not panic")
		}
	}()
	Register("registry_plain", []Game{GameMagic}, func(*mtgmatcher.Backend, Authenticator, Options) (Scraper, error) {
		return registryPlain{}, nil
	})
}

func TestGameOfFoldsCase(t *testing.T) {
	for _, game := range AllGames {
		got, err := GameOf(&mtgmatcher.Backend{Game: strings.ToLower(string(game))})
		if err != nil || got != game {
			t.Errorf("GameOf(%q) = %v, %v", strings.ToLower(string(game)), got, err)
		}
	}
}

func TestAuthenticators(t *testing.T) {
	t.Setenv("REGISTRY_TEST_ENV", "from-env")
	t.Setenv("REGISTRY_TEST_EMPTY", "")
	if got, err := (EnvAuthenticator{}).Secret("REGISTRY_TEST_ENV"); err != nil || got != "from-env" {
		t.Errorf("env secret = %q, %v", got, err)
	}
	_, err := (EnvAuthenticator{}).Secret("REGISTRY_TEST_EMPTY")
	if !errors.Is(err, ErrMissingSecret) || !strings.Contains(err.Error(), "REGISTRY_TEST_EMPTY env var") {
		t.Errorf("empty env secret err = %v", err)
	}
	m := MapAuthenticator{"a": "1", "empty": ""}
	if got, err := m.Secret("a"); err != nil || got != "1" {
		t.Errorf("map secret = %q, %v", got, err)
	}
	if _, err := m.Secret("empty"); !errors.Is(err, ErrMissingSecret) {
		t.Errorf("empty map secret err = %v", err)
	}
	if got, err := OptionalSecret(m, "absent"); err != nil || got != "" {
		t.Errorf("optional absent = %q, %v", got, err)
	}
	if got, err := OptionalSecret(nil, "absent"); err != nil || got != "" {
		t.Errorf("optional with nil auth = %q, %v", got, err)
	}
	if got, err := OptionalSecret(m, "a"); err != nil || got != "1" {
		t.Errorf("optional present = %q, %v", got, err)
	}
}

func TestResourceTypes(t *testing.T) {
	var opts Options
	WithResource("skus", map[int]string{1: "a"}).apply(&opts)
	if got, err := Resource[map[int]string](opts, "skus"); err != nil || got[1] != "a" {
		t.Errorf("typed resource = %v, %v", got, err)
	}
	if got, err := Resource[map[int]string](opts, "absent"); err != nil || got != nil {
		t.Errorf("absent resource = %v, %v", got, err)
	}
	_, err := Resource[[]string](opts, "skus")
	if err == nil || !strings.Contains(err.Error(), "resource skus is a map[int]string, not a []string") {
		t.Errorf("mismatched resource err = %v", err)
	}
}
