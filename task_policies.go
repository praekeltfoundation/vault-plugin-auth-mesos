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
			logical.ReadOperation:   b.pathTaskPoliciesRead,
		},
	}
}

// taskPolicies is used to store policies for a task.
type taskPolicies struct {
	Policies []string
}

// mkTaskPolicies gives us a less verbose way to build a taskPolicies value.
func mkTaskPolicies(policies []string) taskPolicies {
	return taskPolicies{Policies: policies}
}

// tpKey builds a task policy storage key.
func tpKey(taskPrefix string) string {
	return "task-policies/" + taskPrefix
}

// pathTaskPoliciesUpdate is the "task-policies" update request handler.
func (b *mesosBackend) pathTaskPoliciesUpdate(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	rh := requestHelper{ctx: ctx, storage: req.Storage}

	taskIDPrefix := d.Get("task-id-prefix").(string)
	if len(taskIDPrefix) == 0 {
		return logical.ErrorResponse("missing or invalid task-id-prefix"), nil
	}

	policies := d.Get("policies").([]string)
	if len(policies) == 0 {
		return logical.ErrorResponse("missing or invalid policies"), nil
	}

	b.Logger().Info("TASK POLICIES", "task-id-prefix", taskIDPrefix, "policies", policies)

	err := rh.store(tpKey(taskIDPrefix), mkTaskPolicies(policies))
	return &logical.Response{}, err
}

// pathTaskPoliciesRead is the "task-policies" read request handler.
func (b *mesosBackend) pathTaskPoliciesRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	rh := requestHelper{ctx: ctx, storage: req.Storage}

	taskIDPrefix := d.Get("task-id-prefix").(string)
	if len(taskIDPrefix) == 0 {
		return logical.ErrorResponse("missing or invalid task-id-prefix"), nil
	}

	var tp taskPolicies
	decode := func(se *logical.StorageEntry) error {
		if se == nil {
			// Empty taskPolicies struct.
			return nil
		}
		return se.DecodeJSON(&tp)
	}
	err := rh.fetch(tpKey(taskIDPrefix), decode)
	// A fetch failure will leave us with a valid but empty taskPolicies value,
	// and any response we return alongside an error will be ignored.
	resp := &logical.Response{
		Data: jsonobj{
			"policies": tp.Policies,
		},
	}
	return resp, err
}
