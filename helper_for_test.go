package mesosAuthPlugin

import (
	"context"
	"testing"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/helper/logging"
	"github.com/hashicorp/vault/logical"
	"github.com/stretchr/testify/suite"
)

// This file contains infrastructure and tools common to multiple tests.

// jsonobj is an alias for type a JSON object gets unmarshalled into, because
// building nested map[string]interface{}{ ... } literals is awful.
type jsonobj = map[string]interface{}

// TestSuite is a testify test suite object that we can attach helper methods
// to.
type TestSuite struct {
	suite.Suite
	cleanups []func()

	storage logical.Storage
	backend *mesosBackend
}

func Test_TestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

// AddCleanup schedules the given cleanup function to be run after the test.
// Think of it like `defer`, except it applies to the whole test rather than
// the specific function it appears in.
func (ts *TestSuite) AddCleanup(f func()) {
	ts.cleanups = append([]func(){f}, ts.cleanups...)
}

// SetupTest clears all our TestSuite state at the start of each test, because
// the same object is shared across all tests.
func (ts *TestSuite) SetupTest() {
	ts.cleanups = []func(){}
	ts.storage = nil
	ts.backend = nil
}

// TearDownTest calls the registered cleanup functions.
func (ts *TestSuite) TearDownTest() {
	for _, f := range ts.cleanups {
		f()
	}
}

// WithoutError accepts a (result, error) pair, immediately fails the test if
// there is an error, and returns just the result if there is no error. It
// accepts and returns the result value as an `interface{}`, so it may need to
// be cast back to whatever type it should be afterwards.
func (ts *TestSuite) WithoutError(result interface{}, err error) interface{} {
	ts.T().Helper()
	ts.Require().NoError(err)
	return result
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

func (ts *TestSuite) mkReq(path string, data jsonobj) *logical.Request {
	return &logical.Request{
		Operation:  logical.UpdateOperation,
		Connection: &logical.Connection{},
		Path:       path,
		Data:       data,
		Storage:    ts.storage,
	}
}

// HandleRequest is a thin wrapper around the backend's HandleRequest method to
// avoid some boilerplate in the tests.
func (ts *TestSuite) HandleRequest(req *logical.Request) (*logical.Response, error) {
	ts.Require().NotNil(ts.backend, "Backend not set up.")
	return ts.backend.HandleRequest(context.Background(), req)
}

func (ts *TestSuite) Login(taskID string) *logical.Auth {
	req := ts.mkReq("login", jsonobj{"task-id": taskID})

	resp := ts.WithoutError(ts.HandleRequest(req)).(*logical.Response)
	return resp.Auth
}

func (ts *TestSuite) ResponseError(resp *logical.Response, errMsg string) {
	ts.Nil(resp.Auth)
	ts.Nil(resp.Secret)
	ts.Equal(resp.Data, jsonobj{"error": errMsg})
}
