package mtgmatcher

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// Normalize caches whatever it is handed, and callers hand it search
// queries, so the two bounds that keep an arbitrary string from being
// retained have to hold.
func TestNormalizeCacheRejectsOversizedKeys(t *testing.T) {
	long := strings.Repeat("z", normalizeCacheMaxKey+1)

	before := normalizeCacheSize.Load()
	got := Normalize(long)
	after := normalizeCacheSize.Load()

	if after != before {
		t.Errorf("a %d byte key was cached, cap is %d", len(long), normalizeCacheMaxKey)
	}
	if _, found := normalizeCache.Load().Load(long); found {
		t.Error("the oversized key is in the map")
	}
	// Still correct, just uncached
	if got != Normalize(long) {
		t.Error("an uncached key normalized inconsistently")
	}
}

// A name of datastore length keeps being cached.
func TestNormalizeCacheKeepsRealNames(t *testing.T) {
	name := "_cache_test_ Okina, Temple to the Grandfathers"
	if len(name) > normalizeCacheMaxKey {
		t.Fatalf("test name is %d bytes, over the %d limit", len(name), normalizeCacheMaxKey)
	}
	Normalize(name)
	if _, found := normalizeCache.Load().Load(name); !found {
		t.Error("a datastore-length name was not cached")
	}
}

// Filling the cache must reset it rather than freeze it: a frozen cache
// never takes another entry, so a datastore reloaded later would run
// uncached for the rest of the process.
func TestNormalizeCacheResetsWhenFull(t *testing.T) {
	original := normalizeCache.Load()
	originalSize := normalizeCacheSize.Load()
	defer func() {
		normalizeCache.Store(original)
		normalizeCacheSize.Store(originalSize)
	}()

	normalizeCache.Store(&sync.Map{})
	normalizeCacheSize.Store(normalizeCacheCap)

	Normalize("_cache_test_ trigger the reset")

	if size := normalizeCacheSize.Load(); size >= normalizeCacheCap {
		t.Errorf("cache size is %d, expected a reset below %d", size, normalizeCacheCap)
	}
	if _, found := normalizeCache.Load().Load("_cache_test_ trigger the reset"); !found {
		t.Error("the call that tripped the reset did not get cached")
	}
}

// The cache must not change what Normalize answers, cached or not.
func TestNormalizeCacheAgreesWithUncached(t *testing.T) {
	uncached := func(str string) string {
		out := strings.TrimSpace(str)
		out = strings.ToLower(out)
		return replacer.Replace(out)
	}
	for _, name := range []string{
		"Okina, Temple to the Grandfathers", "  Lightning Bolt  ",
		"Teacher's Pest", "Cat Warriors", "Æther Vial", "Sword of Fire and Ice",
		strings.Repeat("y", normalizeCacheMaxKey+40),
	} {
		if got, want := Normalize(name), uncached(name); got != want {
			t.Errorf("Normalize(%q) = %q, uncached gives %q", name, got, want)
		}
		// second call takes the cached path where one exists
		if got, want := Normalize(name), uncached(name); got != want {
			t.Errorf("cached Normalize(%q) = %q, uncached gives %q", name, got, want)
		}
	}
}

// Concurrent use has to be safe, including across a reset.
func TestNormalizeCacheConcurrent(t *testing.T) {
	original := normalizeCache.Load()
	originalSize := normalizeCacheSize.Load()
	defer func() {
		normalizeCache.Store(original)
		normalizeCacheSize.Store(originalSize)
	}()
	normalizeCache.Store(&sync.Map{})
	normalizeCacheSize.Store(0)

	var wg sync.WaitGroup
	for w := range 8 {
		wg.Go(func() {
			for i := range 2000 {
				name := fmt.Sprintf("_cache_test_ card %d %d", w, i)
				if Normalize(name) == "" {
					t.Errorf("empty result for %q", name)
					return
				}
				if i%500 == 0 {
					normalizeCacheSize.Store(normalizeCacheCap)
				}
			}
		})
	}
	wg.Wait()
}
