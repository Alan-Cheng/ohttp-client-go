package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/chris-wood/ohttp-go"
	"github.com/spf13/cobra"
)

var (
	gatewayURL string
	targetURL  string
	keysURL    string
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "ohttp-client",
	Short: "OHTTP client tool for making encrypted requests through a gateway",
	Long: `OHTTP client tool that allows you to make encrypted HTTP requests through an OHTTP gateway.
This tool downloads OHTTP public keys, encapsulates your request, sends it through the gateway,
and decrypts the response.`,
	RunE: runOHTTPRequest,
}

func init() {
	rootCmd.Flags().StringVarP(&gatewayURL, "gateway", "g", "", "OHTTP gateway URL (required)")
	rootCmd.Flags().StringVarP(&targetURL, "target", "t", "", "Target URL to request (required)")
	rootCmd.Flags().StringVarP(&keysURL, "keys", "k", "", "OHTTP public keys URL (required)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	
	rootCmd.MarkFlagRequired("gateway")
	rootCmd.MarkFlagRequired("target")
	rootCmd.MarkFlagRequired("keys")
}

func runOHTTPRequest(cmd *cobra.Command, args []string) error {
	if verbose {
		fmt.Printf("Gateway URL: %s\n", gatewayURL)
		fmt.Printf("Target URL: %s\n", targetURL)
		fmt.Printf("Keys URL: %s\n", keysURL)
		fmt.Println()
	}

	// 1) 下載 OHTTP 公鑰
	if verbose {
		fmt.Printf("下載 OHTTP 金鑰檔案: %s\n", keysURL)
	}
	resp, err := http.Get(keysURL)
	if err != nil {
		return fmt.Errorf("下載金鑰檔案失敗: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下載金鑰檔案非 200 狀態: %d", resp.StatusCode)
	}
	keyConfigBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("讀取金鑰檔案失敗: %w", err)
	}

	// 2) 解析 PublicConfig
	publicConfig, err := ohttp.UnmarshalPublicConfig(keyConfigBytes)
	if err != nil {
		return fmt.Errorf("解析金鑰配置失敗: %w", err)
	}
	if verbose {
		fmt.Println("成功解析金鑰配置。")
	}

	// 3) 建立內層 HTTP 請求並轉成 BHTTP
	targetReq, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return fmt.Errorf("建立 target 請求失敗: %w", err)
	}
	
	// 從URL解析host
	parsedURL, err := http.ParseRequestURI(targetURL)
	if err != nil {
		return fmt.Errorf("解析目標URL失敗: %w", err)
	}
	targetReq.Host = parsedURL.Host

	// 轉為 BHTTP
	binaryReq := ohttp.BinaryRequest(*targetReq)
	breqBytes, err := binaryReq.Marshal()
	if err != nil {
		return fmt.Errorf("序列化 BHTTP 失敗: %w", err)
	}
	if verbose {
		fmt.Println("已建立 BHTTP 內層請求。")
	}

	// 4) 封裝為 OHTTP
	client := ohttp.NewDefaultClient(publicConfig)
	encReq, ctx, err := client.EncapsulateRequest(breqBytes)
	if err != nil {
		return fmt.Errorf("封裝請求失敗: %w", err)
	}
	if verbose {
		fmt.Println("已成功封裝 OHTTP 請求。")
	}

	// 5) 發送到 Gateway
	relayRequest, err := http.NewRequest(http.MethodPost, gatewayURL, bytes.NewReader(encReq.Marshal()))
	if err != nil {
		return fmt.Errorf("建立外層 POST 請求失敗: %w", err)
	}
	relayRequest.Header.Set("Content-Type", "message/ohttp-req")

	relayResp, err := http.DefaultClient.Do(relayRequest)
	if err != nil {
		return fmt.Errorf("發送到 Gateway 失敗: %w", err)
	}
	defer relayResp.Body.Close()
	if relayResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(relayResp.Body)
		return fmt.Errorf("Gateway 外層狀態非 200: %d, body=%s", relayResp.StatusCode, string(body))
	}

	// 6) 解封 OHTTP 回應 → 取得 BHTTP bytes
	encRespBytes, err := io.ReadAll(relayResp.Body)
	if err != nil {
		return fmt.Errorf("讀取外層回應失敗: %w", err)
	}
	encResp, err := ohttp.UnmarshalEncapsulatedResponse(encRespBytes)
	if err != nil {
		return fmt.Errorf("解析封裝回應失敗: %w", err)
	}
	brespBytes, err := ctx.DecapsulateResponse(encResp)
	if err != nil {
		return fmt.Errorf("解密回應失敗: %w", err)
	}

	// 7) BHTTP → http.Response
	httpResp, err := ohttp.UnmarshalBinaryResponse(brespBytes)
	if err != nil {
		return fmt.Errorf("BHTTP 轉回 http.Response 失敗: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("讀取內層回應失敗: %w", err)
	}

	fmt.Println("成功解密回應！")
	fmt.Println("--------------------")
	fmt.Printf("Status: %d\n", httpResp.StatusCode)
	fmt.Println(string(body))
	fmt.Println("--------------------")
	
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
