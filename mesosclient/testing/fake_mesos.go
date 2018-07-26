// Package testing provides tools for testing Mesos client interactions.
package testing

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	mesos "github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/master"
)

// These constants represent the "lifecyle stages" of a Mesos task. Each task
// state (except TASK_UNKNOWN) maps to exactly one lifecycle stage.
const (
	stateActive = iota
	stateTerminated
	stateUnreachable
)

// stateLifecycle should really be a constant, but that's not allowed for maps.
var stateLifecycle = map[mesos.TaskState]int{
	mesos.TASK_STAGING:  stateActive,
	mesos.TASK_STARTING: stateActive,
	mesos.TASK_RUNNING:  stateActive,
	mesos.TASK_KILLING:  stateActive,

	mesos.TASK_FINISHED:         stateTerminated,
	mesos.TASK_FAILED:           stateTerminated,
	mesos.TASK_KILLED:           stateTerminated,
	mesos.TASK_ERROR:            stateTerminated,
	mesos.TASK_GONE:             stateTerminated,
	mesos.TASK_GONE_BY_OPERATOR: stateTerminated,

	mesos.TASK_LOST:        stateUnreachable,
	mesos.TASK_DROPPED:     stateUnreachable,
	mesos.TASK_UNREACHABLE: stateUnreachable,
}

// A taskMap is a collection of tasks.
type taskMap map[string]*mesos.Task

// FakeMesos pretends to be a subset of the Mesos v1 API. Only protobuf
// payloads are supported, not JSON.
type FakeMesos struct {
	*httptest.Server
	tasks taskMap
}

// NewFakeMesos does what it says on the tin. It needs to be stopped with a
// call to .Close() when the test is over.
func NewFakeMesos() *FakeMesos {
	fm := FakeMesos{tasks: taskMap{}}
	fm.Server = httptest.NewServer(http.HandlerFunc(fm.handleAPI))
	return &fm
}

// GetBaseURL returns the fake server's base URL.
func (fm *FakeMesos) GetBaseURL() string {
	return fm.Server.URL
}

// GetAPIURL returns the fake server's API URL.
func (fm *FakeMesos) GetAPIURL() string {
	return fm.GetBaseURL() + "/api/v1"
}

// handleAPI parses and dispatches Mesos API calls.
func (fm *FakeMesos) handleAPI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}
	if r.Header.Get("Content-Type") != "application/x-protobuf" {
		http.Error(w, "", http.StatusUnsupportedMediaType)
		return
	}
	// Various kinds of bad data can happily unmarshal to a valid zero-value
	// Call struct. Rather than worry about all the different ways this can
	// fail, we just ignore any errors and pretend they're valid zero-value
	// requests.
	var call master.Call
	bytes, _ := ioutil.ReadAll(r.Body)
	_ = call.Unmarshal(bytes)
	switch call.Type {
	case master.Call_GET_TASKS:
		fm.respondGetTasks(w)
	default:
		http.Error(w, "invalid operation: "+call.Type.String(), 400)
	}
}

// appendTask lets us avoid specifying the slice we're appending to twice.
func appendTask(tl *[]mesos.Task, t mesos.Task) { *tl = append(*tl, t) }

// getTasks collects the tasks we know about into a suitable container.
//
// The PendingTasks (not yet accepted by Mesos) and OrphanTasks (deprecated
// since Mesos 1.2.0) fields will always be empty. Tasks in state TASK_UNKNOWN
// are never returned, because Mesos doesn't know about them.
func (fm *FakeMesos) getTasks() *master.Response_GetTasks {
	getTasks := master.Response_GetTasks{}
	for _, task := range fm.tasks {
		switch stateLifecycle[*task.State] {
		case stateActive:
			appendTask(&getTasks.Tasks, copyTask(task))
		case stateTerminated:
			appendTask(&getTasks.CompletedTasks, copyTask(task))
		case stateUnreachable:
			appendTask(&getTasks.UnreachableTasks, copyTask(task))
		}
	}
	return &getTasks
}

// copyTask serialises and deserialises a task in order to deep-copy it.
func copyTask(taskIn *mesos.Task) mesos.Task {
	// We ignore all errors, because we expect the generated serialisation code
	// to correctly-roud-trip any task we give it.
	bytes, _ := taskIn.Marshal()
	var taskOut mesos.Task
	_ = taskOut.Unmarshal(bytes)
	return taskOut
}

// respondGetTasks returns a GET_TASKS response.
func (fm *FakeMesos) respondGetTasks(w http.ResponseWriter) {
	resp := master.Response{
		Type:     master.Response_GET_TASKS,
		GetTasks: fm.getTasks(),
	}
	data, err := resp.Marshal()
	err2panic(err)
	w.Header().Set("Content-Type", "application/x-protobuf")
	_, _ = w.Write(data)
}

// AddTask adds one or more new tasks to fake Mesos. Panics if a task already
// exists.
func (fm *FakeMesos) AddTask(tasks ...mesos.Task) {
	for _, task := range tasks {
		if _, ok := fm.tasks[task.TaskID.Value]; ok {
			panic(fmt.Sprintf("Duplicate task: %s", task.TaskID.Value))
		}
		taskCopy := copyTask(&task)
		fm.tasks[task.TaskID.Value] = &taskCopy
	}
}

// RemoveTask removes one or more tasks by id. Missing tasks are ignored.
func (fm *FakeMesos) RemoveTask(taskIDs ...string) {
	for _, taskID := range taskIDs {
		delete(fm.tasks, taskID)
	}
}

// TaskUpdateFunc is the type of a function that updates a task.
//
// The task passed in should be updated in-place. Updates to TaskID are not
// supported and may result in undefined behaviour.
type TaskUpdateFunc func(task *mesos.Task)

// UpdateTask updates one or more tasks using the given update function. Panics
// if a task doesn't exist.
func (fm *FakeMesos) UpdateTask(updateFunc TaskUpdateFunc, taskIDs ...string) {
	for _, taskID := range taskIDs {
		task, ok := fm.tasks[taskID]
		if !ok {
			panic(fmt.Sprintf("Unkown task: %s", task.TaskID.Value))
		}
		updateFunc(task)
	}
}

// UpdateState returns a closure that updates the state of a task.
//
// This is less trivial than it appears, because Task.State is a pointer and
// you can't take the address of a constant.
func UpdateState(state mesos.TaskState) TaskUpdateFunc {
	return func(task *mesos.Task) { task.State = &state }
}

// err2panic lets us turn "impossible" errors into panics without leaving
// untested error handlers in our code.
func err2panic(err error) {
	if err != nil {
		panic(err)
	}
}
