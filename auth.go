package mesosAuthPlugin

import (
	"context"
	"fmt"
	"strings"
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
	if !verifyTaskExists(taskID) {
		return nil, logical.ErrPermissionDenied
	}

	prefix := taskIDPrefix(taskID)

	b.Logger().Info("LOGIN", "task-id", taskID, "prefix", prefix, "RemoteAddr", req.Connection.RemoteAddr)

	policies, err := getTaskPolicies(ctx, req.Storage, prefix)
	if err != nil {
		return nil, err
	}

	// TODO: Validate the task-id and look up the associated list of policies.
	// TODO: Make the renewal period configurable?
	return &logical.Response{
		Auth: &logical.Auth{
			Policies: policies,
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

func verifyTaskExists(taskID string) bool {
	if taskID == "" {
		return false
	}

	// TODO: Verify that the task exists.
	return true
}

func getTaskPolicies(ctx context.Context, storage logical.Storage, taskPrefix string) ([]string, error) {
	se, err := storage.Get(ctx, tpKey(taskPrefix))
	if err != nil {
		// noqa: (Not actually a tag that does anything, sadly.)
		return nil, err
	}
	if se == nil {
		return nil, logical.ErrPermissionDenied
	}

	var tp taskPolicies
	if err := se.DecodeJSON(&tp); err != nil {
		return nil, err
	}
	return tp.Policies, nil
}

func taskIDPrefix(taskID string) string {
	idx := strings.LastIndex(taskID, ".")
	if idx < 1 {
		// We have no task prefix (either no dot or nothing before the last
		// dot), so return the whole taskID.
		return taskID
	}
	return taskID[0:idx]
}
