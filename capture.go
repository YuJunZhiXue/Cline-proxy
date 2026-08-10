package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	captureLogDir string
	captureIndex  int
)

func init() {
	exe, _ := os.Executable()
	captureLogDir = filepath.Join(filepath.Dir(exe), "capture-logs")
}

type CaptureEntry struct {
	Step       int               `json:"step"`
	Name       string            `json:"name"`
	Timestamp  string            `json:"timestamp"`
	Method     string            `json:"method"`
	URL        string            `json:"url"`
	ReqHeaders map[string]string `json:"req_headers"`
	ReqBody    string            `json:"req_body"`
	StatusCode int               `json:"status_code"`
	RespHeaders map[string]string `json:"resp_headers"`
	RespBody   string            `json:"resp_body"`
	Cookies    map[string]string `json:"cookies,omitempty"`
	Notes      string            `json:"notes,omitempty"`
}

func captureRequest(name, method, rawURL, reqBody string, headers map[string]string) (*CaptureEntry, error) {
	captureIndex++
	entry := &CaptureEntry{
		Step:       captureIndex,
		Name:       name,
		Timestamp:  time.Now().Format(time.RFC3339Nano),
		Method:     method,
		URL:        rawURL,
		ReqHeaders: headers,
		ReqBody:    reqBody,
		Cookies:    make(map[string]string),
	}

	// Print step header
	fmt.Println("")
	fmt.Println(strings.Repeat("=", 72))
	fmt.Printf("  STEP %d: %s\n", captureIndex, name)
	fmt.Println(strings.Repeat("=", 72))
	fmt.Printf("  URL:    %s\n", rawURL)
	fmt.Printf("  Method: %s\n", method)
	fmt.Println(strings.Repeat("-", 72))

	// Build request
	var bodyReader io.Reader
	if reqBody != "" {
		bodyReader = strings.NewReader(reqBody)
	}
	req, err := http.NewRequest(method, rawURL, bodyReader)
	if err != nil {
		return entry, fmt.Errorf("create request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Dump request
	reqDump, _ := httputil.DumpRequestOut(req, true)
	fmt.Println("--- REQUEST ---")
	fmt.Println(string(reqDump))
	fmt.Println(strings.Repeat("-", 72))

	// Send with verbose output
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
		Timeout: 60 * time.Second,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			fmt.Printf("  [REDIRECT] %d -> %s\n", r.Response.StatusCode, r.URL)
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			// Capture redirect
			captureIndex++
			redirEntry := &CaptureEntry{
				Step:      captureIndex,
				Name:      name + " [REDIRECT]",
				Timestamp: time.Now().Format(time.RFC3339Nano),
				Method:    r.Method,
				URL:       r.URL.String(),
				Cookies:   make(map[string]string),
			}
			allEntries = append(allEntries, redirEntry)
			saveCaptureEntry(redirEntry)
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
		entry.Notes = fmt.Sprintf("ERROR: %v", err)
		allEntries = append(allEntries, entry)
		saveCaptureEntry(entry)
		return entry, err
	}
	defer resp.Body.Close()

	// Record response
	entry.StatusCode = resp.StatusCode
	entry.RespHeaders = make(map[string]string)
	for k, vals := range resp.Header {
		entry.RespHeaders[k] = strings.Join(vals, ", ")
	}

	respBytes, _ := io.ReadAll(resp.Body)
	entry.RespBody = string(respBytes)

	// Extract cookies
	for _, c := range resp.Cookies() {
		entry.Cookies[c.Name] = c.Value
	}

	// Dump response
	fmt.Println("--- RESPONSE ---")
	respDump, _ := httputil.DumpResponse(resp, false)
	fmt.Println(string(respDump))
	fmt.Println("--- RESPONSE BODY ---")
	// Pretty print JSON if possible
	var prettyJSON bytes.Buffer
	if json.Valid(respBytes) {
		json.Indent(&prettyJSON, respBytes, "", "  ")
		fmt.Println(prettyJSON.String())
	} else {
		fmt.Println(entry.RespBody)
	}
	if len(entry.Cookies) > 0 {
		fmt.Println("--- COOKIES ---")
		for k, v := range entry.Cookies {
			fmt.Printf("  %s = %s\n", k, v)
		}
	}
	fmt.Println(strings.Repeat("=", 72))

	allEntries = append(allEntries, entry)
	saveCaptureEntry(entry)
	return entry, nil
}

var allEntries []*CaptureEntry

func saveCaptureEntry(entry *CaptureEntry) {
	os.MkdirAll(captureLogDir, 0755)
	filename := filepath.Join(captureLogDir, fmt.Sprintf("step-%02d-%s.json", entry.Step, sanitizeFilename(entry.Name)))
	data, _ := json.MarshalIndent(entry, "", "  ")
	os.WriteFile(filename, data, 0644)

	// Also append to full log
	fullLog := filepath.Join(captureLogDir, "full-capture.json")
	var entries []*CaptureEntry
	if existing, err := os.ReadFile(fullLog); err == nil {
		json.Unmarshal(existing, &entries)
	}
	if entries == nil {
		entries = make([]*CaptureEntry, 0)
	}
	entries = append(entries, entry)
	finalData, _ := json.MarshalIndent(entries, "", "  ")
	os.WriteFile(fullLog, finalData, 0644)
}

func sanitizeFilename(s string) string {
	r := strings.NewReplacer(
		" ", "-",
		"[", "",
		"]", "",
		"/", "-",
		":", "-",
		"\"", "",
	)
	return r.Replace(s)
}

func doFullCapture() error {
	fmt.Println("")
	fmt.Println("╔" + strings.Repeat("═", 70) + "╗")
	fmt.Println("║         Cline OAuth 完整流量抓包 - 全流程记录                    ║")
	fmt.Println("╚" + strings.Repeat("═", 70) + "╝")
	fmt.Printf("  日志目录: %s\n", captureLogDir)
	fmt.Println("")

	os.MkdirAll(captureLogDir, 0755)
	captureIndex = 0

	// ================================================================
	// PHASE 1: WorkOS Device Authorization
	// ================================================================
	fmt.Println("")
	fmt.Println(strings.Repeat("█", 72))
	fmt.Println("  PHASE 1: WorkOS 设备码认证 - 请求设备码")
	fmt.Println(strings.Repeat("█", 72))

	entry1, err := captureRequest(
		"WorkOS Device Auth - Request Device Code",
		"POST",
		"https://api.workos.com/user_management/authorize/device",
		"client_id=client_01K3A541FN8TA3EPPHTD2325AR",
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
			"User-Agent":   "Cline/3.0.47",
		},
	)
	if err != nil {
		return fmt.Errorf("step 1 failed: %w", err)
	}

	// Parse device auth response
	var deviceResp struct {
		DeviceCode              string `json:"device_code"`
		UserCode                string `json:"user_code"`
		VerificationURI         string `json:"verification_uri"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		Interval                int    `json:"interval"`
		ExpiresIn               int    `json:"expires_in"`
	}
	json.Unmarshal([]byte(entry1.RespBody), &deviceResp)

	// ================================================================
	// PHASE 2: User Interaction
	// ================================================================
	fmt.Println("")
	fmt.Println(strings.Repeat("█", 72))
	fmt.Println("  PHASE 2: 用户授权 (请在浏览器操作)")
	fmt.Println(strings.Repeat("█", 72))
	fmt.Println("")
	fmt.Println("  请打开以下链接并在浏览器中完成授权:")
	fmt.Println("  " + deviceResp.VerificationURIComplete)
	fmt.Println("")
	fmt.Println("  验证码: " + deviceResp.UserCode)
	fmt.Println("")
	fmt.Println("  完成后按 ENTER 继续...")
	fmt.Scanln()

	// ================================================================
	// PHASE 3: Poll WorkOS for token
	// ================================================================
	fmt.Println("")
	fmt.Println(strings.Repeat("█", 72))
	fmt.Println("  PHASE 3: WorkOS Token 轮询")
	fmt.Println(strings.Repeat("█", 72))

	interval := deviceResp.Interval
	if interval < 5 {
		interval = 5
	}
	deadline := time.Now().Add(time.Duration(deviceResp.ExpiresIn) * time.Second)

	var workosAccessToken, workosRefreshToken string

	pollCount := 0
	for time.Now().Before(deadline) {
		pollCount++
		pollBody := fmt.Sprintf(
			"grant_type=urn:ietf:params:oauth:grant-type:device_code&device_code=%s&client_id=client_01K3A541FN8TA3EPPHTD2325AR",
			deviceResp.DeviceCode,
		)

		entry, err := captureRequest(
			fmt.Sprintf("WorkOS Poll Token (attempt %d)", pollCount),
			"POST",
			"https://api.workos.com/user_management/authenticate",
			pollBody,
			map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
				"User-Agent":   "Cline/3.0.47",
			},
		)
		if err != nil {
			fmt.Printf("  Poll error: %v, retrying in %ds...\n", err, interval)
			time.Sleep(time.Duration(interval) * time.Second)
			continue
		}

		if entry.StatusCode == 200 {
			var authResp struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
			}
			json.Unmarshal([]byte(entry.RespBody), &authResp)
			workosAccessToken = authResp.AccessToken
			workosRefreshToken = authResp.RefreshToken
			fmt.Println("\n  ✓ WorkOS 授权成功!")
			break
		}

		// Check if it's pending
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal([]byte(entry.RespBody), &errResp) == nil && errResp.Error == "authorization_pending" {
			fmt.Printf("  ⏳ 等待授权中... (%d秒后重试)\n", interval)
			time.Sleep(time.Duration(interval) * time.Second)
			continue
		}
		if errResp.Error == "slow_down" {
			interval += 5
			fmt.Printf("  🐢 slow_down, 增加间隔到 %ds\n", interval)
			time.Sleep(time.Duration(interval) * time.Second)
			continue
		}
		fmt.Printf("  ⚠ Poll error response: %s, retrying...\n", entry.RespBody)
		time.Sleep(time.Duration(interval) * time.Second)
	}

	if workosAccessToken == "" {
		return fmt.Errorf("failed to get WorkOS token within expiry time")
	}

	// ================================================================
	// PHASE 4: Cline Register
	// ================================================================
	fmt.Println("")
	fmt.Println(strings.Repeat("█", 72))
	fmt.Println("  PHASE 4: Cline 注册 - 交换 WorkOS Token -> Cline Token")
	fmt.Println(strings.Repeat("█", 72))

	registerBody := fmt.Sprintf(
		`{"accessToken":"%s","refreshToken":"%s"}`,
		workosAccessToken, workosRefreshToken,
	)

	entry4, err := captureRequest(
		"Cline Register - Exchange WorkOS Token",
		"POST",
		"https://api.cline.bot/api/v1/auth/register",
		registerBody,
		map[string]string{
			"Content-Type":  "application/json",
			"User-Agent":    "Cline/3.0.47",
			"HTTP-Referer":  "https://cline.bot",
			"X-Title":       "Cline",
			"X-CLIENT-TYPE": "cline-sdk",
			"X-CLIENT-VERSION": "3.0.47",
			"X-PLATFORM":    "terminal",
		},
	)
	if err != nil {
		return fmt.Errorf("step 4 failed: %w", err)
	}

	// Parse Cline register response
	var clineRegResp struct {
		Data struct {
			AccessToken  string `json:"accessToken"`
			RefreshToken string `json:"refreshToken"`
			ExpiresAt    any    `json:"expiresAt"`
			UserInfo     *struct {
				Email string `json:"email"`
			} `json:"userInfo"`
		} `json:"data"`
	}
	json.Unmarshal([]byte(entry4.RespBody), &clineRegResp)

	if clineRegResp.Data.RefreshToken == "" {
		return fmt.Errorf("cline registration did not return refreshToken")
	}

	fmt.Println("")
	fmt.Println("✨✨✨✨✨✨✨✨✨✨✨✨✨✨✨✨✨✨✨✨✨✨✨✨")
	fmt.Println("  🎉 登录成功！关键 Token 如下:")
	fmt.Println("")
	fmt.Println("  📧 Email:       " + getEmail(clineRegResp.Data.UserInfo))
	fmt.Println("  🔑 RefreshToken:")
	fmt.Println("  " + clineRegResp.Data.RefreshToken)
	fmt.Println("")
	fmt.Println("  🔑 AccessToken:")
	fmt.Println("  " + clineRegResp.Data.AccessToken)
	fmt.Println("✨✨✨✨✨✨✨✨✨✨✨✨✨✨✨✨✨✨✨✨✨✨✨✨")
	fmt.Println("")

	// ================================================================
	// PHASE 5: Test Token Refresh
	// ================================================================
	fmt.Println("")
	fmt.Println(strings.Repeat("█", 72))
	fmt.Println("  PHASE 5: 验证 Token Refresh (用拿到的refreshToken续期)")
	fmt.Println(strings.Repeat("█", 72))

	refreshBody := fmt.Sprintf(
		`{"refreshToken":"%s","grantType":"refresh_token"}`,
		clineRegResp.Data.RefreshToken,
	)

	_, err = captureRequest(
		"Cline Token Refresh Verification",
		"POST",
		"https://api.cline.bot/api/v1/auth/refresh",
		refreshBody,
		map[string]string{
			"Content-Type":  "application/json",
			"User-Agent":    "Cline/3.0.47",
			"HTTP-Referer":  "https://cline.bot",
			"X-Title":       "Cline",
			"X-CLIENT-TYPE": "cline-sdk",
			"X-CLIENT-VERSION": "3.0.47",
			"X-PLATFORM":    "terminal",
		},
	)
	if err != nil {
		fmt.Printf("  ⚠ Token refresh test failed: %v\n", err)
		fmt.Println("  (refreshToken still valid for proxy use)")
	} else {
		fmt.Println("  ✓ Token refresh 验证成功!")
	}

	// ================================================================
	// PHASE 6: Test Chat API (optional)
	// ================================================================
	fmt.Println("")
	fmt.Println("  按 ENTER 测试 Chat API 请求 (或 Ctrl+C 跳过)")
	fmt.Scanln()

	fmt.Println("")
	fmt.Println(strings.Repeat("█", 72))
	fmt.Println("  PHASE 6: 测试 Chat API 请求")
	fmt.Println(strings.Repeat("█", 72))

	chatBody := `{
		"model": "cline-free/glm-5.2",
		"messages": [{"role":"user","content":"Hello, say hi"}],
		"max_tokens": 100,
		"stream": false
	}`

	_, err = captureRequest(
		"Cline Chat API Test",
		"POST",
		"https://api.cline.bot/api/v1/chat/completions",
		chatBody,
		map[string]string{
			"Authorization":  "Bearer workos:" + clineRegResp.Data.AccessToken,
			"Content-Type":   "application/json",
			"User-Agent":     "Cline/3.0.47",
			"HTTP-Referer":   "https://cline.bot",
			"X-Title":        "Cline",
			"X-IS-MULTIROOT": "false",
			"X-CLIENT-TYPE":  "cline-sdk",
			"X-CLIENT-VERSION": "3.0.47",
			"X-PLATFORM":     "terminal",
			"X-PLATFORM-VERSION": "3.0.47",
			"X-CORE-VERSION": "0.0.66",
			"X-Task-ID":      "sess_" + fmt.Sprintf("%d", time.Now().UnixMilli()),
		},
	)
	if err != nil {
		fmt.Printf("  ⚠ Chat API test: %v\n", err)
	}

	// ================================================================
	// SUMMARY
	// ================================================================
	fmt.Println("")
	fmt.Println(strings.Repeat("╬", 72))
	fmt.Println("  抓包完成! 所有记录已保存到:")
	fmt.Println("  " + captureLogDir)
	fmt.Println("")
	fmt.Println("  关键文件:")
	fmt.Println("  - " + filepath.Join(captureLogDir, "full-capture.json") + " (完整记录)")
	fmt.Println("  - " + filepath.Join(captureLogDir, "step-*.json") + " (每一步独立文件)")
	fmt.Println("")
	fmt.Println("  重要 Token 汇总:")
	fmt.Println("")
	fmt.Println("  WorkOS AccessToken:")
	fmt.Println("  " + workosAccessToken)
	fmt.Println("")
	fmt.Println("  WorkOS RefreshToken:")
	fmt.Println("  " + workosRefreshToken)
	fmt.Println("")
	fmt.Println("  Cline AccessToken:")
	fmt.Println("  " + clineRegResp.Data.AccessToken)
	fmt.Println("")
	fmt.Println("  Cline RefreshToken (永久保存这个!):")
	fmt.Println("  " + clineRegResp.Data.RefreshToken)
	fmt.Println("")
	fmt.Println("  Cline User Email:")
	fmt.Println("  " + getEmail(clineRegResp.Data.UserInfo))
	fmt.Println(strings.Repeat("╬", 72))

	return nil
}

func getEmail(ui *struct {
	Email string `json:"email"`
}) string {
	if ui != nil && ui.Email != "" {
		return ui.Email
	}
	return "unknown"
}
