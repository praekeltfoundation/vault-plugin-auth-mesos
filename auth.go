package mesosauth

import (
	"context"
	"fmt"
	"strings"

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
			"task-id": {Type: framework.TypeString},
		},
		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.UpdateOperation: b.pathLogin,
		},
	}
}

// taskInstances is used to track taskIDS that have logged in.
type taskInstances struct {
	TaskIDs map[string]bool
}

// tiKey builds a task instances storage key.
func tiKey(taskPrefix string) string {
	return "task-instances/" + taskPrefix
}

// pathLogin (the method) is the "login" path request handler.
func (b *mesosBackend) pathLogin(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	rh := requestHelper{ctx: ctx, storage: req.Storage}

	cfg, err := rh.getConfig()
	if err != nil {
		return nil, err
	}

	taskID := d.Get("task-id").(string)
	if !b.verifyTaskExists(taskID) {
		return nil, logical.ErrPermissionDenied
	}

	prefix, err := taskIDPrefix(taskID)
	if err != nil {
		return nil, logical.ErrPermissionDenied
	}

	b.Logger().Info("LOGIN", "task-id", taskID, "prefix", prefix, "RemoteAddr", req.Connection.RemoteAddr)

	policies, err := rh.getTaskPolicies(prefix)
	if err != nil {
		return nil, err
	}

	// TODO: Clean out stale entries.
	if err := rh.verifyTaskNotLoggedIn(taskID, prefix); err != nil {
		return nil, err
	}

	// TODO: Make the renewal period configurable?
	return &logical.Response{
		Auth: &logical.Auth{
			Policies: policies,
			Period:   cfg.TTL,
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
	rh := requestHelper{ctx: ctx, storage: req.Storage}

	cfg, err := rh.getConfig()
	if err != nil {
		return nil, err
	}

	b.Logger().Info("RENEW", "auth", fmt.Sprintf("%#v", req.Auth))

	taskID := req.Auth.InternalData["task-id"].(string)
	if !b.verifyTaskExists(taskID) {
		return nil, fmt.Errorf("task %s not found during renewal", taskID)
	}

	// We make a (shallow) copy of the Auth struct from the request so that we
	// can update the renewal period (in case the config has changed since last
	// time) without modifying the request data.
	auth := *req.Auth
	auth.Period = cfg.TTL

	return &logical.Response{Auth: &auth}, nil
}

// verifyTaskExists checks that a taskID is valid and identifies an existing
// task.
func (b *mesosBackend) verifyTaskExists(taskID string) bool {
	if taskID == "" {
		return false
	}

	b.Logger().Debug("TODO: Check task in mesos.")
	// TODO: Verify that the task exists.
	return temporarySetOfExistingTasks[taskID]
}

// verifyTaskNotLoggedIn checks that a taskID is not already logged in and
// marks it as logged in for next time.
func (rh *requestHelper) verifyTaskNotLoggedIn(taskID string, prefix string) error {
	instances, err := rh.getTaskInstances(prefix)
	if err != nil {
		// noqa: (Not actually a tag that does anything, sadly.)
		return err
	}
	if instances[taskID] {
		// This task has already logged in.
		return logical.ErrPermissionDenied
	}

	instances[taskID] = true
	return rh.store(tiKey(prefix), taskInstances{TaskIDs: instances})
}

// getTaskPolicies fetches the policies for a taskID prefix.
func (rh *requestHelper) getTaskPolicies(taskPrefix string) ([]string, error) {
	var tp taskPolicies
	decode := func(se *logical.StorageEntry) error {
		if se == nil {
			return logical.ErrPermissionDenied
		}
		return se.DecodeJSON(&tp)
	}
	err := rh.fetch(tpKey(taskPrefix), decode)
	return tp.Policies, err
}

// getTaskInstances fetches the policies for a taskID prefix.
func (rh *requestHelper) getTaskInstances(taskPrefix string) (map[string]bool, error) {
	var ti taskInstances
	decode := func(se *logical.StorageEntry) error {
		if se == nil {
			ti.TaskIDs = map[string]bool{}
			return nil
		}
		return se.DecodeJSON(&ti)
	}
	err := rh.fetch(tiKey(taskPrefix), decode)
	return ti.TaskIDs, err
}

// taskIDPrefix extracts the prefix from a taskID.
func taskIDPrefix(taskID string) (string, error) {
	idx := strings.LastIndex(taskID, ".")
	if idx < 1 {
		// We have no task prefix (-1) or an empty task prefix (0).
		return "", fmt.Errorf("malformed task-id: \"%s\"", taskID)
	}
	return taskID[0:idx], nil
}
