package mesosauth

import (
	"testing"
	"time"

	"github.com/hashicorp/vault/logical"
	mesos "github.com/mesos/mesos-go/api/v1/lib"
	"github.com/stretchr/testify/suite"

	mctesting "github.com/praekeltfoundation/vault-plugin-auth-mesos/mesosclient/testing"
)

// See helper_for_test.go for common infrastructure and tools.

// AuthTests is a testify test suite object that we can attach helper methods
// to.
type AuthTests struct{ TestSuite }

// Test_Auth is a standard Go test function that runs our test suite's tests.
func Test_Auth(t *testing.T) { suite.Run(t, new(AuthTests)) }

//////////////////////
// Tests for login. //
//////////////////////

// Can't log in without a taskID.
func (ts *AuthTests) Test_login_no_taskID() {
	ts.SetupBackendWithMesos()
	req := ts.mkReq("login", jsonobj{})
	ts.HandleRequestError(req, "permission denied")
}

// Can't log in with a taskID that doesn't exist.
func (ts *AuthTests) Test_login_missing_taskID() {
	ts.SetupBackendWithMesos()
	ts.SetTaskPolicies("missing-task", "insurance")
	req := ts.mkReq("login", jsonobj{"task-id": "missing-task.abc-123"})
	ts.HandleRequestError(req, "permission denied")
}

// Can't log in with a taskID that doesn't have a Marathon-style app prefix.
func (ts *AuthTests) Test_login_taskID_with_no_prefix() {
	ts.SetupBackendWithMesos()
	ts.AddTask(mkTask("noprefix", "abc-123", mesos.TASK_RUNNING))
	req := ts.mkReq("login", jsonobj{"task-id": "abc-123"})
	ts.HandleRequestError(req, "permission denied")
}

// Can't log in with a taskID that doesn't have policies configured for its
// prefix.
func (ts *AuthTests) Test_login_unregistered_taskID() {
	ts.SetupBackendWithMesos()
	ts.AddTask(mkTask("unregistered", "unregistered-task.abc-123", mesos.TASK_RUNNING))
	req := ts.mkReq("login", jsonobj{"task-id": "unregistered-task.abc-123"})
	ts.HandleRequestError(req, "permission denied")
}

// Can log in with a taskID that exists and has policies configured for its
// prefix.
func (ts *AuthTests) Test_login_good_taskID() {
	ts.SetupBackendWithMesos()
	ts.AddTask(mkTask("existing", "task-that-exists.abc-123", mesos.TASK_RUNNING))
	ts.SetTaskPolicies("task-that-exists", "insurance")

	req := ts.mkReq("login", jsonobj{"task-id": "task-that-exists.abc-123"})

	resp := ts.HandleRequest(req)
	ts.Nil(resp.Warnings)
	ts.Nil(resp.Secret)
	ts.Equal(resp.Auth, &logical.Auth{
		Policies:     []string{"insurance"},
		Period:       10 * time.Minute,
		LeaseOptions: logical.LeaseOptions{Renewable: true},
		InternalData: jsonobj{"task-id": "task-that-exists.abc-123"},
	})
}

// Can't log in more than once with the same taskID.
func (ts *AuthTests) Test_login_only_once() {
	ts.SetupBackendWithMesos()
	ts.AddTask(mkTask("mine", "my-task.abc-123", mesos.TASK_RUNNING))
	ts.SetTaskPolicies("my-task", "insurance")

	auth := ts.Login("my-task.abc-123")
	ts.Equal(auth.Policies, []string{"insurance"})

	req := ts.mkReq("login", jsonobj{"task-id": "my-task.abc-123"})
	ts.HandleRequestError(req, "permission denied")
}

// A task that is not yet running can't log in.
func (ts *AuthTests) Test_login_staging_task() {
	ts.SetupBackendWithMesos()
	ts.AddTask(mkTask("staging", "staging-task.abc-123", mesos.TASK_STAGING))
	ts.SetTaskPolicies("staging-task", "insurance")
	req := ts.mkReq("login", jsonobj{"task-id": "unregistered-task.abc-123"})
	ts.HandleRequestError(req, "permission denied")
}

// A failed task can't log in.
func (ts *AuthTests) Test_login_failed_task() {
	ts.SetupBackendWithMesos()
	ts.AddTask(mkTask("failed", "failed-task.abc-123", mesos.TASK_FAILED))
	ts.SetTaskPolicies("failed-task", "insurance")
	req := ts.mkReq("login", jsonobj{"task-id": "unregistered-task.abc-123"})
	ts.HandleRequestError(req, "permission denied")
}

// Token renewal period is configurable.
func (ts *AuthTests) Test_login_period_configurable() {
	ts.SetupBackendWithMesos()
	ts.HandleRequestSuccess(ts.mkReq("config", jsonobj{"period": "420s"}))
	ts.AddTask(mkTask("task", "task.abc-123", mesos.TASK_RUNNING))
	ts.SetTaskPolicies("task", "insurance")

	resp := ts.HandleRequest(ts.mkReq("login", jsonobj{"task-id": "task.abc-123"}))
	ts.Equal(resp.Auth, &logical.Auth{
		Policies:     []string{"insurance"},
		Period:       7 * time.Minute,
		LeaseOptions: logical.LeaseOptions{Renewable: true},
		InternalData: jsonobj{"task-id": "task.abc-123"},
	})
}

// Can't log in with an unconfigured backend or bad Mesos.
func (ts *AuthTests) Test_login_unconfigured_or_bad() {
	ts.SetupBackend()
	ts.HandleRequestError(ts.mkReq("login", jsonobj{}), "backend not configured")

	ts.SetTaskPolicies("task", "insurance")
	ts.ConfigureBackend("ftp://bad")
	errmsg := `Post ftp://bad/api/v1: unsupported protocol scheme "ftp"`
	ts.HandleRequestError(ts.mkReq("login", jsonobj{"task-id": "task.abc-123"}), errmsg)
}

////////////////////////
// Tests for renewal. //
////////////////////////

// mkRenew builds a renewal request.
func (ts *AuthTests) mkRenew(auth *logical.Auth) *logical.Request {
	return &logical.Request{
		Operation:       "renew",
		Path:            "login",
		Auth:            auth,
		Unauthenticated: false,
		Storage:         ts.storage,
		Connection:      &logical.Connection{},
	}
}

// Can't renew if you're not logged in.
func (ts *AuthTests) Test_renewal_not_logged_in() {
	ts.SetupBackendWithMesos()

	ts.HandleRequestError(ts.mkRenew(nil), "request has no secret")
}

// Can't renew if you somehow have no taskID.
func (ts *AuthTests) Test_renewal_no_taskID() {
	ts.SetupBackendWithMesos()
	ts.AddTask(mkTask("logged-in", "logged-in-task.abc-123", mesos.TASK_RUNNING))
	ts.SetTaskPolicies("logged-in-task", "foreign")
	auth := ts.Login("logged-in-task.abc-123")

	auth.InternalData["task-id"] = ""
	ts.HandleRequestError(ts.mkRenew(auth), "missing task-id")
	auth.InternalData = nil
	ts.HandleRequestError(ts.mkRenew(auth), "missing task-id")
}

// Can renew if you are logged in and your task still exists.
func (ts *AuthTests) Test_renewal_logged_in() {
	ts.SetupBackendWithMesos()
	ts.AddTask(mkTask("logged-in", "logged-in-task.abc-123", mesos.TASK_RUNNING))
	ts.SetTaskPolicies("logged-in-task", "foreign")
	auth := ts.Login("logged-in-task.abc-123")

	resp := ts.HandleRequest(ts.mkRenew(auth))
	ts.Equal(auth, resp.Auth)
}

// Can't renew if your task is finished.
func (ts *AuthTests) Test_renewal_task_ended() {
	ts.SetupBackendWithMesos()
	ts.AddTask(mkTask("short", "short-task.abc-123", mesos.TASK_RUNNING))
	ts.SetTaskPolicies("short-task", "foreign")
	auth := ts.Login("short-task.abc-123")
	ts.UpdateTask(mctesting.UpdateState(mesos.TASK_FINISHED), "short-task.abc-123")

	ts.HandleRequestError(ts.mkRenew(auth), "task short-task.abc-123 not found during renewal")
}

// Renewal period is configurable between login and renewal.
func (ts *AuthTests) Test_renewal_period_configurable() {
	ts.SetupBackendWithMesos()
	ts.AddTask(mkTask("task", "task.abc-123", mesos.TASK_RUNNING))
	ts.SetTaskPolicies("task", "insurance")
	auth := ts.Login("task.abc-123")
	ts.Equal(auth.Period, 10*time.Minute)

	ts.HandleRequestSuccess(ts.mkReq("config", jsonobj{"period": "420s"}))

	resp := ts.HandleRequest(ts.mkRenew(auth))
	// The renewal auth has the updated period.
	ts.Equal(resp.Auth.Period, 7*time.Minute)
	// The original login auth is unmodified.
	ts.Equal(auth.Period, 10*time.Minute)
}

// Can't renew with an unconfigured backend or bad Mesos.
func (ts *AuthTests) Test_renewal_unconfigured_or_bad() {
	ts.SetupBackendWithMesos()
	ts.AddTask(mkTask("logged-in", "logged-in-task.abc-123", mesos.TASK_RUNNING))
	ts.SetTaskPolicies("logged-in-task", "foreign")
	auth := ts.Login("logged-in-task.abc-123")

	ts.DeleteStored("config")
	ts.HandleRequestError(ts.mkRenew(auth), "backend not configured")

	ts.ConfigureBackend("ftp://bad")
	errmsg := `Post ftp://bad/api/v1: unsupported protocol scheme "ftp"`
	ts.HandleRequestError(ts.mkRenew(auth), errmsg)
}

// mkTask builds a simple task value.
func mkTask(name, id string, state mesos.TaskState) mesos.Task {
	return mesos.Task{
		Name:   name,
		TaskID: mesos.TaskID{Value: id},
		State:  &state,
	}
}
