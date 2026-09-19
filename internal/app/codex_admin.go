package app

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

// ============================================================================
// Codex 后台管理 REST API
//   GET  /admin/api/codex/config          - 获取配置
//   POST /admin/api/codex/config/update   - 更新配置
//   GET  /admin/api/codex/accounts        - 账号列表
//   POST /admin/api/codex/accounts/add    - 添加账号 (auth.json 或手动)
//   POST /admin/api/codex/accounts/update - 编辑 custom 账号
//   POST /admin/api/codex/accounts/delete - 删除账号
//   POST /admin/api/codex/accounts/refresh - 刷新指定账号 token
//   POST /admin/api/codex/models/fetch    - 从供应商获取可用模型列表
// ============================================================================

// GET /admin/api/codex/config
func handleCodexConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	cfg := getCodexConfig()
	data := map[string]any{
		"enabled":       cfg.Enabled,
		"originator":    cfg.Originator,
		"clientVersion": cfg.ClientVersion,
		"osType":        cfg.OSType,
		"osVersion":     cfg.OSVersion,
		"arch":          cfg.Arch,
	}
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Data: data})
}

// POST /admin/api/codex/config/update
func handleCodexConfigUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	var params codexConfigData
	if err := json.Unmarshal(body, &params); err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	cfg := getCodexConfig()
	cfg.Enabled = params.Enabled
	if params.Originator != "" {
		cfg.Originator = params.Originator
	}
	if params.ClientVersion != "" {
		cfg.ClientVersion = params.ClientVersion
	}
	if params.OSType != "" {
		cfg.OSType = params.OSType
	}
	if params.OSVersion != "" {
		cfg.OSVersion = params.OSVersion
	}
	if params.Arch != "" {
		cfg.Arch = params.Arch
	}
	setCodexConfig(cfg)
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Message: "codex config updated"})
}

// GET /admin/api/codex/accounts
func handleCodexAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	p := loadCodexPool()
	codexPoolMu.Lock()
	accounts := make([]map[string]any, 0, len(p.Accounts))
	for i, acc := range p.Accounts {
		accType := acc.Type
		if accType == "" {
			accType = "oauth"
		}
		row := map[string]any{
			"index":       i,
			"type":        accType,
			"accountId":   acc.AccountID,
			"email":       acc.Email,
			"status":      acc.Status,
			"lastUsed":    acc.LastUsed,
			"usageCount":  acc.UsageCount,
			"tokensTotal": acc.TokensTotal,
			"tokensToday": acc.TokensToday,
			"createdAt":   acc.CreatedAt,
		}
		if accType == "custom" {
			row["customUrl"] = acc.CustomURL
			row["models"] = acc.Models
			row["modelMapping"] = acc.ModelMapping
		}
		accounts = append(accounts, row)
	}
	codexPoolMu.Unlock()
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Data: accounts})
}

// POST /admin/api/codex/accounts/add
// 支持三种导入方式:
//   1. auth.json 格式: {"tokens": {"refresh_token": "...", "access_token": "...", ...}}
//   2. OAuth 手动: {"type":"oauth", "refreshToken": "...", "accountId": "..."}
//   3. Custom 供应商: {"type":"custom", "customUrl":"...", "customApiKey":"...", "models":[...], "modelMapping":{...}}
func handleCodexAccountAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}

	// 先检测是否为 custom 类型
	var probe struct {
		Type string `json:"type"`
	}
	json.Unmarshal(body, &probe)

	if probe.Type == "custom" {
		var params struct {
			CustomURL    string            `json:"customUrl"`
			CustomAPIKey string            `json:"customApiKey"`
			Models       []string          `json:"models"`
			ModelMapping map[string]string `json:"modelMapping"`
		}
		if err := json.Unmarshal(body, &params); err != nil {
			writeAPI(w, http.StatusBadRequest, apiResponse{Error: "invalid JSON: " + err.Error()})
			return
		}
		if params.CustomURL == "" {
			writeAPI(w, http.StatusBadRequest, apiResponse{Error: "customUrl is required"})
			return
		}
		if params.CustomAPIKey == "" {
			writeAPI(w, http.StatusBadRequest, apiResponse{Error: "customApiKey is required"})
			return
		}
		acc := &CodexAccount{
			Type:         "custom",
			CustomURL:    params.CustomURL,
			CustomAPIKey: params.CustomAPIKey,
			Models:       params.Models,
			ModelMapping: params.ModelMapping,
			Status:       "active",
			CreatedAt:    time.Now(),
		}
		p := loadCodexPool()
		codexPoolMu.Lock()
		p.Accounts = append(p.Accounts, acc)
		codexPoolMu.Unlock()
		saveCodexPool()
		writeAPI(w, http.StatusOK, apiResponse{
			Success: true,
			Message: "custom supplier added",
			Data:    map[string]any{"customUrl": acc.CustomURL, "models": acc.Models},
		})
		return
	}

	// OAuth 账号: 尝试 auth.json 格式
	var acc *CodexAccount
	if authAcc, err := importCodexAuthJSON(body); err == nil {
		acc = authAcc
	} else {
		// 尝试手动格式
		var params struct {
			RefreshToken string `json:"refreshToken"`
			AccountID    string `json:"accountId"`
			Email        string `json:"email"`
		}
		if err := json.Unmarshal(body, &params); err != nil {
			writeAPI(w, http.StatusBadRequest, apiResponse{Error: "invalid JSON: " + err.Error()})
			return
		}
		if params.RefreshToken == "" {
			writeAPI(w, http.StatusBadRequest, apiResponse{Error: "refreshToken is required"})
			return
		}
		acc = &CodexAccount{
			Type:         "oauth",
			RefreshToken: params.RefreshToken,
			AccountID:    params.AccountID,
			Email:        params.Email,
			Status:       "active",
			CreatedAt:    time.Now(),
		}
		// 尝试刷新获取 account_id 和 email
		if err := refreshCodexToken(acc); err != nil {
			acc.Status = "refresh_failed"
			log.Printf("  codex add: token refresh failed (will retry): %v", err)
		}
	}

	// 加入池
	p := loadCodexPool()
	codexPoolMu.Lock()
	p.Accounts = append(p.Accounts, acc)
	codexPoolMu.Unlock()
	saveCodexPool()

	writeAPI(w, http.StatusOK, apiResponse{
		Success: true,
		Message: "codex account added",
		Data: map[string]any{
			"accountId": acc.AccountID,
			"email":     acc.Email,
			"status":    acc.Status,
		},
	})
}

// POST /admin/api/codex/accounts/update
// 编辑 custom 账号的 URL/APIKey/Models/ModelMapping
// body: {"index": 0, "customUrl":"...", "customApiKey":"...", "models":[...], "modelMapping":{...}}
func handleCodexAccountUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	var params struct {
		Index        int               `json:"index"`
		CustomURL    string            `json:"customUrl"`
		CustomAPIKey string            `json:"customApiKey"`
		Models       []string          `json:"models"`
		ModelMapping map[string]string `json:"modelMapping"`
	}
	if err := json.Unmarshal(body, &params); err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	p := loadCodexPool()
	codexPoolMu.Lock()
	if params.Index < 0 || params.Index >= len(p.Accounts) {
		codexPoolMu.Unlock()
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "index out of range"})
		return
	}
	acc := p.Accounts[params.Index]
	if acc.Type != "custom" {
		codexPoolMu.Unlock()
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "only custom accounts can be edited"})
		return
	}
	if params.CustomURL != "" {
		acc.CustomURL = params.CustomURL
	}
	// apiKey 为空表示不修改
	if params.CustomAPIKey != "" {
		acc.CustomAPIKey = params.CustomAPIKey
	}
	acc.Models = params.Models
	acc.ModelMapping = params.ModelMapping
	codexPoolMu.Unlock()
	saveCodexPool()
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Message: "custom account updated"})
}

// POST /admin/api/codex/accounts/delete
// body: {"index": 0}
func handleCodexAccountDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	var params struct {
		Index int `json:"index"`
	}
	if err := json.Unmarshal(body, &params); err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	p := loadCodexPool()
	codexPoolMu.Lock()
	if params.Index < 0 || params.Index >= len(p.Accounts) {
		codexPoolMu.Unlock()
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "index out of range"})
		return
	}
	p.Accounts = append(p.Accounts[:params.Index], p.Accounts[params.Index+1:]...)
	if p.CurrentIdx >= len(p.Accounts) && len(p.Accounts) > 0 {
		p.CurrentIdx = 0
	}
	codexPoolMu.Unlock()
	saveCodexPool()
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Message: "account deleted"})
}

// POST /admin/api/codex/accounts/refresh
// body: {"index": 0}
func handleCodexAccountRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	var params struct {
		Index int `json:"index"`
	}
	if err := json.Unmarshal(body, &params); err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	p := loadCodexPool()
	codexPoolMu.Lock()
	if params.Index < 0 || params.Index >= len(p.Accounts) {
		codexPoolMu.Unlock()
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "index out of range"})
		return
	}
	acc := p.Accounts[params.Index]
	codexPoolMu.Unlock()

	if err := refreshCodexToken(acc); err != nil {
		writeAPI(w, http.StatusInternalServerError, apiResponse{Error: err.Error()})
		return
	}
	writeAPI(w, http.StatusOK, apiResponse{
		Success: true,
		Message: "token refreshed",
		Data: map[string]any{
			"accountId": acc.AccountID,
			"email":     acc.Email,
			"status":    acc.Status,
		},
	})
}

// POST /admin/api/codex/models/fetch
// 从第三方供应商获取可用模型列表 (不保存, 仅返回)
// body: {"url": "https://...", "apiKey": "sk-..."}
func handleCodexModelsFetch(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	var params struct {
		URL    string `json:"url"`
		APIKey string `json:"apiKey"`
		Index  *int   `json:"index"` // 可选: 传入账号 index, 用存储的 URL+Key
	}
	if err := json.Unmarshal(body, &params); err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	// 如果传了 index, 从账号池取存储的 URL+Key
	if params.Index != nil {
		p := loadCodexPool()
		codexPoolMu.Lock()
		idx := *params.Index
		if idx < 0 || idx >= len(p.Accounts) {
			codexPoolMu.Unlock()
			writeAPI(w, http.StatusBadRequest, apiResponse{Error: "index out of range"})
			return
		}
		acc := p.Accounts[idx]
		codexPoolMu.Unlock()
		if acc.Type != "custom" {
			writeAPI(w, http.StatusBadRequest, apiResponse{Error: "only custom accounts"})
			return
		}
		params.URL = acc.CustomURL
		params.APIKey = acc.CustomAPIKey
	}
	if params.URL == "" || params.APIKey == "" {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "url and apiKey are required"})
		return
	}
	models, err := fetchCodexCustomModels(params.URL, params.APIKey)
	if err != nil {
		writeAPI(w, http.StatusBadGateway, apiResponse{Error: err.Error()})
		return
	}
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Data: map[string]any{"models": models}})
}
