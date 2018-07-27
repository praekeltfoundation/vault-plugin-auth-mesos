package testing

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

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

	ts.Equal(fm.tasks, taskMap{})
	fm.AddTask(task)
	ts.Equal(fm.tasks, taskMap{"abc-123": &task})
}

// We can add many new tasks to FakeMesos.
func (ts *FakeMesosTests) Test_AddTask_multiple() {
	fm := NewFakeMesos()
	ts.AddCleanup(fm.Close)

	task1 := mkTask("task", "abc-123", mesos.TASK_RUNNING)
	task2 := mkTask("task", "abc-124", mesos.TASK_RUNNING)

	ts.Equal(fm.tasks, taskMap{})
	fm.AddTask(task1, task2)
	ts.Equal(fm.tasks, taskMap{"abc-123": &task1, "abc-124": &task2})
}

// We can't add duplicate tasks to FakeMesos.
func (ts *FakeMesosTests) Test_AddTask_duplicate() {
	fm := NewFakeMesos()
	ts.AddCleanup(fm.Close)

	task1 := mkTask("task1", "abc-123", mesos.TASK_STAGING)
	task1dup := mkTask("task1", "abc-123", mesos.TASK_RUNNING)
	task2 := mkTask("task1", "abc-124", mesos.TASK_RUNNING)

	ts.Equal(fm.tasks, taskMap{})

	ts.Panics(func() { fm.AddTask(task1, task1dup, task2) })
	// We successfully added the first task, but not the duplicate or the
	// second task.
	ts.Equal(fm.tasks, taskMap{"abc-123": &task1})
}

// We can remove a task from FakeMesos.
func (ts *FakeMesosTests) Test_RemoveTask_one() {
	fm := NewFakeMesos()
	ts.AddCleanup(fm.Close)

	task := mkTask("task", "abc-123", mesos.TASK_RUNNING)
	fm.AddTask(task)
	ts.Equal(fm.tasks, taskMap{"abc-123": &task})

	fm.RemoveTask("abc-123")
	ts.Equal(fm.tasks, taskMap{})
}

// We can remove multiple tasks from FakeMesos. Missing tasks are ignored.
func (ts *FakeMesosTests) Test_RemoveTask_multiple() {
	fm := NewFakeMesos()
	ts.AddCleanup(fm.Close)

	task1 := mkTask("task", "abc-123", mesos.TASK_RUNNING)
	task2 := mkTask("task", "abc-124", mesos.TASK_RUNNING)
	task3 := mkTask("hello", "world-123", mesos.TASK_RUNNING)

	fm.AddTask(task1, task2, task3)
	ts.Equal(fm.tasks, taskMap{
		"abc-123":   &task1,
		"abc-124":   &task2,
		"world-123": &task3,
	})

	fm.RemoveTask("abc-123", "abc-127", "world-123")
	ts.Equal(fm.tasks, taskMap{"abc-124": &task2})
}

// We can update a task in FakeMesos.
func (ts *FakeMesosTests) Test_UpdateTask_one() {
	fm := NewFakeMesos()
	ts.AddCleanup(fm.Close)

	task := mkTask("task", "abc-123", mesos.TASK_RUNNING)
	fm.AddTask(task)
	ts.Equal(fm.tasks, taskMap{"abc-123": &task})

	fm.UpdateTask(func(t *mesos.Task) { t.Name = "cask" }, "abc-123")

	updatedTask := mkTask("cask", "abc-123", mesos.TASK_RUNNING)
	ts.Equal(fm.tasks, taskMap{"abc-123": &updatedTask})
}

// We can update multiple tasks in FakeMesos.
func (ts *FakeMesosTests) Test_UpdateTask_multiple() {
	fm := NewFakeMesos()
	ts.AddCleanup(fm.Close)

	task1 := mkTask("task", "abc-123", mesos.TASK_RUNNING)
	task2 := mkTask("task", "abc-124", mesos.TASK_RUNNING)
	task3 := mkTask("hello", "world-123", mesos.TASK_RUNNING)

	fm.AddTask(task1, task2, task3)
	ts.Equal(fm.tasks, taskMap{
		"abc-123":   &task1,
		"abc-124":   &task2,
		"world-123": &task3,
	})

	fm.UpdateTask(UpdateState(mesos.TASK_FINISHED), "abc-123", "world-123")

	updatedTask1 := mkTask("task", "abc-123", mesos.TASK_FINISHED)
	updatedTask3 := mkTask("hello", "world-123", mesos.TASK_FINISHED)
	ts.Equal(fm.tasks, taskMap{
		"abc-123":   &updatedTask1,
		"abc-124":   &task2,
		"world-123": &updatedTask3,
	})
}

// We can't update missing tasks in FakeMesos.
func (ts *FakeMesosTests) Test_UpdateTask_missing() {
	fm := NewFakeMesos()
	ts.AddCleanup(fm.Close)

	task1 := mkTask("task", "abc-123", mesos.TASK_RUNNING)
	task2 := mkTask("task", "abc-124", mesos.TASK_RUNNING)
	task3 := mkTask("hello", "world-123", mesos.TASK_RUNNING)

	fm.AddTask(task1, task2, task3)
	ts.Equal(fm.tasks, taskMap{
		"abc-123":   &task1,
		"abc-124":   &task2,
		"world-123": &task3,
	})

	f := UpdateState(mesos.TASK_FINISHED)
	ts.Panics(func() { fm.UpdateTask(f, "abc-123", "abc-127", "world-123") })
	// We successfully updated task1, but not the missing task or task3.
	updatedTask1 := mkTask("task", "abc-123", mesos.TASK_FINISHED)
	ts.Equal(fm.tasks, taskMap{
		"abc-123":   &updatedTask1,
		"abc-124":   &task2,
		"world-123": &task3,
	})
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

// Even without tasks to get, we still simulate request latency.
func (ts *FakeMesosTests) Test_API_GET_TASKS_latency() {
	latency := 200 * time.Millisecond
	fm := NewFakeMesos()
	fm.SetLatency(latency)
	ts.AddCleanup(fm.Close)

	start := time.Now()
	resp := ts.postAPI(fm.GetAPIURL(), master.Call_GET_TASKS)
	ts.Equal(resp.StatusCode, 200)
	elapsed := time.Since(start)
	ts.Truef(elapsed >= latency, "Expected latency of at least %s, got %s.", latency, elapsed)
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

// We can get the tasks if tasks exist.
func (ts *FakeMesosTests) Test_err2panic() {
	ts.NotPanics(func() { err2panic(nil) })
	err := errors.New("oops")
	ts.PanicsWithValue(err, func() { err2panic(err) })
}
