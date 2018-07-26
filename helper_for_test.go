package mesosauth

import (
	"context"
	"os"
	"time"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/helper/logging"
	"github.com/hashicorp/vault/logical"
	mesos "github.com/mesos/mesos-go/api/v1/lib"
	mctesting "github.com/praekeltfoundation/vault-plugin-auth-mesos/mesosclient/testing"
	"github.com/praekeltfoundation/vault-plugin-auth-mesos/testutils"
)

// This file contains infrastructure and tools common to multiple tests.

// TestSuite is a testify test suite object that we can attach helper methods
// to.
type TestSuite struct {
	testutils.TestSuite
	storage   logical.Storage
	fakeMesos *mctesting.FakeMesos
	backend   *mesosBackend
}

// SetupTest clears all our TestSuite state at the start of each test, because
// the same object is shared across all tests.
func (ts *TestSuite) SetupTest() {
	ts.TestSuite.SetupTest()

	ts.storage = nil
	ts.fakeMesos = nil
	ts.backend = nil
}

// SetupBackend creates an unconfigured backend object (and associated storage
// object) for use in a test.
func (ts *TestSuite) SetupBackend() {
	ts.Require().Nil(ts.backend, "Backend already set up.")

	ts.storage = &logical.InmemStorage{}
	config := &logical.BackendConfig{
		Logger:      logging.NewVaultLogger(log.Trace),
		StorageView: ts.storage,
	}

	ts.backend = ts.WithoutError(Factory(context.Background(), config)).(*mesosBackend)
}

// SetupBackendWithMesos creates a FakeMesos and a backend configured to use
// it. This is handy for tests that require a configured backend that
// communicates with Mesos.
func (ts *TestSuite) SetupBackendWithMesos() {
	ts.SetupBackend()
	ts.fakeMesos = mctesting.NewFakeMesos()
	ts.fakeMesos.SetLatency(getLatencyFromEnv())
	ts.AddCleanup(ts.fakeMesos.Close)
	ts.ConfigureBackend(ts.fakeMesos.GetBaseURL())
}

// getLatencyFromEnv reads the FakeMesos request latency from the environment.
// An unset envvar (the default) means no latency. Invalid duration strings
// cause panic and chaos.
func getLatencyFromEnv() time.Duration {
	latency, err := time.ParseDuration(getenv("VPAM_FAKE_MESOS_LATENCY", "0s"))
	if err != nil {
		panic(err)
	}
	return latency
}

// getenv is a wrapper around os.Getenv that returns a default value for empty
// or unset envvars.
func getenv(key, dflt string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return dflt
}

// ConfigureBackend configures the backend with the minimum mandatory settings
// on top of the defaults.
func (ts *TestSuite) ConfigureBackend(baseURL string) {
	ts.requireBackend()
	cfg := configDefault()
	cfg.BaseURL = baseURL
	ts.PutStored("config", cfg)
}

// requireBackend asserts that this test has a non-nil backend.
func (ts *TestSuite) requireBackend() {
	ts.Require().NotNil(ts.backend, "Backend not set up.")
}

// requireFakeMesos asserts that this test has a non-nil fakeMesos.
func (ts *TestSuite) requireFakeMesos() {
	ts.Require().NotNil(ts.fakeMesos, "FakeMesos not set up.")
}

// AddTask adds one or more new tasks to fake Mesos. Panics if a task already
// exists.
func (ts *TestSuite) AddTask(tasks ...mesos.Task) {
	ts.requireFakeMesos()
	ts.fakeMesos.AddTask(tasks...)
}

// RemoveTask removes one or more tasks by id. Missing tasks are ignored.
func (ts *TestSuite) RemoveTask(taskIDs ...string) {
	ts.requireFakeMesos()
	ts.fakeMesos.RemoveTask(taskIDs...)
}

// UpdateTask updates one or more tasks using the given update function. Panics
// if a task doesn't exist.
func (ts *TestSuite) UpdateTask(updateFunc mctesting.TaskUpdateFunc, taskIDs ...string) {
	ts.requireFakeMesos()
	ts.fakeMesos.UpdateTask(updateFunc, taskIDs...)
}

// mkReq builds a basic update request object.
func (ts *TestSuite) mkReq(path string, data jsonobj) *logical.Request {
	return &logical.Request{
		Operation:  logical.UpdateOperation,
		Connection: &logical.Connection{},
		Path:       path,
		Data:       data,
		Storage:    ts.storage,
	}
}

// mkReadReq builds a basic read request object.
func (ts *TestSuite) mkReadReq(path string) *logical.Request {
	return &logical.Request{
		Operation:  logical.ReadOperation,
		Connection: &logical.Connection{},
		Path:       path,
		Storage:    ts.storage,
	}
}

// HandleRequestRaw is a thin wrapper around the backend's HandleRequest method
// to avoid some boilerplate in the tests.
func (ts *TestSuite) HandleRequestRaw(req *logical.Request) (*logical.Response, error) {
	ts.requireBackend()
	return ts.backend.HandleRequest(context.Background(), req)
}

// HandleRequestError asserts that a request errors.
func (ts *TestSuite) HandleRequestError(req *logical.Request, errmsg string) {
	_, err := ts.HandleRequestRaw(req)
	ts.EqualError(err, errmsg)
}

// HandleRequest is a thin wrapper around HandleRequestRaw to handle
// non-response errors.
func (ts *TestSuite) HandleRequest(req *logical.Request) *logical.Response {
	return ts.WithoutError(ts.HandleRequestRaw(req)).(*logical.Response)
}

// HandleRequestSuccess asserts that the response is not an error.
func (ts *TestSuite) HandleRequestSuccess(req *logical.Request) *logical.Response {
	resp := ts.HandleRequest(req)
	ts.NoError(resp.Error())
	return resp
}

// Login makes a login request which is required to be successful and returns
// the resulting auth data.
func (ts *TestSuite) Login(taskID string) *logical.Auth {
	req := ts.mkReq("login", jsonobj{"task-id": taskID})
	resp := ts.HandleRequestSuccess(req)
	return resp.Auth
}

// GetStored retrieves a value from Vault storage.
func (ts *TestSuite) GetStored(key string) *logical.StorageEntry {
	return ts.WithoutError(ts.storage.Get(context.Background(), key)).(*logical.StorageEntry)
}

// PutStored writes a value to Vault storage.
func (ts *TestSuite) PutStored(key string, value interface{}) {
	ts.Require().NoError(
		ts.storage.Put(context.Background(), ts.mkStorageEntry(key, value)))
}

// DeleteStored removes a value from Vault storage.
func (ts *TestSuite) DeleteStored(key string) {
	ts.Require().NoError(ts.storage.Delete(context.Background(), key))
}

// mkStorageEntry builds a StorageEntry object with errors handled.
func (ts *TestSuite) mkStorageEntry(key string, value interface{}) *logical.StorageEntry {
	return ts.WithoutError(logical.StorageEntryJSON(key, value)).(*logical.StorageEntry)
}

// StoredEqual asserts that the value stored at a particular key is equal to
// the given value.
func (ts *TestSuite) StoredEqual(key string, expected interface{}) {
	ts.Equal(ts.GetStored(key), ts.mkStorageEntry(key, expected))
}

// SetTaskPolicies sets task policies through the API.
func (ts *TestSuite) SetTaskPolicies(taskPrefix string, policies ...string) {
	ts.HandleRequestSuccess(ts.mkReq("task-policies", tpParams(taskPrefix, policies)))
}

// tpParams removes boilerplate from request creation.
func tpParams(taskPrefix string, policies interface{}) jsonobj {
	return jsonobj{"task-id-prefix": taskPrefix, "policies": policies}
}
