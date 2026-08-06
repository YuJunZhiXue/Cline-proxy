package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	workosClientID       = "client_01K3A541FN8TA3EPPHTD2325AR"
	workosDeviceAuthURL  = "https://api.workos.com/user_management/authorize/device"
	workosAuthenticateURL = "https://api.workos.com/user_management/authenticate"
	clineAPIBase         = "https://api.cline.bot/api/v1"
)

type credentials struct {
	RefreshToken string `json:"refreshToken"`
}

type deviceAuthResp struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	Interval                int    `json:"interval"`
	ExpiresIn               int    `json:"expires_in"`
}

type authenticateResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

type clineAuthResp struct {
	Data struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		ExpiresAt    any    `json:"expiresAt"`
		UserInfo     *struct {
			Email string `json:"email"`
		} `json:"userInfo"`
	} `json:"data"`
}

type clineRefreshResp struct {
	Data struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		ExpiresAt    any    `json:"expiresAt"`
	} `json:"data"`
}

var (
	cachedToken      string
	cachedExpiry     int64
	cachedRefreshTok string
	credentialsPath  string
)

func init() {
	credentialsPath = findCredentialsFile()
}

func findCredentialsFile() string {
	// First, try next to the executable
	exe, err := os.Executable()
	if err == nil {
		p := filepath.Join(filepath.Dir(exe), ".cline-credentials.json")
		if fileExists(p) {
			return p
		}
	}
	// Second, try current working directory
	pwd, err := os.Getwd()
	if err == nil {
		p := filepath.Join(pwd, ".cline-credentials.json")
		if fileExists(p) {
			return p
		}
	}
	// Default to executable directory
	if err == nil {
		return filepath.Join(filepath.Dir(exe), ".cline-credentials.json")
	}
	pwd, _ = os.Getwd()
	return filepath.Join(pwd, ".cline-credentials.json")
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func loadCredentials() *credentials {
	data, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil
	}
	var c credentials
	if err := json.Unmarshal(data, &c); err != nil {
		return nil
	}
	return &c
}

func saveCredentials(rt string) {
	c := credentials{RefreshToken: rt}
	data, _ := json.MarshalIndent(c, "", "  ")
	if err := os.WriteFile(credentialsPath, data, 0600); err != nil {
		log.Printf("Failed to save credentials: %v", err)
		return
	}
	log.Printf("Credentials saved to %s", credentialsPath)
}

func workosDeviceAuth() (*deviceAuthResp, error) {
	form := url.Values{"client_id": {workosClientID}}
	resp, err := httpPostForm(workosDeviceAuthURL, form)
	if err != nil {
		return nil, fmt.Errorf("workos device auth: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body := readBody(resp)
		return nil, fmt.Errorf("workos device auth failed: %d %s", resp.StatusCode, truncate(body, 200))
	}

	var d deviceAuthResp
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, fmt.Errorf("workos device auth decode: %w", err)
	}
	return &d, nil
}

func pollWorkosToken(deviceCode string, interval, expiresIn int) (*authenticateResp, error) {
	deadline := time.Now().Add(time.Duration(expiresIn) * time.Second)
	currentInterval := interval
	if currentInterval < 5 {
		currentInterval = 5
	}

	for time.Now().Before(deadline) {
		form := url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code": {deviceCode},
			"client_id":   {workosClientID},
		}
		resp, err := httpPostForm(workosAuthenticateURL, form)
		if err != nil {
			return nil, fmt.Errorf("workos poll: %w", err)
		}

		var a authenticateResp
		if err := json.NewDecoder(resp.Body).Decode(&a); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("workos poll decode: %w", err)
		}
		resp.Body.Close()

		if resp.StatusCode == 200 {
			return &a, nil
		}

		switch a.Error {
		case "authorization_pending":
			time.Sleep(time.Duration(currentInterval) * time.Second)
		case "slow_down":
			currentInterval += 5
			time.Sleep(time.Duration(currentInterval) * time.Second)
		default:
			errDesc := a.ErrorDesc
			if errDesc == "" {
				errDesc = a.Error
			}
			return nil, fmt.Errorf("workos polling error: %s", errDesc)
		}
	}
	return nil, fmt.Errorf("device authorization expired (timeout)")
}

func registerWithCline(workosAccess, workosRefresh string) (*clineAuthResp, error) {
	body := map[string]string{
		"accessToken":  workosAccess,
		"refreshToken": workosRefresh,
	}
	resp, err := httpPostJSON(clineAPIBase+"/auth/register", body)
	if err != nil {
		return nil, fmt.Errorf("cline register: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b := readBody(resp)
		return nil, fmt.Errorf("cline register failed: %d %s", resp.StatusCode, truncate(b, 200))
	}

	var c clineAuthResp
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		return nil, fmt.Errorf("cline register decode: %w", err)
	}
	return &c, nil
}

func refreshClineToken(refreshToken string) (*clineRefreshResp, error) {
	body := map[string]string{
		"refreshToken": refreshToken,
		"grantType":    "refresh_token",
	}
	resp, err := httpPostJSON(clineAPIBase+"/auth/refresh", body)
	if err != nil {
		return nil, fmt.Errorf("cline refresh: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("cline refresh failed: %d", resp.StatusCode)
	}

	var c clineRefreshResp
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		return nil, fmt.Errorf("cline refresh decode: %w", err)
	}
	return &c, nil
}

func getToken() (string, error) {
	if cachedToken != "" && time.Now().UnixMilli() < cachedExpiry {
		return cachedToken, nil
	}

	creds := loadCredentials()
	if creds != nil && creds.RefreshToken != "" {
		resp, err := refreshClineToken(creds.RefreshToken)
		if err == nil && resp.Data.AccessToken != "" {
			cachedToken = "workos:" + resp.Data.AccessToken
			cachedRefreshTok = resp.Data.RefreshToken
			if cachedRefreshTok == "" {
				cachedRefreshTok = creds.RefreshToken
			}
			cachedExpiry = parseExpiry(resp.Data.ExpiresAt) - 60000
			saveCredentials(cachedRefreshTok)
			return cachedToken, nil
		}
		log.Printf("Token refresh failed: %v", err)
	}
	return "", fmt.Errorf("no valid credentials. Run with --login flag first")
}

func parseExpiry(exp any) int64 {
	switch v := exp.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case string:
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			return t.UnixMilli()
		}
		t, err = time.Parse(time.RFC3339Nano, v)
		if err == nil {
			return t.UnixMilli()
		}
	}
	return 0
}

func doLogin() error {
	fmt.Println("\nStarting Cline OAuth login...")

	device, err := workosDeviceAuth()
	if err != nil {
		return err
	}

	authURL := device.VerificationURIComplete
	if authURL == "" {
		authURL = device.VerificationURI
	}

	fmt.Println("  1. Open this URL in your browser:")
	fmt.Println("     " + authURL)
	fmt.Println("  2. Enter code: " + device.UserCode)
	fmt.Println("  3. Log in with Google, GitHub, or email")

	// Try to open browser automatically
	_ = openBrowser(authURL)

	fmt.Println("  Waiting for authorization...")

	interval := device.Interval
	if interval < 5 {
		interval = 5
	}
	expiresIn := device.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 300
	}

	workosTok, err := pollWorkosToken(device.DeviceCode, interval, expiresIn)
	if err != nil {
		return err
	}

	fmt.Println("  WorkOS authorized. Registering with Cline...")

	cline, err := registerWithCline(workosTok.AccessToken, workosTok.RefreshToken)
	if err != nil {
		return err
	}

	if cline.Data.RefreshToken == "" {
		return fmt.Errorf("cline registration missing refresh token")
	}

	saveCredentials(cline.Data.RefreshToken)
	cachedToken = "workos:" + cline.Data.AccessToken
	cachedRefreshTok = cline.Data.RefreshToken
	cachedExpiry = parseExpiry(cline.Data.ExpiresAt) - 60000

	email := "unknown"
	if cline.Data.UserInfo != nil && cline.Data.UserInfo.Email != "" {
		email = cline.Data.UserInfo.Email
	}
	fmt.Printf("  Login successful! Account: %s\n", email)
	return nil
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch {
	case isWindows():
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		// Try common browser openers
		for _, candidate := range []string{"xdg-open", "open", "gnome-open"} {
			if _, err := os.Stat("/usr/bin/" + candidate); err == nil {
				cmd = candidate
				break
			}
			if _, err := os.Stat("/usr/local/bin/" + candidate); err == nil {
				cmd = candidate
				break
			}
		}
	}

	if cmd == "" {
		return fmt.Errorf("no browser opener found")
	}

	return runCommand(cmd, args...)
}

func isWindows() bool {
	return strings.Contains(strings.ToLower(os.Getenv("OS")), "windows")
}
