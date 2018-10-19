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
	url string
}

// NewClient builds a new Client object that queries a Mesos API endpoint at
// the given base URL. The baseURL parameter should have the form
// "http://host:port".
func NewClient(baseURL string) *Client {
	return &Client{
		url: fmt.Sprintf("%s/api/v1", baseURL),
	}
}

// getSender returns a Sender for the given URL.
func (c *Client) getSender(url string) calls.Sender {
	return httpmaster.NewSender(httpcli.New(httpcli.Endpoint(url)).Send)
}

// getDefaultSender returns a Sender for the default URL.
func (c *Client) getDefaultSender() calls.Sender {
	return c.getSender(c.url)
}

// GetTasks makes a GET_TASKS API call and returns the collection of tasks.
func (c *Client) GetTasks(ctx context.Context) (*master.Response_GetTasks, error) {
	respData, err := c.makeCall(ctx, calls.NonStreaming(calls.GetTasks()))
	if err != nil {
		return nil, err
	}

	return respData.GetTasks, nil
}

// makeCall makes the given API call and returns the response.
func (c *Client) makeCall(ctx context.Context, rf calls.RequestFunc) (*master.Response, error) {
	resp, err := c.getDefaultSender().Send(ctx, rf)
	if err != nil {
		return nil, err
	}
	var respData master.Response
	if err := resp.Decode(&respData); err != nil {
		return nil, err
	}

	return &respData, nil
}
