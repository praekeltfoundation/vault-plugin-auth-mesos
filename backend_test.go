package mesosAuthPlugin

import (
	"context"

	"github.com/hashicorp/vault/logical"
)

// See helper_for_test.go for common infrastructure and tools.

// The purpose of the factory is to build backends.
func (ts *TestSuite) Test_Factory_builds_backend() {
	b, err := Factory(context.Background(), &logical.BackendConfig{})
	ts.NoError(err)
	ts.Equal(b.Type(), logical.TypeCredential)
	ts.Equal(b.SpecialPaths(), &logical.Paths{
		Unauthenticated: []string{"login"},
	})
}
