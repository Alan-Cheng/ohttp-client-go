package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Alan-Cheng/ohttp-client-go/ohttpclient"
	"github.com/spf13/cobra"
)

var (
	gatewayURL  string
	keysURL     string
	verbose     bool
	method      string
	data        string
	headers     []string
	prettyJSON  bool
	targetURL   string
	targetFlag  string
)

var rootCmd = &cobra.Command{
	Use:   "ohttpc [flags] URL",
	Short: "OHTTP curl-like client",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// 使用 -t 指定 URL，否則取最後參數
		if targetFlag != "" {
			targetURL = targetFlag
		} else if len(args) > 0 {
			targetURL = args[0]
		} else {
			return fmt.Errorf("未指定目標 URL")
		}

		// 建立 body reader
		var bodyReader *bytes.Reader
		if data != "" {
			bodyReader = bytes.NewReader([]byte(data))
		} else {
			bodyReader = bytes.NewReader(nil)
		}

		// 建立 HTTP request
		req, err := http.NewRequest(method, targetURL, bodyReader)
		if err != nil {
			return fmt.Errorf("建立請求失敗: %w", err)
		}

		// 設定 Headers
		for _, h := range headers {
			parts := strings.SplitN(h, ":", 2)
			if len(parts) == 2 {
				req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}
		}

		// 呼叫 OHTTP library
		resp, err := ohttpclient.DoRequest(ohttpclient.Config{
			GatewayURL: gatewayURL,
			KeysURL:    keysURL,
			Verbose:    verbose,
			Request:    req,
		})
		if err != nil {
			return err
		}

		// verbose 輸出 request
		if verbose {
			fmt.Printf("> %s %s\n", method, targetURL)
			for k, v := range req.Header {
				fmt.Printf("> %s: %s\n", k, strings.Join(v, ","))
			}
			fmt.Println()
		}

		// pretty print JSON
		if prettyJSON {
			var buf bytes.Buffer
			if err := json.Indent(&buf, resp.Body, "", "  "); err == nil {
				fmt.Printf("< HTTP %d\n%s\n", resp.Status, buf.String())
				return nil
			}
		}

		// 普通輸出
		fmt.Printf("< HTTP %d\n", resp.Status)
		fmt.Println(string(resp.Body))
		return nil
	},
}

func init() {
	rootCmd.Flags().StringVarP(&gatewayURL, "gateway", "g", "", "OHTTP gateway URL (required)")
	rootCmd.Flags().StringVarP(&keysURL, "keys", "k", "", "OHTTP public keys URL (required)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().StringVarP(&method, "request", "X", "GET", "HTTP method")
	rootCmd.Flags().StringVarP(&data, "data", "d", "", "HTTP body")
	rootCmd.Flags().StringArrayVarP(&headers, "header", "H", nil, "Custom header")
	rootCmd.Flags().BoolVarP(&prettyJSON, "json", "j", false, "Pretty print JSON response if applicable")

	// 新增 -t / --target 別名
	rootCmd.Flags().StringVarP(&targetFlag, "target", "t", "", "Target URL (alias)")

	rootCmd.MarkFlagRequired("gateway")
	rootCmd.MarkFlagRequired("keys")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
