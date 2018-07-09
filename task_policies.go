package mesosauth

import (
	"context"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

// pathTaskPolicies returns the "task-policies" path struct. It is a function
// rather than a method because we never call it once the backend struct is
// built and we don't want name collisions with any request handler methods.
func pathTaskPolicies(b *mesosBackend) *framework.Path {
	return &framework.Path{
		Pattern: "task-policies",
		Fields: map[string]*framework.FieldSchema{
			"task-id-prefix": {Type: framework.TypeString},
			"policies":       {Type: framework.TypeCommaStringSlice},
		},
		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.UpdateOperation: b.pathTaskPoliciesUpdate,
		},
	}
}

// taskPolicies is used to store policies for a task.
type taskPolicies struct {
	Policies []string
}

// taskPolicies gives us a less verbose way to build a taskPolicies value.
func mkTaskPolicies(policies []string) taskPolicies {
	return taskPolicies{Policies: policies}
}

// tpKey builds a task policy storage key.
func tpKey(tip string) string {
	return "task-policies/" + tip
}

// pathTaskPoliciesUpdate is the "task-policies" update request handler.
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

	err := store(ctx, req.Storage, tpKey(taskIDPrefix), mkTaskPolicies(policies))
	return &logical.Response{}, err
}
