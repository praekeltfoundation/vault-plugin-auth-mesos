package mesosauth

import (
	"testing"
	"time"

	"github.com/hashicorp/vault/logical"
	"github.com/stretchr/testify/suite"
)

// See helper_for_test.go for common infrastructure and tools.

// ConfigTests is a testify test suite object that we can attach helper
// methods to.
type ConfigTests struct{ TestSuite }

// Test_Config is a standard Go test function that runs our test suite's
// tests.
func Test_Config(t *testing.T) { suite.Run(t, new(ConfigTests)) }

// We cannot create an invalid config.
func (ts *ConfigTests) Test_create_invalid() {
	ts.SetupBackend()
	ts.Nil(ts.GetStored("config"))

	req := ts.mkReq("config", jsonobj{"ttl": "42s"})
	resp := ts.HandleRequest(req)
	ts.EqualError(resp.Error(), "base-url not configured")

	ts.Nil(ts.GetStored("config"))
}

// We can create a new config.
func (ts *ConfigTests) Test_create_full() {
	ts.SetupBackend()
	ts.Nil(ts.GetStored("config"))

	req := ts.mkReq("config", jsonobj{
		"base-url": "http://master.mesos:5050",
		"ttl":      "42s",
	})
	ts.Equal(ts.HandleRequest(req), &logical.Response{})

	ts.StoredEqual("config", config{
		BaseURL: "http://master.mesos:5050",
		TTL:     42 * time.Second,
	})
}

// We can completely replace a config.
func (ts *ConfigTests) Test_update_full() {
	ts.SetupBackend()
	ts.HandleRequestSuccess(ts.mkReq("config", jsonobj{
		"base-url": "http://master.mesos:5050",
		"ttl":      "42s",
	}))
	ts.StoredEqual("config", config{
		BaseURL: "http://master.mesos:5050",
		TTL:     42 * time.Second,
	})

	req := ts.mkReq("config", jsonobj{
		"base-url": "http://localhost:5050",
		"ttl":      "7m",
	})
	ts.Equal(ts.HandleRequest(req), &logical.Response{})

	ts.StoredEqual("config", config{
		BaseURL: "http://localhost:5050",
		TTL:     420 * time.Second,
	})
}

// We can partially update a config.
func (ts *ConfigTests) Test_update_partial() {
	ts.SetupBackend()
	ts.HandleRequestSuccess(ts.mkReq("config", jsonobj{
		"base-url": "http://master.mesos:5050",
		"ttl":      "42s",
	}))
	ts.StoredEqual("config", config{
		BaseURL: "http://master.mesos:5050",
		TTL:     42 * time.Second,
	})

	// Update just the TTL.
	req1 := ts.mkReq("config", jsonobj{"ttl": "7m"})
	ts.Equal(ts.HandleRequest(req1), &logical.Response{})

	ts.StoredEqual("config", config{
		BaseURL: "http://master.mesos:5050",
		TTL:     420 * time.Second,
	})

	// Update just the base URL.
	req2 := ts.mkReq("config", jsonobj{"base-url": "http://localhost:5050"})
	ts.Equal(ts.HandleRequest(req2), &logical.Response{})

	ts.StoredEqual("config", config{
		BaseURL: "http://localhost:5050",
		TTL:     420 * time.Second,
	})
}

// If there is no config to read, we get a nil response.
func (ts *ConfigTests) Test_read_no_config() {
	ts.SetupBackend()
	ts.Nil(ts.GetStored("config"))

	req := ts.mkReadReq("config")
	ts.Nil(ts.HandleRequest(req))
}

// We can read the existing config.
func (ts *ConfigTests) Test_read_existing_config() {
	ts.SetupBackend()
	ts.HandleRequestSuccess(ts.mkReq("config", jsonobj{
		"base-url": "http://master.mesos:5050",
		"ttl":      "420s",
	}))

	req := ts.mkReadReq("config")
	ts.Equal(ts.HandleRequest(req), &logical.Response{
		Data: jsonobj{
			"base-url": "http://master.mesos:5050",
			"ttl":      "7m0s",
		},
	})
}

// We cannot read or update a broken config.
func (ts *ConfigTests) Test_broken_config() {
	ts.SetupBackend()
	// Manually write a config value that cannot be unmarshalled.
	ts.PutStored("config", jsonobj{"BaseURL": jsonobj{}})

	errmsg := "json: cannot unmarshal object into Go struct field config.BaseURL of type string"
	ts.HandleRequestError(ts.mkReadReq("config"), errmsg)
	ts.HandleRequestError(ts.mkReq("config", jsonobj{"ttl": "42s"}), errmsg)
}
