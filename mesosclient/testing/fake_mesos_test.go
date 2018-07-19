package testing

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	mesos "github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/master"
	"github.com/stretchr/testify/suite"

	"github.com/praekeltfoundation/vault-plugin-auth-mesos/testutils"
)

// FakeMesosTests is a testify test suite object that we can attach helper
// methods to.
type FakeMesosTests struct{ testutils.TestSuite }

// Test_FakeMesos is a standard Go test function that runs our test suite's
// tests.
func Test_FakeMesos(t *testing.T) { suite.Run(t, new(FakeMesosTests)) }

// mkTask builds a simple task value.
func mkTask(name, id string, state mesos.TaskState) mesos.Task {
	return mesos.Task{
		Name:   name,
		TaskID: mesos.TaskID{Value: id},
		State:  &state,
	}
}

// We can add a new task to FakeMesos.
func (ts *FakeMesosTests) Test_AddTask_new() {
	fm := NewFakeMesos()
	ts.AddCleanup(fm.Close)

	task := mkTask("task", "abc-123", mesos.TASK_RUNNING)

	ts.Equal(fm.tasks, map[string]mesos.Task{})
	fm.AddTask(task)
	ts.Equal(fm.tasks, map[string]mesos.Task{"abc-123": task})
}

// We can add many new tasks to FakeMesos.
func (ts *FakeMesosTests) Test_AddTask_multiple() {
	fm := NewFakeMesos()
	ts.AddCleanup(fm.Close)

	task1 := mkTask("task", "abc-123", mesos.TASK_RUNNING)
	task2 := mkTask("task", "abc-124", mesos.TASK_RUNNING)

	ts.Equal(fm.tasks, map[string]mesos.Task{})
	fm.AddTask(task1, task2)
	ts.Equal(fm.tasks, map[string]mesos.Task{"abc-123": task1, "abc-124": task2})
}

// We can't add duplicate tasks to FakeMesos.
func (ts *FakeMesosTests) Test_AddTask_duplicate() {
	fm := NewFakeMesos()
	ts.AddCleanup(fm.Close)

	task1 := mkTask("task1", "abc-123", mesos.TASK_STAGING)
	task1dup := mkTask("task1", "abc-123", mesos.TASK_RUNNING)
	task2 := mkTask("task1", "abc-124", mesos.TASK_RUNNING)

	ts.Equal(fm.tasks, map[string]mesos.Task{})

	ts.Panics(func() { fm.AddTask(task1, task1dup, task2) })
	// We successfully added the first task, but not the duplicate or the
	// second task.
	ts.Equal(fm.tasks, map[string]mesos.Task{"abc-123": task1})
}

// Without tasks to get, we get no tasks.
func (ts *FakeMesosTests) Test_getTasks_empty() {
	fm := NewFakeMesos()
	ts.AddCleanup(fm.Close)

	ts.Equal(*fm.getTasks(), master.Response_GetTasks{})
}

// Staging and running tasks belong in the Tasks field. (We don't bother
// testing all of these.)
func (ts *FakeMesosTests) Test_getTasks_staging_running() {
	fm := NewFakeMesos()
	ts.AddCleanup(fm.Close)

	task1 := mkTask("task1", "abc-123", mesos.TASK_STAGING)
	task2 := mkTask("task2", "abc-124", mesos.TASK_RUNNING)
	fm.AddTask(task1, task2)

	getTasks := fm.getTasks()
	// The order of the tasks is arbitrary, so we can't just assert on the
	// whole collection.
	ts.ElementsMatch(getTasks.Tasks, []mesos.Task{task1, task2})
	// All other fields should be empty.
	ts.Equal(*getTasks, master.Response_GetTasks{Tasks: getTasks.Tasks})
}

// Finished and failed tasks belong in the CompletedTasks field. (We don't
// bother testing all of these.)
func (ts *FakeMesosTests) Test_getTasks_finished_failed() {
	fm := NewFakeMesos()
	ts.AddCleanup(fm.Close)

	task1 := mkTask("task1", "abc-123", mesos.TASK_FINISHED)
	task2 := mkTask("task2", "abc-124", mesos.TASK_FAILED)
	fm.AddTask(task1, task2)

	getTasks := fm.getTasks()
	// The order of the tasks is arbitrary, so we can't just assert on the
	// whole collection.
	ts.ElementsMatch(getTasks.CompletedTasks, []mesos.Task{task1, task2})
	// All other fields should be empty.
	ts.Equal(*getTasks, master.Response_GetTasks{CompletedTasks: getTasks.CompletedTasks})
}

// Lost and unreachable tasks belong in the UnreachableTasks field. (We don't
// bother testing all of these.)
func (ts *FakeMesosTests) Test_getTasks_lost_unreachable() {
	fm := NewFakeMesos()
	ts.AddCleanup(fm.Close)

	task1 := mkTask("task1", "abc-123", mesos.TASK_LOST)
	task2 := mkTask("task2", "abc-124", mesos.TASK_UNREACHABLE)
	fm.AddTask(task1, task2)

	getTasks := fm.getTasks()
	// The order of the tasks is arbitrary, so we can't just assert on the
	// whole collection.
	ts.ElementsMatch(getTasks.UnreachableTasks, []mesos.Task{task1, task2})
	// All other fields should be empty.
	ts.Equal(*getTasks, master.Response_GetTasks{UnreachableTasks: getTasks.UnreachableTasks})
}

// Non-API paths return 404.
func (ts *FakeMesosTests) Test_API_bad_path() {
	fm := NewFakeMesos()
	ts.AddCleanup(fm.Close)

	resp := ts.WithoutError(http.Get(fm.GetBaseURL() + "/phpmyadmin")).(*http.Response)
	ts.Equal(resp.StatusCode, 404)
}

// Non-POST requests return 405.
func (ts *FakeMesosTests) Test_API_non_POST() {
	fm := NewFakeMesos()
	ts.AddCleanup(fm.Close)

	resp := ts.WithoutError(http.Get(fm.GetAPIURL())).(*http.Response)
	ts.Equal(resp.StatusCode, 405)
}

// Non-protobuf requests return 415.
func (ts *FakeMesosTests) Test_API_bad_content_type() {
	fm := NewFakeMesos()
	ts.AddCleanup(fm.Close)

	resp := ts.getResp(http.Post(fm.GetAPIURL(), "image/png", strings.NewReader("not a png")))
	ts.Equal(resp.StatusCode, 415)
}

// Non-call request payloads return 400.
func (ts *FakeMesosTests) Test_API_bad_request_payload() {
	fm := NewFakeMesos()
	ts.AddCleanup(fm.Close)

	body := strings.NewReader("not a protobuf")
	resp := ts.getResp(http.Post(fm.GetAPIURL(), "application/x-protobuf", body))
	ts.Equal(resp.StatusCode, 400)
}

// Unimplemented requests types also return 400.
func (ts *FakeMesosTests) Test_API_unimplemented_request() {
	fm := NewFakeMesos()
	ts.AddCleanup(fm.Close)

	resp := ts.postAPI(fm.GetAPIURL(), master.Call_GET_METRICS)
	ts.Equal(resp.StatusCode, 400)
}

// We can get the tasks even if there are none.
func (ts *FakeMesosTests) Test_API_GET_TASKS_no_tasks() {
	fm := NewFakeMesos()
	ts.AddCleanup(fm.Close)

	resp := ts.postAPI(fm.GetAPIURL(), master.Call_GET_TASKS)
	ts.Equal(resp.StatusCode, 200)

	var respData master.Response
	respBytes := ts.WithoutError(ioutil.ReadAll(resp.Body)).([]byte)
	ts.NoError(respData.Unmarshal(respBytes))
	ts.Equal(respData, master.Response{
		Type:     master.Response_GET_TASKS,
		GetTasks: &master.Response_GetTasks{},
	})
}

// We can get the tasks if tasks exist.
func (ts *FakeMesosTests) Test_API_GET_TASKS_some_tasks() {
	fm := NewFakeMesos()
	ts.AddCleanup(fm.Close)

	task := mkTask("task", "abc-123", mesos.TASK_RUNNING)
	fm.AddTask(task)

	resp := ts.postAPI(fm.GetAPIURL(), master.Call_GET_TASKS)
	ts.Equal(resp.StatusCode, 200)

	var respData master.Response
	respBytes := ts.WithoutError(ioutil.ReadAll(resp.Body)).([]byte)
	ts.NoError(respData.Unmarshal(respBytes))
	ts.Equal(respData, master.Response{
		Type: master.Response_GET_TASKS,
		GetTasks: &master.Response_GetTasks{
			Tasks: []mesos.Task{task},
		},
	})
}

// getResp is a type signature hack.
func (ts *FakeMesosTests) getResp(resp *http.Response, err error) *http.Response {
	return ts.WithoutError(resp, err).(*http.Response)
}

// postAPI wraps some API calls.
func (ts *FakeMesosTests) postAPI(url string, callType master.Call_Type) *http.Response {
	call := &master.Call{Type: callType}
	body := bytes.NewReader(ts.WithoutError(call.Marshal()).([]byte))
	return ts.getResp(http.Post(url, "application/x-protobuf", body))
}
