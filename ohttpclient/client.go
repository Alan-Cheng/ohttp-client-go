package ohttpclient

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/chris-wood/ohttp-go"
)

// Config 用來傳入必要參數
type Config struct {
	GatewayURL string
	KeysURL    string
	Verbose    bool
	Request    *http.Request // 支援傳入任何 http.Request
}

// Response 包裝回傳資料
type Response struct {
	Status int
	Body   []byte
}

// DoRequest 執行 OHTTP 請求並回傳解密後結果
func DoRequest(cfg Config) (*Response, error) {
	if cfg.Verbose {
		fmt.Printf("Gateway URL: %s\nTarget URL: %s\nKeys URL: %s\n\n", cfg.GatewayURL, cfg.Request.URL.String(), cfg.KeysURL)
	}

	// 1) 下載 OHTTP 公鑰
	if cfg.Verbose {
		fmt.Printf("下載 OHTTP 金鑰檔案: %s\n", cfg.KeysURL)
	}
	resp, err := http.Get(cfg.KeysURL)
	if err != nil {
		return nil, fmt.Errorf("下載金鑰檔案失敗: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下載金鑰檔案非 200 狀態: %d", resp.StatusCode)
	}
	keyConfigBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("讀取金鑰檔案失敗: %w", err)
	}

	// 2) 解析 PublicConfig
	publicConfig, err := ohttp.UnmarshalPublicConfig(keyConfigBytes)
	if err != nil {
		return nil, fmt.Errorf("解析金鑰配置失敗: %w", err)
	}
	if cfg.Verbose {
		fmt.Println("成功解析金鑰配置。")
	}

	// 3) 封裝內層請求為 BHTTP
	binaryReq := ohttp.BinaryRequest(*cfg.Request)
	breqBytes, err := binaryReq.Marshal()
	if err != nil {
		return nil, fmt.Errorf("序列化 BHTTP 失敗: %w", err)
	}
	if cfg.Verbose {
		fmt.Println("已建立 BHTTP 內層請求。")
	}

	// 4) 封裝 OHTTP
	client := ohttp.NewDefaultClient(publicConfig)
	encReq, ctx, err := client.EncapsulateRequest(breqBytes)
	if err != nil {
		return nil, fmt.Errorf("封裝請求失敗: %w", err)
	}
	if cfg.Verbose {
		fmt.Println("已成功封裝 OHTTP 請求。")
	}

	// 5) 發送到 Gateway
	relayRequest, err := http.NewRequest(http.MethodPost, cfg.GatewayURL, bytes.NewReader(encReq.Marshal()))
	if err != nil {
		return nil, fmt.Errorf("建立外層 POST 請求失敗: %w", err)
	}
	relayRequest.Header.Set("Content-Type", "message/ohttp-req")

	relayResp, err := http.DefaultClient.Do(relayRequest)
	if err != nil {
		return nil, fmt.Errorf("發送到 Gateway 失敗: %w", err)
	}
	defer relayResp.Body.Close()
	if relayResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(relayResp.Body)
		return nil, fmt.Errorf("Gateway 外層狀態非 200: %d, body=%s", relayResp.StatusCode, string(body))
	}

	// 6) 解封 OHTTP 回應
	encRespBytes, err := io.ReadAll(relayResp.Body)
	if err != nil {
		return nil, fmt.Errorf("讀取外層回應失敗: %w", err)
	}
	encResp, err := ohttp.UnmarshalEncapsulatedResponse(encRespBytes)
	if err != nil {
		return nil, fmt.Errorf("解析封裝回應失敗: %w", err)
	}
	brespBytes, err := ctx.DecapsulateResponse(encResp)
	if err != nil {
		return nil, fmt.Errorf("解密回應失敗: %w", err)
	}

	httpResp, err := ohttp.UnmarshalBinaryResponse(brespBytes)
	if err != nil {
		return nil, fmt.Errorf("BHTTP 轉回 http.Response 失敗: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("讀取內層回應失敗: %w", err)
	}

	if cfg.Verbose {
		fmt.Println("成功解密回應！")
	}

	return &Response{
		Status: httpResp.StatusCode,
		Body:   body,
	}, nil
}
