package mesosclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/mesos/mesos-go/api/v1/lib/httpcli"
	"github.com/mesos/mesos-go/api/v1/lib/httpcli/apierrors"
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
	resp, err := c.makeCallWithRedirect(ctx, rf, c.url, 10)
	if err != nil {
		return nil, err
	}
	var respData master.Response
	if err := resp.Decode(&respData); err != nil {
		return nil, err
	}

	return &respData, nil
}

// makeCallWithRedirect makes the given API call to the given URL and handles
// redirects.
func (c *Client) makeCallWithRedirect(ctx context.Context, rf calls.RequestFunc, url string, redirs int) (*httpcli.Response, error) {
	if redirs <= 0 {
		return nil, fmt.Errorf("too many redirects")
	}
	resp, err := c.getSender(url).Send(ctx, rf)
	if apierrors.CodeNotLeader.Matches(err) {
		res := resp.(*httpcli.Response)
		newURL := buildURL(url, res.Header.Get("Location"))
		return c.makeCallWithRedirect(ctx, rf, newURL, redirs-1)
	}
	if err != nil {
		return nil, err
	}
	return resp.(*httpcli.Response), nil
}

// buildURL returns newURL as-is if it has a scheme, otherwise it sticks the
// scheme from oldURL on the front and returns that. We use string juggling
// instead of net/url because I'm so tired of having to "handle" "errors"
// everywhere.
func buildURL(oldURL, newURL string) string {
	if strings.HasPrefix(newURL, "//") {
		// No scheme, so copy the one from the old URL.
		scheme := strings.SplitN(oldURL, "//", 2)[0]
		return scheme + newURL
	}
	return newURL
}
