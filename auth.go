package mesosAuthPlugin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

// TODO: Replace this with calls to mesos.
var temporarySetOfExistingTasks = map[string]bool{}

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
	if !b.verifyTaskExists(taskID) {
		return nil, logical.ErrPermissionDenied
	}

	prefix, err := taskIDPrefix(taskID)
	if err != nil {
		return nil, logical.ErrPermissionDenied
	}

	b.Logger().Info("LOGIN", "task-id", taskID, "prefix", prefix, "RemoteAddr", req.Connection.RemoteAddr)

	policies, err := getTaskPolicies(ctx, req.Storage, prefix)
	if err != nil {
		return nil, err
	}

	// TODO: Make the renewal period configurable?
	return &logical.Response{
		Auth: &logical.Auth{
			Policies: policies,
			Period:   10 * time.Minute,
			LeaseOptions: logical.LeaseOptions{
				Renewable: true,
			},
			// Stash task-id so we can check it again for renewals.
			InternalData: jsonobj{"task-id": taskID},
		},
	}, nil
}

// authRenew is the renew callback for tokens created by this plugin.
func (b *mesosBackend) authRenew(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	b.Logger().Info("RENEW", "auth", fmt.Sprintf("%#v", req.Auth))

	taskID := req.Auth.InternalData["task-id"].(string)
	if !b.verifyTaskExists(taskID) {
		return nil, fmt.Errorf("task %s not found during renewal", taskID)
	}

	// For a standard periodic renewal, we only need to return the Auth struct
	// we're given in the request.
	return &logical.Response{Auth: req.Auth}, nil
}

func (b *mesosBackend) verifyTaskExists(taskID string) bool {
	if taskID == "" {
		return false
	}

	b.Logger().Debug("TODO: Check task in mesos.")
	// TODO: Verify that the task exists.
	return temporarySetOfExistingTasks[taskID]
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
	err = se.DecodeJSON(&tp)
	return tp.Policies, err
}

func taskIDPrefix(taskID string) (string, error) {
	idx := strings.LastIndex(taskID, ".")
	if idx < 1 {
		// We have no task prefix (no dot or nothing before the last dot).
		return "", fmt.Errorf("malformed task-id")
	}
	return taskID[0:idx], nil
}
