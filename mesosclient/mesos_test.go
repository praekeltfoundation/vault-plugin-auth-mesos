package mesosclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	mesos "github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/master"
	"github.com/stretchr/testify/suite"

	mesostest "github.com/praekeltfoundation/vault-plugin-auth-mesos/mesosclient/testing"
	"github.com/praekeltfoundation/vault-plugin-auth-mesos/testutils"
)

// MesosClientTests is a testify test suite object that we can attach helper
// methods to.
type MesosClientTests struct{ testutils.TestSuite }

// Test_MesosClient is a standard Go test function that runs our test suite's
// tests.
func Test_MesosClient(t *testing.T) { suite.Run(t, new(MesosClientTests)) }

// mkTask builds a simple task value.
func mkTask(name, id string, state mesos.TaskState) mesos.Task {
	return mesos.Task{
		Name:   name,
		TaskID: mesos.TaskID{Value: id},
		State:  &state,
	}
}

// We get an error if the client isn't talking to a Mesos API.
func (ts *MesosClientTests) Test_bad_server() {
	srv := httptest.NewServer(http.HandlerFunc(http.NotFound))
	ts.AddCleanup(srv.Close)
	client := NewClient(srv.URL)

	_, err := client.GetTasks(context.Background())
	ts.Error(err)
	ts.Contains(err.Error(), "404 page not found")
}

// We get an error if the client gets a malformed response.
func (ts *MesosClientTests) Test_bad_response() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Claim to return protobuf data, but actually return garbage instead.
		w.Header().Set("Content-Type", "application/x-protobuf")
		_, _ = w.Write([]byte("this is not a protobuf")) // #nosec G104
	})
	srv := httptest.NewServer(handler)
	ts.AddCleanup(srv.Close)
	client := NewClient(srv.URL)

	_, err := client.GetTasks(context.Background())
	ts.Error(err)
	ts.Contains(err.Error(), "proto: Response:")
}

// We can get the tasks even if there are none.
func (ts *MesosClientTests) Test_GetTasks_no_tasks() {
	fm := mesostest.NewFakeMesos()
	ts.AddCleanup(fm.Close)
	client := NewClient(fm.GetBaseURL())

	rgt := ts.getTasks(client)
	ts.Equal(rgt, &master.Response_GetTasks{})
}

// We can get the tasks if tasks exist.
func (ts *MesosClientTests) Test_GetTasks_some_tasks() {
	fm := mesostest.NewFakeMesos()
	ts.AddCleanup(fm.Close)
	client := NewClient(fm.GetBaseURL())

	task := mkTask("task", "abc-123", mesos.TASK_RUNNING)
	fm.AddTask(task)

	rgt := ts.getTasks(client)
	ts.Equal(rgt, &master.Response_GetTasks{Tasks: []mesos.Task{task}})
}

// We can make a successful request with a redirect.
func (ts *MesosClientTests) Test_GetTasks_redirect() {
	// Where we want to end up.
	fm := mesostest.NewFakeMesos()
	ts.AddCleanup(fm.Close)
	redirURL := fm.GetBaseURL()[5:len(fm.GetBaseURL())]
	// Where we start.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirURL+"/api/v1", http.StatusTemporaryRedirect)
	}))
	ts.AddCleanup(srv.Close)
	client := NewClient(srv.URL)

	rgt := ts.getTasks(client)
	ts.Equal(rgt, &master.Response_GetTasks{})
}

// We can make a successful request with a redirect.
func (ts *MesosClientTests) Test_GetTasks_redirect_withscheme() {
	// Where we want to end up.
	fm := mesostest.NewFakeMesos()
	ts.AddCleanup(fm.Close)
	// Where we start.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, fm.GetBaseURL()+"/api/v1", http.StatusTemporaryRedirect)
	}))
	ts.AddCleanup(srv.Close)
	client := NewClient(srv.URL)

	rgt := ts.getTasks(client)
	ts.Equal(rgt, &master.Response_GetTasks{})
}

// We eventually fail in a redirect loop.
func (ts *MesosClientTests) Test_GetTasks_too_many_redirects() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirURL := fmt.Sprintf("//%s%s", r.Host, r.URL)
		http.Redirect(w, r, redirURL, http.StatusTemporaryRedirect)
	}))
	ts.AddCleanup(srv.Close)
	client := NewClient(srv.URL)

	_, err := client.GetTasks(context.Background())
	ts.Error(err)
	ts.Contains(err.Error(), "too many redirects")
}

// getResp is a wrapper around all the type and error juggling noise.
func (ts *MesosClientTests) getTasks(client *Client) *master.Response_GetTasks {
	return ts.WithoutError(client.GetTasks(context.Background())).(*master.Response_GetTasks)
}
