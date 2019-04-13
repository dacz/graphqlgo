// Package graphqlgo provides a low level GraphQL client.
//
// The purpose is not to be fast or memory efficient client. The intentions were:
//
// * to support existing standards
// * to allow maximum request & response debugging
//
// The graphqlgo.Client is not concurrent safe and not intended to be.
// If you need to make conccurrent requests, instantiate separate clients for them.
//
// The Client keeps it's state (as request headers, data sent, received headers etc.).
// The request and response it can be inspected after the request.
// Every client.Run resets this state therefore you can use the client for multiple
// non-concurrent requests.
//
// Options for Client
//
// To specify your own http.Client, use the WithHTTPClient option:
//   httpclient := &http.Client{}
//   client := graphqlgo.NewClient("https://...", graphql.WithHTTPClient(httpclient))
//
// To specify headers to be added to every request:
// 	 authHeader := http.Header{ "Authorization", {"someToken"}}
//   client := graphqlgo.NewClient("https://...", graphql.WithHeaders(authHeader))
// The headers "Content-Type": "application/json; charset=utf-8"
// and "Accept": "application/json; charset=utf-8"
// are specified by default, you don't need to set them.
//
// To close the http request immediately so the socket can be resused
//   client := graphqlgo.NewClient("https://...", ImmediatelyCloseReqBody())
//
// Options for request
//
// To specify variables for the request or operation name
//   vars := map[string]interface{}{
//		"code": "AF",
//	  }
//    query := `
//  	query continent($code: String!) {
//  		continent(code: $code) {
//  			code
//  			name
//  		}
//  	}`
//   opName := "continent"
//   eq := graphqlgo.NewRequest(
//		query,
//		graphqlgo.WithVars(vars),
//		graphqlgo.WithOperationName(opName)
// 	 )
//
// See example.
package graphqlgo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

// Client is a client for interacting with a GraphQL API.
type Client struct {
	Endpoint   string
	httpClient *http.Client

	// headers to be set on every request with this client (like auth)
	// these are added automatically:
	// Set("Content-Type", "application/json; charset=utf-8")
	// Set("Accept", "application/json; charset=utf-8")
	Header http.Header

	// closeReq will close the request body immediately allowing for reuse of http client
	closeReq bool

	// Inspect contains info about the Req and Res of the last Run
	// is resets on the beginning of every client.Run()
	InspectRun map[string]interface{}
}

// InspectData provides info about request and response
type InspectData map[string]interface{}

// type InspectData struct {
// 	ReqHeaders       http.Header
// 	ReqBody          *RequestBody
// 	ResStatusCode    int
// 	ResHeaders       http.Header
// 	ResCookies       []*http.Cookie
// 	ResContentLength int64
// 	ResBody          string
// }

// NewClient makes a new Client capable of making GraphQL requests.
func NewClient(endpoint string, opts ...ClientOption) *Client {
	c := &Client{
		Endpoint: endpoint,
	}

	// Default headers.
	c.Header = make(map[string][]string)
	c.Header.Set("Content-Type", "application/json; charset=utf-8")
	c.Header.Set("Accept", "application/json; charset=utf-8")

	for _, optFn := range opts {
		optFn(c)
	}

	if c.httpClient == nil {
		c.httpClient = http.DefaultClient
	}
	return c
}

// RequestBody reflect the structure of GraphQL Request
type RequestBody struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	OperationName *string                `json:"operationName"`
}

// Run executes the query and unmarshals the response from the data field
// into the response object.
// Pass in a nil response object to skip response parsing.
func (c *Client) Run(ctx context.Context, req *Request, resp interface{}) ([]GraphQLError, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// here should be done some releasing/resetting the logging headers etc..
	c.InspectRun = InspectData{}

	var requestBody bytes.Buffer
	requestBodyObj := RequestBody{
		Query:         req.q,
		Variables:     req.vars,
		OperationName: req.opName,
	}

	// adding for possibility to inspect
	c.InspectRun["ReqBody"] = &requestBodyObj

	if err := json.NewEncoder(&requestBody).Encode(requestBodyObj); err != nil {
		return nil, errors.Wrap(err, "encode body")
	}

	r, err := http.NewRequest(http.MethodPost, c.Endpoint, &requestBody)
	if err != nil {
		return nil, err
	}
	r.Close = c.closeReq

	// Adds headers defined on the client.
	for key, values := range c.Header {
		for _, value := range values {
			r.Header.Add(key, value)
		}
	}

	// Adds headers defined on the current request.
	for key, values := range req.Header {
		for _, value := range values {
			r.Header.Add(key, value)
		}
	}
	c.InspectRun["ReqHeaders"] = r.Header

	r = r.WithContext(ctx)
	res, err := c.httpClient.Do(r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// capture for inspection
	c.InspectRun["ResStatusCode"] = res.StatusCode
	c.InspectRun["ResHeaders"] = res.Header
	c.InspectRun["ResCookies"] = res.Cookies() // should return func to be consistent with http?
	c.InspectRun["ResContentLength"] = res.ContentLength

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP Error %v: graphql server returned a non-200 status code", res.StatusCode)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, res.Body); err != nil {
		return nil, errors.Wrap(err, "reading body")
	}

	// capture for inspection
	c.InspectRun["ResBody"] = buf.String()

	var gr GraphQLResponse

	// inject own type
	gr.Data = resp

	if err := json.NewDecoder(&buf).Decode(&gr); err != nil {
		return nil, errors.Wrap(err, "decoding response")
	}

	if len(gr.Errors) > 0 {
		return gr.Errors, nil
	}
	return nil, nil
}

// Vars sets a variables for the request
func (req *Request) Vars(vars map[string]interface{}) {
	if req.vars == nil {
		req.vars = make(map[string]interface{})
	}
	req.vars = vars
}

// -----------------------------
// Functional options for client

// WithHTTPClient specifies the underlying http.Client to use when
// making requests.
func WithHTTPClient(httpclient *http.Client) ClientOption {
	return func(client *Client) {
		client.httpClient = httpclient
	}
}

// WithHeaders specifies headers to use with every request.
// These headers are Added not Set.
// If you want to Set them, use client.Header.Set(...)
// Following are added automatically, you don't need to set them.
// Set("Content-Type", "application/json; charset=utf-8")
// Set("Accept", "application/json; charset=utf-8")
func WithHeaders(headers http.Header) ClientOption {
	return func(client *Client) {
		for key, values := range headers {
			for _, value := range values {
				client.Header.Add(key, value)
			}
		}
	}
}

//ImmediatelyCloseReqBody will close the req body immediately after each request body is ready
func ImmediatelyCloseReqBody() ClientOption {
	return func(client *Client) {
		client.closeReq = true
	}
}

// ClientOption are functions that are passed into NewClient to
// modify the behaviour of the Client.
type ClientOption func(*Client)

// ------
// Errors

// GraphQLError comply with specs
// "message": "Name for character with ID 1002 could not be fetched.", // string
// "locations": [ { "line": 6, "column": 7 } ], // []struct{ line int, column int }
// "path": [ "hero", "heroFriends", 1, "name" ] // []interface{}
// "extensions": { // this is map[string]interface{}
// 		"code": "CAN_NOT_FETCH_BY_ID",
// 		"timestamp": "Fri Feb 9 14:33:09 UTC 2018"
// }
type GraphQLError struct {
	Message   string `json:"message"`
	Locations []struct {
		Line   int `json:"Line"`
		Column int `json:"Column"`
	} `json:"locations"`
	Path       []interface{}          `json:"path"`
	Extensions map[string]interface{} `json:"extensions"`
}

// Error satisfies the Error interface
func (e *GraphQLError) Error() string {
	return fmt.Sprintf("graphql error: %s, on path %v", e.Message, e.Path)
}

// --------------------
// Request and response

// GraphQLResponse describes GraphQL response
type GraphQLResponse struct {
	Data   interface{}    `json:"data"`
	Errors []GraphQLError `json:"errors"`
}

// Request is a GraphQL request.
type Request struct {
	q      string
	vars   map[string]interface{}
	opName *string

	// Header represent any request headers that will be set
	// when the request is made.
	Header http.Header
}

// NewRequest makes a new Request with the specified string.
// TODO allow inserting operation name
// (which operation to run in multioperation documents)
func NewRequest(q string, opts ...RequestOption) *Request {
	req := &Request{
		q:      q,
		Header: http.Header{},
	}

	for _, optFn := range opts {
		optFn(req)
	}

	return req
}

// Request's functional options

// RequestOption is a functional option
type RequestOption func(*Request)

// WithVars allows to specify variables for the request
func WithVars(vars map[string]interface{}) RequestOption {
	return func(r *Request) {
		r.vars = vars
	}
}

// WithOperationName allows to specify request's operation name
func WithOperationName(s string) RequestOption {
	return func(r *Request) {
		if s != "" {
			r.opName = &s
		}
	}
}
