package mesosAuthPlugin

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
		resp, err := ts.HandleRequest(req)
		ts.NoError(err)
		ts.ResponseError(resp, "missing or invalid "+ipd.field)
	}
}
