package mesosauth

import (
	"context"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/helper/logging"
	"github.com/hashicorp/vault/logical"
	"github.com/praekeltfoundation/vault-plugin-auth-mesos/testutils"
)

// This file contains infrastructure and tools common to multiple tests.

// TestSuite is a testify test suite object that we can attach helper methods
// to.
type TestSuite struct {
	testutils.TestSuite
	storage logical.Storage
	backend *mesosBackend
}

// SetupTest clears all our TestSuite state at the start of each test, because
// the same object is shared across all tests.
func (ts *TestSuite) SetupTest() {
	ts.TestSuite.SetupTest()

	ts.storage = nil
	ts.backend = nil
	// Clear our hacky task set global for each test.
	temporarySetOfExistingTasks = map[string]bool{}
}

// SetupBackend creates a suitable backend object (and associated storage
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

// mkReq builds a basic request object.
func (ts *TestSuite) mkReq(path string, data jsonobj) *logical.Request {
	return &logical.Request{
		Operation:  logical.UpdateOperation,
		Connection: &logical.Connection{},
		Path:       path,
		Data:       data,
		Storage:    ts.storage,
	}
}

// HandleRequestRaw is a thin wrapper around the backend's HandleRequest method
// to avoid some boilerplate in the tests.
func (ts *TestSuite) HandleRequestRaw(req *logical.Request) (*logical.Response, error) {
	ts.Require().NotNil(ts.backend, "Backend not set up.")
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
