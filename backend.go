package mesosauth

import (
	"context"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

// mesosBackend is our plugin backend object.
type mesosBackend struct {
	*framework.Backend
}

// Factory builds a plugin backend.
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	var b mesosBackend

	b.Backend = &framework.Backend{
		BackendType: logical.TypeCredential,
		AuthRenew:   b.authRenew,
		PathsSpecial: &logical.Paths{
			Unauthenticated: []string{"login"},
		},
		Paths: []*framework.Path{
			pathLogin(&b),
			pathTaskPolicies(&b),
		},
		Invalidate: b.invalidate,
		Clean:      b.cleanup,
	}

	// We unconditionally return &b and whatever error we got from the setup
	// call to avoid some useless error handler boilerplate that we can't test.
	// (Let's hope the caller doesn't assume an error response will always
	// accompany a nil backend.)
	err := b.Setup(ctx, conf)
	return &b, err
}

// TODO: Make this useful or get rid of it.
func (b *mesosBackend) invalidate(_ context.Context, wtf string) {
	b.Logger().Info("INVALIDATE", "wtf", wtf)
}

// TODO: Make this useful or get rid of it.
func (b *mesosBackend) cleanup(_ context.Context) {
	b.Logger().Info("CLEANUP")
}
