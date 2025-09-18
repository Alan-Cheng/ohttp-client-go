# OHTTP Client Go

A command-line tool and Go library for testing Oblivious HTTP (OHTTP) gateways, specifically designed to work with [Cloudflare Privacy Gateway Server](https://github.com/cloudflare/privacy-gateway-server-go).It allows sending OHTTP requests either via the command-line tool or by importing this library into your Go project.

用於測試 Oblivious HTTP (OHTTP) Gateway 的指令工具和 Go 函式庫，設計目的是用於與[Cloudflare Privacy Gateway Server](https://github.com/cloudflare/privacy-gateway-server-go)互動。
你可以透過命令列工具或將此函式庫引入專案，來送出 OHTTP 請求。

## Background

Cloudflare has open-sourced their [OHTTP gateway server](https://github.com/cloudflare/privacy-gateway-server-go) but doesn't provide a client for testing OHTTP gateways. This project fills that gap by providing a curl-like command-line tool and Go library for testing OHTTP functionality.

Cloudflare 開源了[Cloudflare Privacy Gateway Server](https://github.com/cloudflare/privacy-gateway-server-go)，但沒有提供用於測試的客戶端。本專案填補了這個空缺，提供了一個類似 curl 的命令列工具以及 Go 函式庫，用於測試 OHTTP 功能。
## Features

- **Command-line tool** (`ohttpc`) - curl-like interface for OHTTP requests
- **Go library** - programmatic access to OHTTP functionality
- **Full OHTTP support** - handles key fetching, request encapsulation, and response decapsulation
- **Verbose mode** - detailed logging for debugging
- **JSON pretty printing** - formatted JSON responses
- **Flexible HTTP methods** - GET, POST, PUT, DELETE, etc.

## Installation

### Prerequisites

- Go 1.21 or later

### Build from source

```bash
git clone https://github.com/Alan-Cheng/ohttp-client-go.git
cd ohttp-client-go
go mod tidy
go install ./cmd/...
```

## Usage

### Command-line tool (`ohttpc`)

The `ohttpc` tool provides a curl-like interface for making OHTTP requests.

#### Basic syntax

```bash
ohttpc [flags] URL
```

#### Required flags

- `-g, --gateway`: OHTTP gateway URL (required)
- `-k, --keys`: OHTTP public keys(config) URL (required)

#### Optional flags

- `-v, --verbose`: Enable verbose output
- `-X, --request`: HTTP method (default: GET)
- `-d, --data`: HTTP request body
- `-H, --header`: Custom headers (can be used multiple times)
- `-j, --json`: Pretty print JSON responses
- `-t, --target`: Target URL (alternative to positional argument)

#### Examples

**Simple GET request:**
```bash
ohttpc -g https://gateway.example.com/gateway -k https://gateway.example.com/ohttp-configs https://httpbin.org/get
```

**POST request with JSON data:**
```bash
ohttpc -g https://gateway.example.com/gateway -k https://gateway.example.com/ohttp-configs \
  -X POST -d '{"name":"test"}' -H "Content-Type: application/json" https://httpbin.org/post
```

**Verbose output:**
```bash
ohttpc -g https://gateway.example.com/gateway -k https://gateway.example.com/ohttp-configs \
  -v https://httpbin.org/get
```

**Pretty print JSON:**
```bash
ohttpc -g https://gateway.example.com/gateway -k https://gateway.example.com/ohttp-configs \
  -j https://httpbin.org/json
```

**Using target flag:**
```bash
ohttpc -g https://gateway.example.com/gateway -k https://gateway.example.com/ohttp-configs -t https://httpbin.org/get
```

### Go Library

The `ohttpclient` package provides programmatic access to OHTTP functionality.

#### Basic usage

```go
package main

import (
    "net/http"
    "github.com/Alan-Cheng/ohttp-client-go/ohttpclient"
)

func main() {
    // Create HTTP request
    req, _ := http.NewRequest("GET", "https://httpbin.org/get", nil)
    
    // Make OHTTP request
    resp, err := ohttpclient.DoRequest(ohttpclient.Config{
        GatewayURL: "https://gateway.example.com/gateway",
        KeysURL:    "https://gateway.example.com/ohttp-configs",
        Verbose:    true,
        Request:    req,
    })
    
    if err != nil {
        panic(err)
    }
    
    println("Status:", resp.Status)
    println("Body:", string(resp.Body))
}
```

#### POST request with JSON

```go
package main

import (
    "bytes"
    "net/http"
    "github.com/Alan-Cheng/ohttp-client-go/ohttpclient"
)

func main() {
    // Create POST request with JSON data
    jsonData := []byte(`{"message": "Hello OHTTP"}`)
    req, _ := http.NewRequest("POST", "https://httpbin.org/post", bytes.NewReader(jsonData))
    req.Header.Set("Content-Type", "application/json")
    
    // Make OHTTP request
    resp, err := ohttpclient.DoRequest(ohttpclient.Config{
        GatewayURL: "https://gateway.example.com/gateway",
        KeysURL:    "https://gateway.example.com/ohttp-configs",
        Verbose:    false,
        Request:    req,
    })
    
    if err != nil {
        panic(err)
    }
    
    println("Status:", resp.Status)
    println("Body:", string(resp.Body))
}
```

#### Request with custom headers

```go
package main

import (
    "net/http"
    "github.com/Alan-Cheng/ohttp-client-go/ohttpclient"
)

func main() {
    req, _ := http.NewRequest("GET", "https://httpbin.org/headers", nil)
    req.Header.Set("User-Agent", "OHTTP-Client-Go/1.0")
    req.Header.Set("X-Custom-Header", "test-value")
    
    resp, err := ohttpclient.DoRequest(ohttpclient.Config{
        GatewayURL: "https://gateway.example.com/gateway",
        KeysURL:    "https://gateway.example.com/ohttp-configs",
        Verbose:    false,
        Request:    req,
    })
    
    if err != nil {
        panic(err)
    }
    
    println("Status:", resp.Status)
    println("Body:", string(resp.Body))
}
```

## Testing with Cloudflare's Gateway

You can test this client with Cloudflare's public OHTTP gateway:

```bash
# Test with Cloudflare's public gateway
./ohttpc -g https://gateway.privacy-gateway.cloudflare.com/gateway \
  -k https://gateway.privacy-gateway.cloudflare.com/ohttp-configs \
  -v https://httpbin.org/get
```

## How it works

1. **Fetch OHTTP keys**: Downloads the public key configuration from the keys URL
2. **Create Binary HTTP**: Converts the HTTP request to Binary HTTP (BHTTP) format
3. **Encapsulate request**: Encrypts the BHTTP request using OHTTP
4. **Send to gateway**: Posts the encapsulated request to the OHTTP gateway
5. **Decapsulate response**: Decrypts the response from the gateway
6. **Return result**: Converts back to HTTP response format

## Project Structure

```
├── cmd/ohttpc/          # Command-line tool
│   └── main.go
├── ohttpclient/         # Go library
│   └── client.go
├── examples/ 
│   └── basic_use.go     # Example usage
├── go.mod
├── go.sum
└── README.md
```

## Dependencies

- [ohttp-go](https://github.com/chris-wood/ohttp-go) - OHTTP implementation
- [cobra](https://github.com/spf13/cobra) - CLI framework

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

This project is open source. Please check the license file for details.

## Related Projects

- [Cloudflare Privacy Gateway Server](https://github.com/cloudflare/privacy-gateway-server-go) - The OHTTP gateway server this client is designed to work with
- [OHTTP Specification](https://datatracker.ietf.org/doc/draft-ietf-ohai-ohttp/) - The official OHTTP specification