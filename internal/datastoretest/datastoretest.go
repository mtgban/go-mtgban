// Package datastoretest opens a datastore for a test, keeping the unpacked
// file on disk so a suite fetches each one once rather than once per package.
//
// Ten packages load AllPrintings and every one of them is its own process, so
// the suite pays for the same file ten times: 37 seconds against the bucket
// against 15 against a copy already unpacked. The rest is JSON, which nothing
// here can help with.
//
// This is deliberately not part of internal/datastore. A scraper run wants
// the datastore the bucket holds now, and would be wrong to settle for what
// it held an hour ago; a test only wants the file to still be there.
package datastoretest

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/internal/datastore"
)

// ttl is how long an unpacked copy stands in for the bucket's. The datastores
// publish nightly, so an hour is short enough that a test never reasons from
// yesterday's file and long enough to cover a suite run many times over.
const ttl = time.Hour

// Env names the variable that turns the cache off, for a run that has to see
// exactly what the bucket holds.
const Env = "MTGBAN_TEST_CACHE"

// Open returns a reader over the datastore at path. A local path is opened
// where it lies; a remote one is unpacked into the cache directory and read
// from there, and re-fetched once the copy is older than ttl.
func Open(path string) (io.ReadCloser, error) {
	if path == "" {
		return nil, errors.New("no datastore path")
	}
	if !strings.Contains(path, "://") || os.Getenv(Env) == "off" {
		return datastore.Open(path)
	}

	name, err := cacheName(path)
	if err != nil {
		return datastore.Open(path)
	}
	if info, err := os.Stat(name); err == nil && time.Since(info.ModTime()) < ttl {
		if f, err := os.Open(name); err == nil {
			return f, nil
		}
	}
	if err := fetch(path, name); err != nil {
		// A cache that cannot be written is not a reason to fail a test.
		return datastore.Open(path)
	}
	return os.Open(name)
}

// cacheName is where the unpacked copy of a remote path lives. The bucket's
// own path is the key, flattened, so two games' files never collide.
func cacheName(path string) (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "mtgban-datastore")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	flat := strings.NewReplacer("://", "-", "/", "-").Replace(path)
	return filepath.Join(dir, strings.TrimSuffix(flat, ".xz")), nil
}

// fetch unpacks the bucket's copy into the cache, through a temporary file so
// that a suite whose packages run at once never reads a half-written one.
func fetch(path, name string) error {
	reader, err := datastore.Open(path)
	if err != nil {
		return err
	}
	defer reader.Close()

	tmp, err := os.CreateTemp(filepath.Dir(name), filepath.Base(name)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := io.Copy(tmp, reader); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), name)
}
