package mesosAuthPlugin

import (
	"github.com/hashicorp/vault/logical"
)

// See helper_for_test.go for common infrastructure and tools.

var invalidParamData = []struct {
	field string
	data  jsonobj
}{
	{"task-id-prefix", jsonobj{}},
	{"task-id-prefix", jsonobj{"policies": "insurance"}},
	{"task-id-prefix", jsonobj{"task-id-prefix": ""}},
	{"policies", jsonobj{"task-id-prefix": "my-task"}},
	{"policies", jsonobj{"task-id-prefix": "my-task", "policies": ""}},
	{"policies", jsonobj{"task-id-prefix": "my-task", "policies": []string{}}},
}

// Any missing or empty parameter causes a task-policies update request to
// fail.
func (ts *TestSuite) Test_taskPolicies_invalid_params() {
	ts.SetupBackend()
	for _, ipd := range invalidParamData {
		req := ts.mkReq("task-policies", ipd.data)
		resp := ts.HandleRequest(req)
		ts.EqualError(resp.Error(), "missing or invalid "+ipd.field)
	}
}

// A task-policies update containing a single policy succeeds.
func (ts *TestSuite) Test_taskPolicies_simple() {
	ts.SetupBackend()
	ts.Nil(ts.GetStored(tpKey("my-task")))

	req := ts.mkReq("task-policies", tpParams("my-task", "insurance"))

	resp := ts.HandleRequest(req)
	ts.Equal(resp, &logical.Response{})

	ts.StoredEqual(tpKey("my-task"), taskPolicies{[]string{"insurance"}})
}

// A task-policies update overwrites any existing policies for that task.
func (ts *TestSuite) Test_taskPolicies_replace() {
	ts.SetupBackend()
	ts.Nil(ts.GetStored(tpKey("my-task")))

	req1 := ts.mkReq("task-policies", tpParams("my-task", "insurance"))
	ts.Equal(ts.HandleRequest(req1), &logical.Response{})
	ts.StoredEqual(tpKey("my-task"), taskPolicies{[]string{"insurance"}})

	req2 := ts.mkReq("task-policies", tpParams("my-task", "foreign"))
	ts.Equal(ts.HandleRequest(req2), &logical.Response{})
	ts.StoredEqual(tpKey("my-task"), taskPolicies{[]string{"foreign"}})
}
