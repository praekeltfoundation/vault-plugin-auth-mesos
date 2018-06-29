package mesosAuthPlugin

import (
	"time"

	"github.com/hashicorp/vault/logical"
)

// See helper_for_test.go for common infrastructure and tools.

func (ts *TestSuite) Test_login_no_taskID() {
	ts.SetupBackend()
	req := ts.mkReq("login", jsonobj{})

	resp, err := ts.HandleRequest(req)
	ts.EqualError(err, "permission denied")
	ts.Nil(resp)
}

func (ts *TestSuite) Test_login_unregistered_taskID() {
	ts.SetupBackend()
	req := ts.mkReq("login", jsonobj{"task-id": "unregistered-task.abc-123"})

	resp, err := ts.HandleRequest(req)
	ts.EqualError(err, "permission denied")
	ts.Nil(resp)
}

func (ts *TestSuite) Test_login_good_taskID() {
	ts.SetupBackend()
	ts.SetTaskPolicies("task-that-exists", "insurance")

	req := ts.mkReq("login", jsonobj{"task-id": "task-that-exists.abc-123"})

	resp, err := ts.HandleRequest(req)
	ts.Require().NoError(err)
	ts.Nil(resp.Warnings)
	ts.Nil(resp.Secret)
	ts.Equal(resp.Auth, &logical.Auth{
		Policies:     []string{"insurance"},
		Period:       10 * time.Minute,
		LeaseOptions: logical.LeaseOptions{Renewable: true},
	})
}

func (ts *TestSuite) Test_renewal_not_logged_in() {
	ts.SetupBackend()

	req := &logical.Request{
		Operation:       "renew",
		Path:            "login",
		Auth:            nil,
		Unauthenticated: false,
	}

	resp, err := ts.HandleRequest(req)
	ts.EqualError(err, "request has no secret")
	ts.Nil(resp)
}

func (ts *TestSuite) Test_renewal_logged_in() {
	ts.SetupBackend()
	ts.SetTaskPolicies("logged-in-task", "foreign")
	auth := ts.Login("logged-in-task.abc-123")

	req := &logical.Request{
		Operation:       "renew",
		Path:            "login",
		Auth:            auth,
		Unauthenticated: false,
	}

	resp, err := ts.HandleRequest(req)
	ts.NoError(err)
	ts.Equal(auth, resp.Auth)
}
