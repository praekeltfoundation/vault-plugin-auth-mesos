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

// FakeMesos pretends to be a subset of the Mesos v1 API. Only protobuf
// payloads are supported, not JSON.
type FakeMesos struct {
	*httptest.Server
	tasks map[string]mesos.Task
}

// NewFakeMesos does what it says on the tin. It needs to be stopped with a
// call to .Close() when the test is over.
func NewFakeMesos() *FakeMesos {
	fm := FakeMesos{tasks: map[string]mesos.Task{}}
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
			appendTask(&getTasks.Tasks, task)
		case stateTerminated:
			appendTask(&getTasks.CompletedTasks, task)
		case stateUnreachable:
			appendTask(&getTasks.UnreachableTasks, task)
		}
	}
	return &getTasks
}

// respondGetTasks returns a GET_TASKS response.
func (fm *FakeMesos) respondGetTasks(w http.ResponseWriter) {
	resp := master.Response{
		Type:     master.Response_GET_TASKS,
		GetTasks: fm.getTasks(),
	}
	data, err := resp.Marshal()
	if err != nil {
		http.Error(w, "internal error: "+err.Error(), 500)
		return
	}
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
		fm.tasks[task.TaskID.Value] = task
	}
}
