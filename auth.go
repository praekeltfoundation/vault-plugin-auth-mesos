package mesosAuthPlugin

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

// pathLogin (the function) returns the "login" path struct. It is a function
// rather than a method because we never call it once the backend struct is
// built and we don't want name collisions with any request handler methods.
func pathLogin(b *mesosBackend) *framework.Path {
	return &framework.Path{
		Pattern: "login",
		Fields: map[string]*framework.FieldSchema{
			"task-id": &framework.FieldSchema{
				Type: framework.TypeString,
			},
		},
		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.UpdateOperation: b.pathLogin,
		},
	}
}

// pathLogin (the method) is the "login" path request handler.
func (b *mesosBackend) pathLogin(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {

	taskID := d.Get("task-id").(string)
	if len(taskID) < 1 {
		return nil, logical.ErrPermissionDenied
	}

	b.Logger().Info("LOGIN", "task-id", taskID, "RemoteAddr", req.Connection.RemoteAddr)

	// TODO: Validate the task-id and look up the associated list of policies.
	// TODO: Make the renewal period configurable?
	return &logical.Response{
		Auth: &logical.Auth{
			Policies: []string{},
			Period:   10 * time.Minute,
			LeaseOptions: logical.LeaseOptions{
				Renewable: true,
			},
		},
	}, nil
}

// authRenew is the renew callback for tokens created by this plugin.
func (b *mesosBackend) authRenew(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	b.Logger().Info("RENEW", "req", fmt.Sprintf("%#v", req))
	// For an unconditional renewal, we only need to return the Auth struct
	// we're given in the request.
	return &logical.Response{Auth: req.Auth}, nil
}
