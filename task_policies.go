package mesosAuthPlugin

import (
	"context"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

// pathTaskPolicies (the function) returns the "task-policies" path struct. It
// is a function rather than a method because we never call it once the backend
// struct is built and we don't want name collisions with any request handler
// methods.
func pathTaskPolicies(b *mesosBackend) *framework.Path {
	return &framework.Path{
		Pattern: "task-policies",
		Fields: map[string]*framework.FieldSchema{
			"task-id-prefix": &framework.FieldSchema{
				Type: framework.TypeString,
			},
			"policies": &framework.FieldSchema{
				Type: framework.TypeCommaStringSlice,
			},
		},
		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.UpdateOperation: b.pathTaskPoliciesUpdate,
		},
	}
}

type taskPolicies struct {
	Policies []string
}

func tpKey(tip string) string {
	return "task-policies/" + tip
}

// pathTaskPoliciesUpdate (the method) is the "task-policies" update request
// handler.
func (b *mesosBackend) pathTaskPoliciesUpdate(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {

	taskIDPrefix := d.Get("task-id-prefix").(string)
	if len(taskIDPrefix) == 0 {
		return logical.ErrorResponse("missing or invalid task-id-prefix"), nil
	}

	policies := d.Get("policies").([]string)
	if len(policies) == 0 {
		return logical.ErrorResponse("missing or invalid policies"), nil
	}

	b.Logger().Info("TASK POLICIES", "task-id-prefix", taskIDPrefix, "policies", policies)

	storageEntry, err := logical.StorageEntryJSON(tpKey(taskIDPrefix), taskPolicies{policies})
	if err != nil {
		// noqa: (Not actually a tag that does anything, sadly.)
		return nil, err
	}

	if err := req.Storage.Put(ctx, storageEntry); err != nil {
		// noqa: (Not actually a tag that does anything, sadly.)
		return nil, err
	}

	return &logical.Response{}, nil
}
