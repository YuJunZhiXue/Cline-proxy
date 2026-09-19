package app

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// ============================================================================
// Codex OAuth 刷新 + JWT 解析
// ChatGPT 模式下, access_token 过期后用 refresh_token 刷新
// JWT claim "https://api.openai.com/auth" -> chatgpt_account_id
// JWT claim "https://api.openai.com/profile" -> email
// ============================================================================

// refreshCodexToken 用 refresh_token 刷新 access_token
func refreshCodexToken(acc *CodexAccount) error {
	form := fmt.Sprintf("grant_type=refresh_token&refresh_token=%s&client_id=%s",
		acc.RefreshToken, codexOAuthClientID)

	req, err := http.NewRequest("POST", codexTokenURL, strings.NewReader(form))
	if err != nil {
		return fmt.Errorf("create token refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", codexUserAgent())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("token refresh request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("token refresh %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("parse token response: %w", err)
	}

	acc.AccessToken = tokenResp.AccessToken
	// 有些响应不返回新的 refresh_token, 保留旧的
	if tokenResp.RefreshToken != "" {
		acc.RefreshToken = tokenResp.RefreshToken
	}
	// expires_in 秒, 提前 60 秒过期
	acc.ExpiresAt = time.Now().UnixMilli() + int64(tokenResp.ExpiresIn-60)*1000
	acc.Status = "active"

	// 从 access_token 提取 account_id 和 email
	if tokenResp.AccessToken != "" {
		if claims, err := parseCodexJWT(tokenResp.AccessToken); err == nil {
			if acc.AccountID == "" {
				acc.AccountID = claims.AccountID
			}
			if acc.Email == "" {
				acc.Email = claims.Email
			}
		}
	}

	// 注意: 不在此处 saveCodexPool。账号尚未入池时保存会把磁盘文件覆盖为空池,
	// 由调用方在账号入池后统一保存
	log.Printf("  codex token refreshed: account=%s email=%s", acc.AccountID, acc.Email)
	return nil
}

// codexJWTClaims 从 JWT 中解析出的 Codex 相关字段
type codexJWTClaims struct {
	AccountID string
	Email     string
	PlanType  string
}

// parseCodexJWT 解析 JWT, 提取 chatgpt_account_id 和 email
// JWT 格式: header.payload.signature, 只解析 payload (不验签)
func parseCodexJWT(jwtStr string) (*codexJWTClaims, error) {
	parts := strings.Split(jwtStr, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	// base64url 解码 payload
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// 尝试标准 base64
		payload, err = base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, fmt.Errorf("decode JWT payload: %w", err)
		}
	}

	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("unmarshal JWT claims: %w", err)
	}

	result := &codexJWTClaims{}

	// claim "https://api.openai.com/auth" -> chatgpt_account_id
	if auth, ok := claims["https://api.openai.com/auth"].(map[string]any); ok {
		if id, ok := auth["chatgpt_account_id"].(string); ok {
			result.AccountID = id
		}
	}

	// claim "https://api.openai.com/profile" -> email
	if profile, ok := claims["https://api.openai.com/profile"].(map[string]any); ok {
		if email, ok := profile["email"].(string); ok {
			result.Email = email
		}
	}

	// chatgpt_plan_type (来自 id_token 的 claim)
	if plan, ok := claims["chatgpt_plan_type"].(string); ok {
		result.PlanType = plan
	}

	return result, nil
}

// importCodexAuthJSON 从 ~/.codex/auth.json 格式的 JSON 导入账号
// auth.json 结构: {tokens: {access_token, refresh_token, id_token, account_id}}
func importCodexAuthJSON(raw []byte) (*CodexAccount, error) {
	var authData struct {
		Tokens struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			IDToken      string `json:"id_token"`
			AccountID    string `json:"account_id"`
		} `json:"tokens"`
	}
	if err := json.Unmarshal(raw, &authData); err != nil {
		return nil, fmt.Errorf("parse auth.json: %w", err)
	}

	if authData.Tokens.RefreshToken == "" {
		return nil, fmt.Errorf("auth.json: refresh_token is empty")
	}

	acc := &CodexAccount{
		RefreshToken: authData.Tokens.RefreshToken,
		AccessToken:  authData.Tokens.AccessToken,
		AccountID:    authData.Tokens.AccountID,
		Status:       "active",
		CreatedAt:    time.Now(),
	}

	// 从 access_token 提取 account_id 和 email
	if authData.Tokens.AccessToken != "" {
		if claims, err := parseCodexJWT(authData.Tokens.AccessToken); err == nil {
			if acc.AccountID == "" {
				acc.AccountID = claims.AccountID
			}
			acc.Email = claims.Email
		}
	}

	// 从 id_token 提取 email 和 plan_type
	if authData.Tokens.IDToken != "" {
		if claims, err := parseCodexJWT(authData.Tokens.IDToken); err == nil {
			if acc.Email == "" {
				acc.Email = claims.Email
			}
		}
	}

	// 立即刷新一次, 获取有效 access_token 和过期时间
	if err := refreshCodexToken(acc); err != nil {
		// 刷新失败也保留账号, 后续请求时再重试
		acc.Status = "refresh_failed"
		log.Printf("  codex import: token refresh failed (will retry): %v", err)
	}

	return acc, nil
}
