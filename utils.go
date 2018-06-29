package mesosAuthPlugin

// This file contains random bits of code that don't really belong to any
// particular thing.

// jsonobj is an alias for type a JSON object gets unmarshalled into, because
// building nested map[string]interface{}{ ... } literals is awful.
type jsonobj = map[string]interface{}
