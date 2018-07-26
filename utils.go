package mesosauth

// This file contains random bits of code that don't really belong to any
// particular thing.

import (
	"context"
	"errors"

	"github.com/hashicorp/vault/logical"
)

// jsonobj is an alias for the type a JSON object gets unmarshalled into,
// because building nested map[string]interface{}{ ... } literals is awful.
type jsonobj = map[string]interface{}

// requestHelper stores a bunch of request information and provides methods
// that use it to reduce parameter passing boilerplate. Some methods are
// defined in the files that use them.
//
// This is a violation of The Rule About Contexts, but requestHelper is used
// exclusively for requests-scoped storage operations that are intended to
// share the same context and would be buried in noise if the context and
// storage had to be passed to each function individually.
type requestHelper struct {
	ctx     context.Context
	storage logical.Storage
}

// store is a helper function to construct and store a Vault storage entry so
// we can avoid boilerplate in all the places we do this.
func (rh *requestHelper) store(key string, value interface{}) error {
	storageEntry, err := logical.StorageEntryJSON(key, value)
	if err == nil {
		err = rh.storage.Put(rh.ctx, storageEntry)
	}
	return err
}

// fetch is a helper function to fetch a Vault storage entry. To get around
// Go's poor support for building abstractions, it takes a callback for
// decoding the fetched value (which may be nil).
func (rh *requestHelper) fetch(key string, decode func(*logical.StorageEntry) error) error {
	se, err := rh.storage.Get(rh.ctx, key)
	if err == nil {
		err = decode(se)
	}
	return err
}

// getConfig fetches the plugin config from Vault, returning nil if there is no
// config.
func (rh *requestHelper) getConfigOrNil() (*config, error) {
	var cfg *config
	err := rh.fetch("config", func(se *logical.StorageEntry) error {
		if se == nil {
			return nil
		}
		cfg = &config{}
		return se.DecodeJSON(cfg)
	})
	return cfg, err
}

// getConfig fetches the plugin config from Vault, returning an error if there
// is no config.
func (rh *requestHelper) getConfig() (*config, error) {
	cfg, err := rh.getConfigOrNil()
	if cfg == nil && err == nil {
		err = errors.New("backend not configured")
	}
	return cfg, err
}
