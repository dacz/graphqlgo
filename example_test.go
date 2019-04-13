package graphqlgo

import (
	"context"
	"fmt"
	"log"
)

func ExampleClient_Run() {
	// this header will be added to every request by this client
	myHeader := map[string][]string{"X-My-Header": {"someValue"}}
	client := NewClient("https://countries.trevorblades.com/", WithHeaders(myHeader))

	vars := map[string]interface{}{
		"code": "AF",
	}

	req := NewRequest(`
	query continent($code: String!) {
		continent(code: $code) {
			code
			name
		}
	}
	`, WithVars(vars))

	// define a Context for the request
	ctx := context.Background()

	var respData struct {
		Continent struct {
			Code string `json:"code"`
			Name string `json:"name"`
		}
	}
	// note: to get map[string]interface{} use var respData interface{}

	gqlerr, err := client.Run(ctx, req, &respData)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("DATA\n%v\n", respData)
	fmt.Println("--------------")
	fmt.Printf("ERRORS\n%v\n", gqlerr)
	// you can inspect request and response
	// fmt.Println(prettyPrint(client.InspectRun))
	// Output:
	// DATA
	// {{AF Africa}}
	// --------------
	// ERRORS
	// []
}
