package mesosAuthPlugin

import (
	"context"
	"time"

	"github.com/hashicorp/vault/logical"
)

// See helper_for_test.go for common infrastructure and tools.

func (ts *TestSuite) Test_login_no_task_id() {
	b := ts.mkBackend()
	req := ts.mkReq("login", jsonobj{})

	resp, err := b.HandleRequest(context.Background(), req)
	ts.EqualError(err, "permission denied")
	ts.Nil(resp)
}

func (ts *TestSuite) Test_login_good_task_id() {
	b := ts.mkBackend()
	req := ts.mkReq("login", jsonobj{"task-id": "task-that-exists"})

	resp, err := b.HandleRequest(context.Background(), req)
	ts.Require().NoError(err)
	ts.Nil(resp.Warnings)
	ts.Nil(resp.Secret)
	ts.Equal(resp.Auth, &logical.Auth{
		Policies:     []string{},
		Period:       10 * time.Minute,
		LeaseOptions: logical.LeaseOptions{Renewable: true},
	})
}
