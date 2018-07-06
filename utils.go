package mesosAuthPlugin

// This file contains random bits of code that don't really belong to any
// particular thing.

import (
	"context"

	"github.com/hashicorp/vault/logical"
)

// jsonobj is an alias for type a JSON object gets unmarshalled into, because
// building nested map[string]interface{}{ ... } literals is awful.
type jsonobj = map[string]interface{}

// store is a helper function to construct and store a Vault storage entry so
// we can avoid boilerplate in all the places we do this.
func store(ctx context.Context, storage logical.Storage, key string, value interface{}) error {
	storageEntry, err := logical.StorageEntryJSON(key, value)
	if err == nil {
		err = storage.Put(ctx, storageEntry)
	}
	return err
}
