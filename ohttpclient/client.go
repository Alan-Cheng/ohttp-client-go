package ohttpclient

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/chris-wood/ohttp-go"
)

// Config contains necessary parameters
type Config struct {
	GatewayURL string
	KeysURL    string
	Verbose    bool
	Request    *http.Request // Supports any http.Request
}

// Response wraps the returned data
type Response struct {
	Status int
	Body   []byte
}

// DoRequest executes OHTTP request and returns decrypted result
func DoRequest(cfg Config) (*Response, error) {
	if cfg.Verbose {
		fmt.Printf("Gateway URL: %s\nTarget URL: %s\nKeys URL: %s\n\n", cfg.GatewayURL, cfg.Request.URL.String(), cfg.KeysURL)
	}

	// 1) Download OHTTP public key
	if cfg.Verbose {
		fmt.Printf("Downloading OHTTP key file: %s\n", cfg.KeysURL)
	}
	resp, err := http.Get(cfg.KeysURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download key file: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("key file download returned non-200 status: %d", resp.StatusCode)
	}
	keyConfigBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	// 2) Parse PublicConfig
	publicConfig, err := ohttp.UnmarshalPublicConfig(keyConfigBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse key config: %w", err)
	}
	if cfg.Verbose {
		fmt.Println("Successfully parsed key config.")
	}

	// 3) Encapsulate inner request as BHTTP
	binaryReq := ohttp.BinaryRequest(*cfg.Request)
	breqBytes, err := binaryReq.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize BHTTP: %w", err)
	}
	if cfg.Verbose {
		fmt.Println("Created BHTTP inner request.")
	}

	// 4) Encapsulate OHTTP
	client := ohttp.NewDefaultClient(publicConfig)
	encReq, ctx, err := client.EncapsulateRequest(breqBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to encapsulate request: %w", err)
	}
	if cfg.Verbose {
		fmt.Println("Successfully encapsulated OHTTP request.")
	}

	// 5) Send to Gateway
	relayRequest, err := http.NewRequest(http.MethodPost, cfg.GatewayURL, bytes.NewReader(encReq.Marshal()))
	if err != nil {
		return nil, fmt.Errorf("failed to create outer POST request: %w", err)
	}
	relayRequest.Header.Set("Content-Type", "message/ohttp-req")

	relayResp, err := http.DefaultClient.Do(relayRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to send to Gateway: %w", err)
	}
	defer relayResp.Body.Close()
	if relayResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(relayResp.Body)
		return nil, fmt.Errorf("Gateway outer status not 200: %d, body=%s", relayResp.StatusCode, string(body))
	}

	// 6) Decapsulate OHTTP response
	encRespBytes, err := io.ReadAll(relayResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read outer response: %w", err)
	}
	encResp, err := ohttp.UnmarshalEncapsulatedResponse(encRespBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse encapsulated response: %w", err)
	}
	brespBytes, err := ctx.DecapsulateResponse(encResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt response: %w", err)
	}

	httpResp, err := ohttp.UnmarshalBinaryResponse(brespBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert BHTTP back to http.Response: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read inner response: %w", err)
	}

	if cfg.Verbose {
		fmt.Println("Successfully decrypted response!")
	}

	return &Response{
		Status: httpResp.StatusCode,
		Body:   body,
	}, nil
}
