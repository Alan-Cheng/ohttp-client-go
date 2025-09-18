package main

import (
	"net/http"
	"github.com/Alan-Cheng/ohttp-client-go/ohttpclient"
	"fmt"
)

func main() {
	// your target url, you can use any type of HTTP methods
	req, err := http.NewRequest("GET", "https://example.com", nil)
	if err != nil {
		panic(err)
	}

	// OHTTP gateway URL and gateway config endpoint
	resp, err := ohttpclient.DoRequest(ohttpclient.Config{
		GatewayURL: "https://example.com/ohttp/gateway",
		KeysURL:    "https://example.com/ohttp/ohttp-configs",
		Verbose:    true,
		Request:    req,
	})

	if err != nil {
		panic(err)
	}

	// print response status and body
	fmt.Println("Status:", resp.Status)
	fmt.Println("Body:", string(resp.Body))
}