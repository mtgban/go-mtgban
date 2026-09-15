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
	"encoding/json"
	"io"
	"os"

	"github.com/mtgban/simplecloud"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// concurrentDownloads matches what the scraping tools ask B2 for: the
// datastores are tens of megabytes and arrive in ranged parts.
const concurrentDownloads = 20

// Payload reads a datastore and hands back the document a decoder should
// see: the value of "data" where the file is the {"meta":...,"data":...}
// envelope the builders publish, and the file itself where it is not.
//
// Loaders unwrap this for themselves. It is here for the suites that read a
// published datastore without going through one - they decode the document's
// own top level, and would see an empty payload the day the envelope lands.
func Payload(r io.Reader) ([]byte, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	if envelope.Data != nil {
		return envelope.Data, nil
	}
	return raw, nil
}

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

// Read opens the datastore the path names and reads it as the game's,
// handing back the backend without installing it. The game has to be
// registered by the binary, the way its own package's blank import does.
func Read(game, path string) (*mtgmatcher.Backend, error) {
	reader, err := Open(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return mtgmatcher.Open(game, reader)
}
