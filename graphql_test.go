package graphql_test

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/joefitzgerald/graphql"
	. "github.com/onsi/gomega"
)

func TestWithClient(t *testing.T) {
	RegisterTestingT(t)
	var calls int
	testClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			resp := &http.Response{
				Body: ioutil.NopCloser(strings.NewReader(`{"data":{"key":"value"}}`)),
			}
			return resp, nil
		}),
	}

	ctx := context.Background()
	client := graphql.NewClient("", graphql.WithHTTPClient(testClient))

	req := graphql.NewRequest(``)
	client.Run(ctx, req, nil)

	Expect(calls).Should(Equal(1))
}

func TestDo(t *testing.T) {
	RegisterTestingT(t)
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		Expect(r.Method).Should(Equal(http.MethodPost))
		defer r.Body.Close()
		var req graphql.Request
		json.NewDecoder(r.Body).Decode(&req)
		Expect(req.Query).Should(Equal(`query {}`))
		io.WriteString(w, `{
			"data": {
				"something": "yes"
			}
		}`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := graphql.NewClient(srv.URL)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &graphql.Request{Query: "query {}"}, &responseData)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(calls).Should(Equal(1))
	Expect(responseData["something"]).Should(Equal("yes"))
}

func TestDoErr(t *testing.T) {
	RegisterTestingT(t)
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		Expect(r.Method).Should(Equal(http.MethodPost))
		defer r.Body.Close()
		var req graphql.Request
		json.NewDecoder(r.Body).Decode(&req)
		Expect(req.Query).Should(Equal(`query {}`))
		io.WriteString(w, `{
			"errors": [{
				"message": "Something went wrong"
			}]
		}`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := graphql.NewClient(srv.URL)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &graphql.Request{Query: "query {}"}, &responseData)
	Expect(err).ShouldNot(BeNil())
	Expect(err.Error()).Should(Equal("graphql: Something went wrong"))
}

func TestDoNoResponse(t *testing.T) {
	RegisterTestingT(t)
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		Expect(r.Method).Should(Equal(http.MethodPost))
		defer r.Body.Close()
		var req graphql.Request
		json.NewDecoder(r.Body).Decode(&req)
		Expect(req.Query).Should(Equal(`query {}`))
		io.WriteString(w, `{
			"data": {
				"something": "yes"
			}
		}`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := graphql.NewClient(srv.URL)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	err := client.Run(ctx, &graphql.Request{Query: "query {}"}, nil)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(calls).Should(Equal(1))
}

func TestQuery(t *testing.T) {
	RegisterTestingT(t)

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		defer r.Body.Close()
		var req graphql.Request
		json.NewDecoder(r.Body).Decode(&req)
		Expect(req.Query).Should(Equal("query {}"))
		expected := map[string]interface{}{}
		expected["username"] = "tester"
		Expect(req.Variables).Should(Equal(expected))
		_, err := io.WriteString(w, `{"data":{"value":"some data"}}`)
		Expect(err).ShouldNot(HaveOccurred())
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	client := graphql.NewClient(srv.URL)

	req := graphql.NewRequest("query {}")
	req.Var("username", "tester")

	// check variables
	Expect(req).ShouldNot(BeNil())
	Expect(req.Variables["username"]).Should(Equal("tester"))

	var resp struct {
		Value string
	}
	err := client.Run(ctx, req, &resp)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(calls).Should(Equal(1))
	Expect(resp.Value).Should(Equal("some data"))
}

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
