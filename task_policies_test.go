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

func (ts *TestSuite) Test_taskPolicies_invalid_params() {
	ts.SetupBackend()
	for _, ipd := range invalidParamData {
		req := ts.mkReq("task-policies", ipd.data)
		resp := ts.WithoutError(ts.HandleRequest(req)).(*logical.Response)
		ts.EqualError(resp.Error(), "missing or invalid "+ipd.field)
	}
}

func (ts *TestSuite) Test_taskPolicies_simple() {
	ts.SetupBackend()
	ts.Nil(ts.GetStored(tpKey("my-task")))

	req := ts.mkReq("task-policies", tpParams("my-task", "insurance"))

	resp := ts.HandleRequestSuccess(req)
	ts.Equal(resp, &logical.Response{})

	ts.StoredEqual(tpKey("my-task"), taskPolicies{[]string{"insurance"}})
}
