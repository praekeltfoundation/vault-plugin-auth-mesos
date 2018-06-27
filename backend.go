package mesosAuthPlugin

import (
	"context"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

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
		},
		Invalidate: b.invalidate,
		Clean:      b.cleanup,
	}

	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}
	return &b, nil
}

// TODO: Make this useful or get rid of it.
func (b *mesosBackend) invalidate(_ context.Context, wtf string) {
	b.Logger().Info("INVALIDATE", "wtf", wtf)
}

// TODO: Make this useful or get rid of it.
func (b *mesosBackend) cleanup(_ context.Context) {
	b.Logger().Info("CLEANUP")
}
