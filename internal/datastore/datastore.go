// Package datastore opens a game's datastore wherever its path points.
//
// The paths the suites read out of the environment name three different
// things. CI downloads each datastore and hands over a file; a developer
// may point at the bucket the production datastores are published to
// ("b2://mtgban-datastore/lorcana/lorcana.json.xz"), which is what the
// scraping tools are configured with; and a host serving one over http
// answers as well. simplecloud reads all of those and decompresses an .xz
// on the way, so a caller only has to hand the path over as written.
package datastore

import (
	"context"
	"io"
	"os"

	"github.com/mtgban/simplecloud"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// concurrentDownloads matches what the scraping tools ask B2 for: the
// datastores are tens of megabytes and arrive in ranged parts.
const concurrentDownloads = 20

// Open returns a reader over the datastore the path names. The caller
// closes it. Credentials come from the environment, which simplecloud
// deliberately does not read on its own.
func Open(path string) (io.ReadCloser, error) {
	id, key := b2Credentials()
	return simplecloud.Open(context.Background(), path,
		simplecloud.WithB2Credentials(id, key),
		simplecloud.WithConcurrentDownloads(concurrentDownloads))
}

// b2Credentials reads the key pair under either name it is published as.
// The workflows export the plain pair, having read the datastore secrets
// into it; a local .env carries the same two values under the names that
// say which bucket they open, since one file holds the keys to several.
func b2Credentials() (id, key string) {
	id, key = os.Getenv("B2_APPLICATION_KEY_ID"), os.Getenv("B2_APPLICATION_KEY")
	if id == "" && key == "" {
		id, key = os.Getenv("B2_APPLICATION_KEY_ID_DATASTORE"), os.Getenv("B2_APPLICATION_KEY_DATASTORE")
	}
	return id, key
}

// Load opens the datastore the path names and installs it as the global
// one. It is mtgmatcher.LoadDatastoreFile with the path read the way Open
// reads it, for the suites that match through the package-level API.
func Load(path string) error {
	reader, err := Open(path)
	if err != nil {
		return err
	}
	defer reader.Close()
	return mtgmatcher.LoadDatastore(reader)
}
