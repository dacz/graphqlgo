package graphqlgo

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
)

func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}

func TestRunSimple(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost {
			t.Errorf("Should be POST method, but is %s\n", r.Method)
		}

		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal("ReadAll should not return error:", err)
		}

		wantQuery := `{"query":"query {}","variables":null,"operationName":null}`
		gotQuery := strings.TrimSpace(string(b))
		if gotQuery != wantQuery {
			t.Errorf("Wanted %q, got %q", wantQuery, gotQuery)
		}

		_, err = io.WriteString(w, `{
			"data": {
				"something": "yes"
			}
		}`)
		if err != nil {
			t.Errorf("Reasponse write should not error: %v", err)
		}
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	var responseData map[string]string
	gqerr, err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	if err != nil {
		t.Errorf("clientRun should not return error: %v", err)
	}

	if responseData["something"] != "yes" {
		t.Errorf("I wanted some response data but got:\n%s\n", prettyPrint(responseData))
	}
	if gqerr != nil {
		t.Errorf("There should be no graphql errors: \n%s\n", prettyPrint(gqerr))
	}

	if calls != 1 {
		t.Errorf("There should be only 1 call: %d", calls)
	}
}

func TestRunWithOpts(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost {
			t.Errorf("Should be POST method, but is %s\n", r.Method)
		}

		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal("ReadAll should not return error:", err)
		}

		wantQuery := `{"query":"query name($some:ID!) {some(some:$some){else}}","variables":{"some":5},"operationName":"name"}`
		gotQuery := strings.TrimSpace(string(b))
		if gotQuery != wantQuery {
			t.Errorf("Wanted %q, got %q", wantQuery, gotQuery)
		}

		_, err = io.WriteString(w, `{
			"data": {
				"something": "yes"
			}
		}`)
		if err != nil {
			t.Errorf("Reasponse write should not error: %v", err)
		}
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL, WithHeaders(http.Header{"X-Some-Fromcli": {"cliVal"}}))

	qry := "query name($some:ID!) {some(some:$some){else}}"
	vars := map[string]interface{}{"some": 5}

	req := NewRequest(qry, WithVars(vars), WithOperationName("name"))
	req.Header.Add("X-Some-Fromreq", "reqVal")

	ctx1, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	var responseData map[string]string
	gqerr, err := client.Run(ctx1, req, &responseData)
	if err != nil {
		t.Errorf("clientRun should not return error: %v", err)
	}

	// t.Log(prettyPrint(client.InspectRun))
	if responseData["something"] != "yes" {
		t.Errorf("I wanted some response data but got:\n%s\n", prettyPrint(responseData))
	}
	if gqerr != nil {
		t.Errorf("There should be no graphql errors: \n%s\n", prettyPrint(gqerr))
	}

	// t.Log(prettyPrint(client.InspectRun))
	reqHeaders, ok := client.InspectRun["ReqHeaders"].(http.Header)
	if !ok {
		t.Fatal("Request headers should be able to typecast")
	}

	val, ok := reqHeaders["X-Some-Fromcli"]
	if !ok {
		t.Error("Request should contain header 'X-Some-Fromcli'")
	}
	if val[0] != "cliVal" {
		t.Errorf("Header 'X-Some-Fromcli' should be 'cliVal'. This header is %#v\n", val)
	}

	val, ok = reqHeaders["X-Some-Fromreq"]
	if !ok {
		t.Error("Request should contain header 'X-Some-Fromreq'")
	}
	if val[0] != "reqVal" {
		t.Errorf("Header 'X-Some-Fromreq' should be 'reqVal'. This header is %#v\n", val)
	}

	if calls != 1 {
		t.Errorf("There should be only 1 call: %d", calls)
	}

	// next request should have only headers defined on client
	req2 := NewRequest(qry, WithVars(vars), WithOperationName("name"))

	ctx2, cancel2 := context.WithTimeout(ctx, 1*time.Second)
	defer cancel2()

	_, err = client.Run(ctx2, req2, nil)
	if err != nil {
		t.Errorf("clientRun should not return error: %v", err)
	}

	reqHeaders, ok = client.InspectRun["ReqHeaders"].(http.Header)
	if !ok {
		t.Fatal("Request headers should be able to typecast")
	}

	_, ok = reqHeaders["X-Some-Fromreq"]
	if ok {
		t.Errorf("There should not be 'X-Some-Fromreq' in the query but is. All headers: %s\n", prettyPrint(client.InspectRun["ReqHeaders"]))
	}
}

func TestRunWithGraphQLErrors(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost {
			t.Errorf("Should be POST method, but is %s\n", r.Method)
		}

		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal("ReadAll should not return error:", err)
		}

		wantQuery := `{"query":"query {}","variables":null,"operationName":null}`
		gotQuery := strings.TrimSpace(string(b))
		if gotQuery != wantQuery {
			t.Errorf("Wanted %q, got %q", wantQuery, gotQuery)
		}

		_, err = io.WriteString(w, `{
			"data": {
				"something": "yes"
			},
			"errors": [
				{
					"message": "oops",
					"locations": [{"line": 1, "column": 2}],
					"path": ["some", 1],
					"extensions": {
						"code": "failcode"
					}
				}
			]
		}`)
		if err != nil {
			t.Errorf("Response write should not error: %v", err)
		}
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	var responseData map[string]string
	gqerr, err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	if err != nil {
		t.Errorf("clientRun should not return error: %v", err)
	}

	if responseData["something"] != "yes" {
		t.Errorf("I wanted some response data but got:\n%s\n", prettyPrint(responseData))
	}

	if gqerr == nil {
		t.Errorf("There should be graphql errors but were not: %#v\n", gqerr)
	}

	if len(gqerr) != 1 {
		t.Errorf("There should be 1 graphql error, but there are more: %s\n", prettyPrint(gqerr))
	}

	gqlerr := gqerr[0]
	if gqlerr.Message != "oops" {
		t.Errorf("GraphQL error message shoudl be 'oops', but is %s", gqlerr.Message)
	}
	if gqlerr.Extensions["code"] != "failcode" {
		t.Errorf("GraphQL error extensions.code should be 'failcode', but is not. The extensions are %s", prettyPrint(gqlerr.Extensions))
	}

	if calls != 1 {
		t.Errorf("There should be only 1 call: %d", calls)
	}
}

func TestBadRequest(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++

		w.WriteHeader(http.StatusBadRequest)
		_, err := io.WriteString(w, `{"errors": [{"message": "badbad"}]}`)
		if err != nil {
			t.Errorf("Reasponse write should not error: %v", err)
		}
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	var responseData map[string]string
	gqerr, err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	if err == nil {
		t.Errorf("clientRun should return error")
	}

	if gqerr != nil {
		t.Errorf("Error reply should be nil but is %#v\n", gqerr)
	}
	if responseData != nil {
		t.Errorf("Response data should be nil but is %#v\n", responseData)
	}

	if calls != 1 {
		t.Errorf("There should be only 1 call: %d", calls)
	}
}
