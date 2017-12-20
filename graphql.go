// Package graphql provides a low level GraphQL client.
//
//  // create a client (safe to share across requests)
//  client := graphql.NewClient("https://machinebox.io/graphql")
//
//  // make a request
//  req := graphql.NewRequest(`
//      query ($key: String!) {
//          items (id:$key) {
//              field1
//              field2
//              field3
//          }
//      }
//  `)
//
//  // set any variables
//  req.Var("key", "value")
//
//  // run it and capture the response
//  var respData ResponseStruct
//  if err := client.Run(ctx, req, &respData); err != nil {
//      log.Fatal(err)
//  }
//
// Specify client
//
// To specify your own http.Client, use the WithHTTPClient option:
//  httpclient := &http.Client{}
//  client := graphql.NewClient("https://machinebox.io/graphql", graphql.WithHTTPClient(httpclient))
package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

// Client is a client for accessing a GraphQL dataset.
type Client struct {
	endpoint   string
	httpClient *http.Client
}

// NewClient makes a new Client capable of making GraphQL requests.
func NewClient(endpoint string, opts ...ClientOption) *Client {
	c := &Client{
		endpoint: endpoint,
	}
	for _, optionFunc := range opts {
		optionFunc(c)
	}
	if c.httpClient == nil {
		c.httpClient = http.DefaultClient
	}
	return c
}

// Run executes the query and unmarshals the response from the data field
// into the response object.
// Pass in a nil response object to skip response parsing.
// If the request fails or the server returns an error, the first error
// will be returned.
func (c *Client) Run(ctx context.Context, req *Request, resp interface{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	var graphResponse = struct {
		Data   interface{}
		Errors []graphErr
	}{
		Data: resp,
	}

	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	r, err := http.NewRequest(http.MethodPost, c.endpoint, bytes.NewReader(b))
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	r = r.WithContext(ctx)
	res, err := c.httpClient.Do(r)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, res.Body); err != nil {
		return errors.Wrap(err, "reading body")
	}
	if err := json.NewDecoder(&buf).Decode(&graphResponse); err != nil {
		return errors.Wrap(err, "decoding response")
	}
	if len(graphResponse.Errors) > 0 {
		// return first error
		return graphResponse.Errors[0]
	}
	return nil
}

// WithHTTPClient specifies the underlying http.Client to use when
// making requests.
//  NewClient(endpoint, WithHTTPClient(specificHTTPClient))
func WithHTTPClient(httpclient *http.Client) ClientOption {
	return ClientOption(func(client *Client) {
		client.httpClient = httpclient
	})
}

// ClientOption are functions that are passed into NewClient to
// modify the behaviour of the Client.
type ClientOption func(*Client)

type graphErr struct {
	Message string
}

func (e graphErr) Error() string {
	return "graphql: " + e.Message
}

// Request is a GraphQL request.
type Request struct {
	OperationName string                 `json:"operationName,omitempty"`
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
}

// NewRequest makes a new Request with the specified string.
func NewRequest(q string) *Request {
	req := &Request{
		Query: q,
	}
	return req
}

// Var sets a variable.
func (req *Request) Var(key string, value interface{}) {
	if req.Variables == nil {
		req.Variables = make(map[string]interface{})
	}
	req.Variables[key] = value
}
