package mesosclient

import (
	"context"
	"fmt"

	"github.com/mesos/mesos-go/api/v1/lib/httpcli"
	"github.com/mesos/mesos-go/api/v1/lib/httpcli/httpmaster"
	"github.com/mesos/mesos-go/api/v1/lib/master"
	"github.com/mesos/mesos-go/api/v1/lib/master/calls"
)

// Client is a Mesos API client.
type Client struct {
	sender calls.Sender
}

// NewClient builds a new Client object that queries a Mesos API endpoint at
// the given base URL. The baseURL parameter should have the form
// "http://host:port".
func NewClient(baseURL string) *Client {
	endpoint := httpcli.Endpoint(fmt.Sprintf("%s/api/v1", baseURL))
	return &Client{
		sender: httpmaster.NewSender(httpcli.New(endpoint).Send),
	}
}

// GetTasks makes a GET_TASKS API call and returns the collection of tasks.
func (c *Client) GetTasks(ctx context.Context) (*master.Response_GetTasks, error) {
	gt := calls.GetTasks()
	resp, err := c.sender.Send(ctx, calls.NonStreaming(gt))
	if err != nil {
		return nil, err
	}
	var respData master.Response
	if err := resp.Decode(&respData); err != nil {
		// noqa: (Not actually a tag that does anything, sadly.)
		return nil, err
	}

	return respData.GetTasks, nil
}
