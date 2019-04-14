# graphqlgo [![GoDoc](https://godoc.org/github.com/dacz/graphqlgo?status.png)](https://godoc.org/github.com/dacz/graphqlgo) [![Go Report Card](https://goreportcard.com/badge/github.com/dacz/graphqlgo)](https://goreportcard.com/report/github.com/dacz/graphqlgo) [![Build](https://travis-ci.org/dacz/graphqlgo.svg?branch=master)](https://travis-ci.org/dacz/graphqlgo)

GraphQL client package for Go. ALPHA!

**THIS IS WORK IN PROGRESS, API IS UNSTABLE!**

- Respects `context.Context` timeouts and cancellation
- Build and execute GraphQL request (but doesn't support subscriptions)
- Use variables
- Uses GraphQL standard features
- Returns data and all GraphQL errors

The graphqlgo.Client is not concurrent safe and not intended to be. If you need to make concurrent requests, instantiate separate clients for every one of them.

The Client keeps it's state (as request headers, data sent, received headers etc.). The request and response it can be inspected after the request. Every client.Run resets this state therefore you can use the client for multiple non-concurrent requests.

## Usage

```
$ go get github.com/dacz/graphqlgo
```

Options for Client:

- own http client (`WithHTTPClient`)
- http headers (`WithHeaders`) which will be used for every request by the client. The headers `"Content-Type": "application/json; charset=utf-8"` and `"Accept": "application/json; charset=utf-8"` are specified by default, you don't need to set them
- to close http request body immediatelly (`ImmediatelyCloseReqBody`)

Adding or setting headers on existing client: `client.Header.Add` or `client.Header.Set` (same as `http.Header`).

Options for the Request:

- specify graphql variables (`WithVars`)
- specify opration name (`WithOperationName`)
- add request specific headers with instantiated request as `request.Header.Add` or `request.Header.Set` (same as `http.Header`)

## Example

See the docs or [this file](./example_test.go).

## Credits

Forked from [graphql by Machinebox](https://github.com/machinebox/graphql).

The reasons I forked this repo:

- no multipart messages (I consider uploading file to graphql server to be an antipattern)
- API slightly different (to return all errors and errors with errors.extensions)
- introspection of the request and response instead of logging
