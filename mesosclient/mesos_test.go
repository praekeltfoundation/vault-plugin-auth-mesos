package mesosclient

import (
	"context"
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
	fm := mesostest.NewFakeMesos()
	ts.AddCleanup(fm.Close)
	client := NewClient(fm.GetBaseURL() + "/bad")

	_, err := client.GetTasks(context.Background())
	ts.Error(err)
	ts.Contains(err.Error(), "404 page not found")
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

// getResp is a wrapper around all the type and error juggling noise.
func (ts *MesosClientTests) getTasks(client *Client) *master.Response_GetTasks {
	return ts.WithoutError(client.GetTasks(context.Background())).(*master.Response_GetTasks)
}
