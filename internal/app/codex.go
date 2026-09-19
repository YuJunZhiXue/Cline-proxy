package app

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"cline-go-proxy/internal/kit"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

// ============================================================================
// Codex 上游通道: 伪装为 Codex 桌面版客户端, 转发 Responses API 请求
// 支持两种模式:
//   - chatgpt: ChatGPT OAuth 凭证, 走 chatgpt.com/backend-api/codex
//   - custom:  用户自定义 base_url + api_key, 同样伪装 Codex 客户端请求头
// ============================================================================

const (
	codexDefaultBaseURL    = "https://chatgpt.com/backend-api/codex"
	codexTokenURL          = "https://auth.openai.com/oauth/token"
	codexOAuthClientID     = "app_EMoamEEZ73f0CkXaXp7hrann"
	codexDefaultOriginator = "codex_cli_rs"
	codexDefaultVersion     = "0.154.0"
)

// codexCLISystemPrompt 是 Codex CLI 发送给 Responses API 的标识性系统提示词
// 第三方中转商通过检查此字段判断请求是否来自 Codex 客户端
var codexCLISystemPrompt = strings.Join([]string{
	"You are Codex, based on GPT-5. You are running as a coding agent in the Codex CLI on a user's computer.",
	"",
	"## General",
	"",
	"- The arguments to `shell` will be passed to execvp(). Most terminal commands should be prefixed with [\"bash\", \"-lc\"].",
	"- Always set the `workdir` param when using the shell function. Do not use `cd` unless absolutely necessary.",
	"- When searching for text or files, prefer using `rg` or `rg --files` respectively because `rg` is much faster than alternatives like `grep`. (If the `rg` command is not found, then use alternatives.)",
	"",
	"## Editing constraints",
	"",
	"- Default to ASCII when editing or creating files. Only introduce non-ASCII or other Unicode characters when there is a clear justification and the file already uses them.",
	"- Add succinct code comments that explain what is going on if code is not self-explanatory. You should not add comments like \"Assigns the value to the variable\", but a brief comment might be useful ahead of a complex code block that a user would otherwise have to spend time parsing out. Usage of these comments should be rare.",
	"- Try to use apply_patch for single file edits, but it is fine to explore other options to make the edit if it does not work well. Do not use apply_patch for changes that are auto-generated (i.e. generating package.json or running a lint or format command like gofmt) or when scripting is more efficient (such as search and replacing a string across a codebase).",
	"- You may be in a dirty git worktree.",
	"  - NEVER revert existing changes you did not make unless explicitly requested, since these changes were made by the user.",
	"  - If asked to make a commit or code edits and there are unrelated changes to your work or changes that you didn't make in those files, don't revert those changes.",
	"  - If the changes are in files you've touched recently, you should read carefully and understand how you can work with the changes rather than reverting them.",
	"  - If the changes are in unrelated files, just ignore them and don't revert them.",
	"- While you are working, you might notice unexpected changes that you didn't make. If this happens, STOP IMMEDIATELY and ask the user how they would like to proceed.",
	"  - **NEVER** use destructive commands like `git reset --hard` or `git checkout --` unless specifically requested or approved by the user.",
}, "\n")

// CodexAccount Codex 凭证
// Type 为 "oauth" 表示 ChatGPT OAuth 账号, "custom" 表示第三方自定义供应商
// 旧数据无 Type 字段, 视为 "oauth"
type CodexAccount struct {
	Type         string `json:"type,omitempty"` // "oauth" / "custom", 空值=oauth
	// OAuth 字段
	AccountID    string    `json:"accountId"`
	Email        string    `json:"email"`
	RefreshToken string    `json:"refreshToken"`
	AccessToken  string    `json:"-"`
	ExpiresAt    int64     `json:"-"`
	// Custom 字段
	CustomURL    string            `json:"customUrl,omitempty"`
	CustomAPIKey string            `json:"customApiKey,omitempty"`
	Models       []string          `json:"models,omitempty"`       // 该供应商启用的模型列表
	ModelMapping map[string]string `json:"modelMapping,omitempty"` // 模型名映射, 如 {"gpt-5":"gpt-5-turbo"}
	// 通用字段
	Status       string    `json:"status"`
	LastUsed     time.Time `json:"lastUsed"`
	UsageCount   int64     `json:"usageCount"`
	TokensTotal  int64     `json:"tokensTotal"`
	TokensToday  int64     `json:"tokensToday"`
	TokensDate   string    `json:"tokensDate"`
	CreatedAt    time.Time `json:"createdAt"`
}

// CodexAccountPool Codex 账号池持久化结构
type CodexAccountPool struct {
	Accounts   []*CodexAccount `json:"accounts"`
	CurrentIdx int             `json:"currentIdx"` // OAuth 轮询索引
	CustomIdx  int             `json:"customIdx"`  // Custom 轮询索引
}

// codexConfigData Codex 上游配置 (仅客户端伪装参数, 供应商凭证在账号池中)
type codexConfigData struct {
	Enabled       bool   `json:"enabled"`
	Originator    string `json:"originator"`    // 默认 codex_cli_rs
	ClientVersion string `json:"clientVersion"` // 默认 0.154.0
	OSType        string `json:"osType"`        // UA 中的 OS 类型
	OSVersion     string `json:"osVersion"`     // UA 中的 OS 版本
	Arch          string `json:"arch"`          // UA 中的架构
}

func defaultCodexConfig() *codexConfigData {
	return &codexConfigData{
		Enabled:       false,
		Originator:    codexDefaultOriginator,
		ClientVersion: codexDefaultVersion,
		OSType:        "Windows",
		OSVersion:     "11",
		Arch:          "x86_64",
	}
}

var (
	codexConfig     = loadCodexConfig()
	codexConfigMu   sync.Mutex
	codexPool       *CodexAccountPool
	codexPoolMu     sync.Mutex
	codexPoolPath   string
	codexPoolSaveMu sync.Mutex
	codexHTTPClient *http.Client
	codexTransportMu sync.Mutex
)

func init() {
	codexPoolPath = kit.ResolveDataPath(".codex-accounts.json")
	codexHTTPClient = &http.Client{Transport: buildCodexTransport(), Timeout: 120 * time.Second}
	migrateCodexLegacyConfig()
}

// migrateCodexLegacyConfig 将旧版全局 custom 配置迁移为 custom 账号入池
func migrateCodexLegacyConfig() {
	path := kit.ResolveDataPath(".codex-config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var legacy struct {
		Mode          string `json:"mode"`
		CustomBaseURL string `json:"customBaseURL"`
		APIKey        string `json:"apiKey"`
		BaseURL       string `json:"baseURL"` // 更早的旧字段
	}
	if json.Unmarshal(data, &legacy) != nil {
		return
	}
	// 检测旧 custom 配置
	oldURL := legacy.CustomBaseURL
	if oldURL == "" && legacy.BaseURL != "" && legacy.BaseURL != codexDefaultBaseURL {
		oldURL = legacy.BaseURL
	}
	if legacy.Mode != "custom" || oldURL == "" {
		return
	}
	// 迁移: 创建 custom 账号 (仅当池中尚无 custom 账号时)
	p := loadCodexPool()
	codexPoolMu.Lock()
	hasCustom := false
	for _, acc := range p.Accounts {
		if acc.Type == "custom" {
			hasCustom = true
			break
		}
	}
	if !hasCustom {
		p.Accounts = append(p.Accounts, &CodexAccount{
			Type:         "custom",
			CustomURL:    oldURL,
			CustomAPIKey: legacy.APIKey,
			Models:       []string{},
			Status:       "active",
			CreatedAt:    time.Now(),
		})
		codexPoolMu.Unlock()
		saveCodexPool()
		log.Printf("codex: migrated legacy custom config to account")
	} else {
		codexPoolMu.Unlock()
	}
	// 清除旧字段: 重新保存不含 mode/customBaseURL/apiKey 的配置
	setCodexConfig(getCodexConfig())
}

// ============ 配置加载/保存 ============

func loadCodexConfig() *codexConfigData {
	path := kit.ResolveDataPath(".codex-config.json")
	cfg := defaultCodexConfig()
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			log.Printf("codex config parse failed: %v", err)
		}
	}
	// 迁移旧 originator 值
	if cfg.Originator == "" || cfg.Originator == "codex_chatgpt_desktop" {
		cfg.Originator = codexDefaultOriginator
	}
	if cfg.ClientVersion == "" {
		cfg.ClientVersion = codexDefaultVersion
	}
	return cfg
}

func saveCodexConfig() {
	codexConfigMu.Lock()
	defer codexConfigMu.Unlock()
	data, _ := json.MarshalIndent(codexConfig, "", "  ")
	if err := os.WriteFile(kit.ResolveDataPath(".codex-config.json"), data, 0600); err != nil {
		log.Printf("codex config save failed: %v", err)
	}
}

// getCodexConfig 返回配置的拷贝, 避免调用方持有共享指针引发数据竞态
func getCodexConfig() *codexConfigData {
	codexConfigMu.Lock()
	defer codexConfigMu.Unlock()
	cp := *codexConfig
	return &cp
}

func setCodexConfig(c *codexConfigData) {
	codexConfigMu.Lock()
	cp := *c
	codexConfig = &cp
	codexConfigMu.Unlock()
	saveCodexConfig()
}

// ============ 账号池加载/保存 ============

func loadCodexPool() *CodexAccountPool {
	codexPoolMu.Lock()
	defer codexPoolMu.Unlock()
	if codexPool != nil {
		return codexPool
	}
	data, err := os.ReadFile(codexPoolPath)
	if err != nil {
		codexPool = &CodexAccountPool{Accounts: []*CodexAccount{}}
		return codexPool
	}
	var p CodexAccountPool
	if err := json.Unmarshal(data, &p); err != nil {
		codexPool = &CodexAccountPool{Accounts: []*CodexAccount{}}
		return codexPool
	}
	if p.Accounts == nil {
		p.Accounts = []*CodexAccount{}
	}
	codexPool = &p
	return codexPool
}

func saveCodexPool() {
	// 兜底加载: 进程内池尚未初始化时先从磁盘加载,
	// 避免把已有账号文件覆盖为空池
	loadCodexPool()
	codexPoolMu.Lock()
	defer codexPoolMu.Unlock()
	codexPoolSaveMu.Lock()
	defer codexPoolSaveMu.Unlock()
	data, _ := json.MarshalIndent(codexPool, "", "  ")
	if err := os.WriteFile(codexPoolPath, data, 0600); err != nil {
		log.Printf("codex pool save failed: %v", err)
	}
}

// ============ uTLS HTTP 客户端 ============

func buildCodexTransport() *http.Transport {
	t := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
	}
	// https 走 HTTP/2 + uTLS Chrome 指纹, 与 zen 上游共用同一套伪装思路
	t.RegisterProtocol("https", &http2.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
			d := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
			raw, err := d.DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				raw.Close()
				return nil, err
			}
			uconn := utls.UClient(raw, &utls.Config{
				ServerName: host,
				NextProtos: []string{"h2", "http/1.1"},
			}, utls.HelloChrome_120)
			if err := uconn.HandshakeContext(ctx); err != nil {
				raw.Close()
				return nil, err
			}
			return uconn, nil
		},
	})
	return t
}

// ============ 请求头伪装 ============

// codexUserAgent 构造 Codex CLI User-Agent
// 格式: codex_cli_rs/<version> (<OS> <ver>; <arch>)
func codexUserAgent() string {
	cfg := getCodexConfig()
	return fmt.Sprintf("%s/%s (%s %s; %s)",
		cfg.Originator, cfg.ClientVersion, cfg.OSType, cfg.OSVersion, cfg.Arch)
}

// codexSessionContext 保存一次请求的完整会话上下文
// 中转商通过检查 session/thread/window 一致性判断请求是否来自真正的 Codex 客户端
type codexSessionContext struct {
	SessionID      string
	ThreadID       string
	WindowID       string
	InstallationID string
}

// newCodexSessionContext 生成完整的会话上下文
func newCodexSessionContext() *codexSessionContext {
	return &codexSessionContext{
		SessionID:      newCodexSessionID(),
		ThreadID:       newCodexSessionID(),
		WindowID:       newCodexSessionID(),
		InstallationID: newCodexSessionID(),
	}
}

// codexHeaders 构造 Codex 客户端请求头
// chatgpt 模式: 用 access_token + account_id
// custom 模式:  用用户 api_key, 不发 account_id
func codexHeaders(authToken, accountID string, ctx *codexSessionContext) http.Header {
	cfg := getCodexConfig()
	h := http.Header{}
	h.Set("Authorization", "Bearer "+authToken)
	h.Set("Content-Type", "application/json")
	h.Set("originator", cfg.Originator)
	h.Set("User-Agent", codexUserAgent())
	h.Set("OpenAI-Beta", "responses=v1")

	// 会话一致性头: 中转商检查这些字段在 header / body.client_metadata / x-codex-turn-metadata 三处一致
	h.Set("session-id", ctx.SessionID)
	h.Set("thread-id", ctx.ThreadID)
	h.Set("x-client-request-id", ctx.ThreadID) // 必须等于 thread-id
	h.Set("x-codex-window-id", ctx.WindowID)
	h.Set("x-codex-installation-id", ctx.InstallationID)

	// turn metadata: JSON 格式, 包含 session/thread/window/installation ID
	turnMeta := map[string]any{
		"installation_id": ctx.InstallationID,
		"session_id":      ctx.SessionID,
		"thread_id":       ctx.ThreadID,
		"window_id":       ctx.WindowID,
		"window_number":   1,
	}
	if metaJSON, err := json.Marshal(turnMeta); err == nil {
		h.Set("x-codex-turn-metadata", string(metaJSON))
	}

	if accountID != "" {
		h.Set("chatgpt-account-id", accountID)
	}
	return h
}

// newCodexSessionID 生成 Codex 风格的 session_id (UUID v4 格式)
func newCodexSessionID() string {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		// fallback: 时间戳
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	// 设置 UUID v4 版本和变体位
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ============ 模型路由 ============

// isCodexModel 判断模型是否应路由到 Codex 上游
func isCodexModel(model string) bool {
	model = strings.TrimSpace(model)
	if model == "" {
		return false
	}
	// gpt-5 系列、codex 系列、o3/o4 系列
	lower := strings.ToLower(model)
	if strings.HasPrefix(lower, "gpt-5") || strings.HasPrefix(lower, "codex") {
		return true
	}
	if strings.HasPrefix(lower, "o3") || strings.HasPrefix(lower, "o4") {
		return true
	}
	// 显式 codex/ 前缀
	if strings.HasPrefix(lower, "codex/") {
		return true
	}
	return false
}

// ============ 账号选择 ============

// pickCodexAccount 轮询选择一个可用的 OAuth 账号 (跳过 custom 账号)
func pickCodexAccount() *CodexAccount {
	p := loadCodexPool()
	codexPoolMu.Lock()
	defer codexPoolMu.Unlock()
	if len(p.Accounts) == 0 {
		return nil
	}
	// 轮询, 跳过 custom 账号
	for i := 0; i < len(p.Accounts); i++ {
		idx := (p.CurrentIdx + i) % len(p.Accounts)
		acc := p.Accounts[idx]
		if acc.Type == "custom" {
			continue
		}
		if acc.Status == "active" {
			p.CurrentIdx = (idx + 1) % len(p.Accounts)
			return acc
		}
	}
	return nil
}

// pickCodexCustomAccount 按模型匹配 custom 账号并轮询选择
// Models 为空的 custom 账号匹配所有模型
func pickCodexCustomAccount(model string) *CodexAccount {
	if model == "" {
		return nil
	}
	p := loadCodexPool()
	codexPoolMu.Lock()
	defer codexPoolMu.Unlock()

	var matched []*CodexAccount
	for _, acc := range p.Accounts {
		if acc.Type != "custom" || acc.Status != "active" {
			continue
		}
		// Models 为空表示匹配所有模型
		if len(acc.Models) == 0 {
			matched = append(matched, acc)
			continue
		}
		for _, m := range acc.Models {
			if m == model {
				matched = append(matched, acc)
				break
			}
		}
	}

	if len(matched) == 0 {
		return nil
	}
	// 轮询
	idx := p.CustomIdx % len(matched)
	p.CustomIdx = (idx + 1) % len(matched)
	return matched[idx]
}

// codexHasModel 判断模型是否应路由到 Codex 上游
// 包括标准 codex 模型和 custom 账号启用的模型
func codexHasModel(model string) bool {
	if isCodexModel(model) {
		return true
	}
	// 检查 custom 账号是否启用了该模型
	p := loadCodexPool()
	codexPoolMu.Lock()
	defer codexPoolMu.Unlock()
	for _, acc := range p.Accounts {
		if acc.Type != "custom" || acc.Status != "active" {
			continue
		}
		for _, m := range acc.Models {
			if m == model {
				return true
			}
		}
	}
	return false
}

// ensureCodexToken 确保账号有有效 access_token, 过期则刷新
func ensureCodexToken(acc *CodexAccount) error {
	if acc.AccessToken != "" && time.Now().UnixMilli() < acc.ExpiresAt {
		return nil
	}
	return refreshCodexToken(acc)
}

// fetchCodexCustomModels 从第三方供应商获取可用模型列表
// 调用 GET {baseURL}/models, 带 Codex 客户端伪装头
func fetchCodexCustomModels(customURL, apiKey string) ([]string, error) {
	if customURL == "" || apiKey == "" {
		return nil, fmt.Errorf("customURL and apiKey are required")
	}
	base := strings.TrimRight(customURL, "/")
	if strings.HasSuffix(base, "/responses") {
		base = strings.TrimSuffix(base, "/responses")
	}
	modelsURL := base + "/models"

	req, err := http.NewRequest("GET", modelsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create models request: %w", err)
	}
	req.Header = codexHeaders(apiKey, "", newCodexSessionContext())

	log.Printf("  codex: fetching models from %s", kit.Truncate(modelsURL, 80))

	resp, err := codexHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("models request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("models API %d: %s", resp.StatusCode, kit.Truncate(string(bodyBytes), 500))
	}

	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("parse models response: %w", err)
	}

	models := make([]string, 0, len(payload.Data))
	for _, m := range payload.Data {
		if m.ID != "" {
			models = append(models, m.ID)
		}
	}
	return models, nil
}

// ============ 上游调用 ============

// callCodexAPIWithAccount 调用 Codex 上游 /responses 端点
// selected 为 nil 时自动选择账号; custom 账号用 URL+Key, OAuth 账号用 access_token
func callCodexAPIWithAccount(params map[string]any, stream bool, selected *CodexAccount) (*http.Response, error) {
	cfg := getCodexConfig()
	if !cfg.Enabled {
		return nil, fmt.Errorf("codex upstream is not enabled")
	}

	body := buildCodexRequestBody(params, stream)

	// 生成会话上下文: 用于 header 和 body.client_metadata 的一致性
	sessCtx := newCodexSessionContext()

	// 注入 client_metadata 到请求体: 中转商检查其与 header 的一致性
	body["client_metadata"] = map[string]string{
		"session_id":            sessCtx.SessionID,
		"thread_id":             sessCtx.ThreadID,
		"x-codex-window-id":     sessCtx.WindowID,
		"x-codex-installation-id": sessCtx.InstallationID,
	}

	// Custom 账号: 注入 Codex CLI 系统提示词 + 应用模型映射
	if selected != nil && selected.Type == "custom" {
		// 注入 Codex CLI 系统提示词: 中转商通过 instructions 字段验证客户端身份
		if existing, ok := body["instructions"].(string); ok && existing != "" {
			body["instructions"] = codexCLISystemPrompt + "\n\n" + existing
		} else if !ok {
			body["instructions"] = codexCLISystemPrompt
		} else {
			body["instructions"] = codexCLISystemPrompt + "\n\n" + existing
		}
		// 应用模型映射
		if model, ok := body["model"].(string); ok {
			if mapped, ok := selected.ModelMapping[model]; ok && mapped != "" {
				body["model"] = mapped
			}
		}
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal codex body: %w", err)
	}

	var authToken, accountID, endpoint string
	accType := ""
	if selected != nil {
		accType = selected.Type
	}

	if accType == "custom" {
		// 第三方自定义供应商: 用 api_key, 不发 account_id
		if selected.CustomAPIKey == "" {
			return nil, fmt.Errorf("codex custom account: apiKey is empty")
		}
		if selected.CustomURL == "" {
			return nil, fmt.Errorf("codex custom account: customURL is empty")
		}
		authToken = selected.CustomAPIKey
		accountID = ""
		base := strings.TrimRight(selected.CustomURL, "/")
		if strings.HasSuffix(base, "/responses") {
			endpoint = base
		} else {
			endpoint = base + "/responses"
		}
	} else {
		// OAuth 账号: 用 access_token + account_id
		acc := selected
		if acc == nil || acc.Type == "custom" {
			acc = pickCodexAccount()
		}
		if acc == nil {
			return nil, fmt.Errorf("no active codex oauth accounts available")
		}
		if err := ensureCodexToken(acc); err != nil {
			return nil, fmt.Errorf("codex token refresh failed: %w", err)
		}
		authToken = acc.AccessToken
		accountID = acc.AccountID
		endpoint = codexDefaultBaseURL + "/responses"
		selected = acc
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("create codex request: %w", err)
	}
	req.Header = codexHeaders(authToken, accountID, sessCtx)
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	}

	model, _ := body["model"].(string)
	log.Printf("  codex upstream: model=%s stream=%v type=%s endpoint=%s",
		model, stream, accType, kit.Truncate(endpoint, 60))

	resp, err := codexHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("codex request: %w", err)
	}

	// OAuth 账号: 401 刷新重试
	if resp.StatusCode == 401 && accType != "custom" {
		resp.Body.Close()
		if err := refreshCodexToken(selected); err == nil {
			req.Header = codexHeaders(selected.AccessToken, selected.AccountID, sessCtx)
			if stream {
				req.Header.Set("Accept", "text/event-stream")
			}
			resp, err = codexHTTPClient.Do(req)
			if err != nil {
				return nil, fmt.Errorf("codex retry: %w", err)
			}
		}
	}

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("codex API %d: %s", resp.StatusCode, kit.Truncate(string(bodyBytes), 500))
	}

	return resp, nil
}

// buildCodexRequestBody 构造 Responses API 请求体
// 如果入站已是 Responses 格式 (有 input 字段), 则透传; 否则从 chat 格式转换
func buildCodexRequestBody(params map[string]any, stream bool) map[string]any {
	body := map[string]any{}

	// 如果入站已经是 Responses 格式 (有 input 字段), 直接透传
	if _, hasInput := params["input"]; hasInput {
		for k, v := range params {
			body[k] = v
		}
		body["stream"] = stream
		// store 默认 false (ChatGPT 后端推荐)
		if _, ok := body["store"]; !ok {
			body["store"] = false
		}
		return body
	}

	// 从 chat/completions 格式转换
	chatToResponsesRequest(params, body)
	body["stream"] = stream
	if _, ok := body["store"]; !ok {
		body["store"] = false
	}
	return body
}

// chatToResponsesRequest 将 chat/completions 请求体转换为 Responses API 请求体
func chatToResponsesRequest(chat map[string]any, out map[string]any) {
	if m, ok := chat["model"].(string); ok {
		out["model"] = m
	}
	// messages → instructions + input
	if msgs, ok := chat["messages"].([]any); ok {
		var instructions string
		var input []any
		for _, msg := range msgs {
			m, ok := msg.(map[string]any)
			if !ok {
				continue
			}
			role, _ := m["role"].(string)
			switch role {
			case "system":
				content := stringifyChatContent(m["content"])
				if instructions != "" {
					instructions += "\n\n"
				}
				instructions += content
			case "user":
				input = append(input, userContentToResponses(m["content"])...)
			case "assistant":
				// assistant 的 tool_calls 先于文本内容, 保持调用链顺序
				if tcs, ok := m["tool_calls"].([]any); ok {
					for _, tc := range tcs {
						tcm, ok := tc.(map[string]any)
						if !ok || tcm["type"] != "function" {
							continue
						}
						fn, _ := tcm["function"].(map[string]any)
						if fn == nil {
							continue
						}
						callID, _ := tcm["id"].(string)
						name, _ := fn["name"].(string)
						args, _ := fn["arguments"].(string)
						if args == "" {
							args = "{}"
						}
						input = append(input, map[string]any{
							"type":      "function_call",
							"call_id":   callID,
							"name":      name,
							"arguments": args,
						})
					}
				}
				content := stringifyChatContent(m["content"])
				if content != "" {
					input = append(input, map[string]any{
						"type":    "message",
						"role":    "assistant",
						"content": []any{map[string]any{"type": "output_text", "text": content}},
					})
				}
			case "tool":
				// tool 结果转为 function_call_output
				toolCallID, _ := m["tool_call_id"].(string)
				content := stringifyChatContent(m["content"])
				input = append(input, map[string]any{
					"type":    "function_call_output",
					"call_id": toolCallID,
					"output":   content,
				})
			}
		}
		if instructions != "" {
			out["instructions"] = instructions
		}
		if input != nil {
			out["input"] = input
		}
	}
	// max_tokens → max_output_tokens
	if mt, ok := chat["max_tokens"].(float64); ok {
		out["max_output_tokens"] = int(mt)
	} else if mt, ok := chat["max_completion_tokens"].(float64); ok {
		out["max_output_tokens"] = int(mt)
	}
	// reasoning_effort → reasoning.effort
	if re, ok := chat["reasoning_effort"].(string); ok && re != "" {
		out["reasoning"] = map[string]any{"effort": re}
	} else if re, ok := chat["reasoningEffort"].(string); ok && re != "" {
		out["reasoning"] = map[string]any{"effort": re}
	}
	// tools 转换 (chat function tools → Responses function tools)
	if tools, ok := chat["tools"].([]any); ok {
		out["tools"] = chatToolsToResponses(tools)
	}
	if tc, ok := chat["tool_choice"]; ok {
		out["tool_choice"] = tc
	}
	// 透传部分参数
	for _, k := range []string{"temperature", "top_p", "stop", "seed", "user"} {
		if v, ok := chat[k]; ok {
			out[k] = v
		}
	}
}

// chatToolsToResponses 将 chat 格式的 tools 转为 Responses 格式
func chatToolsToResponses(tools []any) []any {
	out := make([]any, 0, len(tools))
	for _, t := range tools {
		tm, ok := t.(map[string]any)
		if !ok {
			continue
		}
		if tm["type"] == "function" {
			fn, _ := tm["function"].(map[string]any)
			if fn != nil {
				rt := map[string]any{"type": "function"}
				if n, ok := fn["name"].(string); ok {
					rt["name"] = n
				}
				if d, ok := fn["description"].(string); ok {
					rt["description"] = d
				}
				if p, ok := fn["parameters"]; ok {
					rt["parameters"] = p
				}
				out = append(out, rt)
			}
		}
	}
	return out
}

// stringifyChatContent 将 chat message content 统一为字符串
func stringifyChatContent(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []any:
		parts := []string{}
		for _, block := range v {
			if b, ok := block.(map[string]any); ok {
				if t, ok := b["text"].(string); ok {
					parts = append(parts, t)
				}
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

// userContentToResponses 将 user 消息的 content (纯文本或多模态块数组)
// 转为 Responses input 消息; 图片块映射为 input_image, 文本块映射为 input_text
func userContentToResponses(content any) []any {
	// 纯字符串: 单条 input_text
	if s, ok := content.(string); ok {
		if s == "" {
			return nil
		}
		return []any{map[string]any{
			"type":    "message",
			"role":    "user",
			"content": []any{map[string]any{"type": "input_text", "text": s}},
		}}
	}
	blocks, ok := content.([]any)
	if !ok {
		return nil
	}
	var outContent []any
	for _, block := range blocks {
		b, ok := block.(map[string]any)
		if !ok {
			continue
		}
		switch b["type"] {
		case "text":
			if t, ok := b["text"].(string); ok && t != "" {
				outContent = append(outContent, map[string]any{"type": "input_text", "text": t})
			}
		case "image_url":
			// chat 格式: {"type":"image_url","image_url":{"url":"data:image/png;base64,..."}}
			imgURL, _ := b["image_url"].(map[string]any)
			if imgURL == nil {
				continue
			}
			u, _ := imgURL["url"].(string)
			if u == "" {
				continue
			}
			item := map[string]any{"type": "input_image", "image_url": u}
			if d, ok := b["detail"].(string); ok && d != "" {
				item["detail"] = d
			}
			outContent = append(outContent, item)
		}
	}
	if outContent == nil {
		return nil
	}
	return []any{map[string]any{
		"type":    "message",
		"role":    "user",
		"content": outContent,
	}}
}

// ============ Responses SSE → Chat SSE 转换 ============

// responsesStreamToChat 将上游 Responses SSE 流转换为 chat.completions SSE 流
// 用于 /v1/chat/completions 入站 + codex 上游的场景
// onUsage 在收到 usage 时回调 (可为 nil), 返回值表示流是否正常收尾 (completed/failed)
// objKeys 返回 map 的 key 列表, 用于调试日志
func objKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func responsesStreamToChat(w http.ResponseWriter, upstream *http.Response, onUsage func(map[string]any)) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Printf("  streaming not supported for client")
		return
	}

	chatID := fmt.Sprintf("chatcmpl-%x", time.Now().UnixMilli())
	model := ""
	finished := false
	finishReason := "stop"
	upstreamError := "" // 上游 error 事件的消息
	// 工具调用映射: Responses item_id (call_id) → chat tool_call 序号
	toolIdx := 0
	toolIdxByID := map[string]int{}

	emitChunk := func(delta map[string]any, finish any) {
		chunk := map[string]any{
			"id":      chatID,
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   model,
			"choices": []any{map[string]any{
				"index":         0,
				"delta":         delta,
				"finish_reason": finish,
			}},
		}
		b, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", string(b))
		flusher.Flush()
	}

	reader := bufio.NewReader(upstream.Body)
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			line = strings.TrimRight(line, "\r\n")
			// Responses SSE 格式: "event: xxx\ndata: {...}"
			if strings.HasPrefix(line, "data: ") {
				payload := strings.TrimSpace(line[6:])
				if payload == "" || payload == "[DONE]" {
					continue
				}
				var obj map[string]any
				if json.Unmarshal([]byte(payload), &obj) != nil {
					log.Printf("  codex SSE parse error: %s", kit.Truncate(payload, 200))
					continue
				}
				// 调试: 记录上游 SSE 事件类型
				eventType, _ := obj["type"].(string)
				log.Printf("  codex SSE event: type=%s keys=%v", eventType, objKeys(obj))
				// 提取 model
				if m, ok := obj["model"].(string); ok && m != "" {
					model = m
				}
				// usage
				if onUsage != nil {
					if u, ok := obj["usage"].(map[string]any); ok && len(u) > 0 {
						onUsage(u)
					}
				}
				switch eventType {
			case "error":
				// 上游返回 error 事件, 记录错误信息供 response.failed 使用
				errBytes, _ := json.Marshal(obj["error"])
				log.Printf("  codex upstream error: %s", kit.Truncate(string(errBytes), 500))
				if e, ok := obj["error"].(map[string]any); ok {
					if msg, ok := e["message"].(string); ok && msg != "" {
						upstreamError = msg
					}
				}

			case "response.output_text.delta":
					delta, _ := obj["delta"].(string)
					emitChunk(map[string]any{"content": delta}, nil)

				case "response.reasoning_summary_text.delta":
					delta, _ := obj["delta"].(string)
					emitChunk(map[string]any{"reasoning_content": delta}, nil)

				case "response.output_item.added":
					// function_call 项开始: 记录映射, 发出带 id/name 的首块
					item, _ := obj["item"].(map[string]any)
					if item != nil && item["type"] == "function_call" {
						itemID, _ := obj["item_id"].(string)
						if itemID == "" {
							itemID, _ = item["id"].(string)
						}
						callID, _ := item["call_id"].(string)
						name, _ := item["name"].(string)
						idx := toolIdx
						toolIdx++
						if itemID != "" {
							toolIdxByID[itemID] = idx
						}
						if callID != "" {
							toolIdxByID[callID] = idx
						}
						emitChunk(map[string]any{
							"tool_calls": []any{map[string]any{
								"index": idx,
								"id":    callID,
								"type":  "function",
								"function": map[string]any{
									"name":      name,
									"arguments": "",
								},
							}},
						}, nil)
					}

				case "response.function_call_arguments.delta":
					delta, _ := obj["delta"].(string)
					// 按 item_id/call_id 映射到对应的 chat tool_call 序号
					itemID, _ := obj["item_id"].(string)
					if itemID == "" {
						itemID, _ = obj["call_id"].(string)
					}
					idx, ok := toolIdxByID[itemID]
					if !ok {
						idx = toolIdx - 1
						if idx < 0 {
							idx = 0
						}
					}
					emitChunk(map[string]any{
						"tool_calls": []any{map[string]any{
							"index": idx,
							"function": map[string]any{"arguments": delta},
						}},
					}, nil)

				case "response.failed":
				// 上游明确失败: 透传错误信息后终止
				finished = true
				finishReason = "stop"
				errMsg := upstreamError
				if errMsg == "" {
					errMsg = "codex upstream failed"
					errObj, _ := obj["response"].(map[string]any)
					if errObj != nil {
						if e, ok := errObj["error"].(map[string]any); ok {
							if msg, ok := e["message"].(string); ok && msg != "" {
								errMsg = msg
							}
						}
					}
				}
				emitChunk(map[string]any{"content": "[Error] " + errMsg}, nil)
				emitChunk(nil, finishReason)
				fmt.Fprintf(w, "data: [DONE]\n\n")
				flusher.Flush()

				case "response.completed":
					// 有工具调用时以 tool_calls 收尾, 纯文本以 stop 收尾
					if toolIdx > 0 {
						finishReason = "tool_calls"
					}
					finished = true
					emitChunk(map[string]any{}, finishReason)
					fmt.Fprintf(w, "data: [DONE]\n\n")
					flusher.Flush()
				}
			}
		}
		if err != nil || finished {
			break
		}
	}
}

// ============ Responses 非流式响应收集 → Chat 格式 ============

// collectCodexResponsesStream 收集 Responses SSE 流并转为 chat.completions 非流式响应
// 仅在收到 response.completed 后返回结果, 上游中断/失败返回错误 (不伪造成功)
func collectCodexResponsesStream(upstream *http.Response) (map[string]any, error) {
	var outText strings.Builder
	var model string
	var usage map[string]any
	completed := false
	// 工具调用收集: item_id/call_id → chat tool_call
	toolIdx := 0
	toolIdxByID := map[string]int{}
	toolCalls := map[int]map[string]any{}

	reader := bufio.NewReader(upstream.Body)
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			line = strings.TrimRight(line, "\r\n")
			if strings.HasPrefix(line, "data: ") {
				payload := strings.TrimSpace(line[6:])
				if payload == "" || payload == "[DONE]" {
					continue
				}
				var obj map[string]any
				if json.Unmarshal([]byte(payload), &obj) != nil {
					continue
				}
				if m, ok := obj["model"].(string); ok && m != "" {
					model = m
				}
				if u, ok := obj["usage"].(map[string]any); ok && len(u) > 0 {
					usage = u
				}
				eventType, _ := obj["type"].(string)
				switch eventType {
				case "response.output_text.delta":
					delta, _ := obj["delta"].(string)
					outText.WriteString(delta)

				case "response.output_item.added":
					item, _ := obj["item"].(map[string]any)
					if item != nil && item["type"] == "function_call" {
						itemID, _ := obj["item_id"].(string)
						if itemID == "" {
							itemID, _ = item["id"].(string)
						}
						callID, _ := item["call_id"].(string)
						idx := toolIdx
						toolIdx++
						if itemID != "" {
							toolIdxByID[itemID] = idx
						}
						if callID != "" {
							toolIdxByID[callID] = idx
						}
						toolCalls[idx] = map[string]any{
							"id":    callID,
							"type":  "function",
							"function": map[string]any{
								"name":      item["name"],
								"arguments": "",
							},
						}
					}

				case "response.function_call_arguments.delta":
					delta, _ := obj["delta"].(string)
					itemID, _ := obj["item_id"].(string)
					if itemID == "" {
						itemID, _ = obj["call_id"].(string)
					}
					idx, ok := toolIdxByID[itemID]
					if !ok {
						idx = toolIdx - 1
						if idx < 0 {
							idx = 0
						}
					}
					if tc := toolCalls[idx]; tc != nil {
						fn := tc["function"].(map[string]any)
						prev, _ := fn["arguments"].(string)
						fn["arguments"] = prev + delta
					}

				case "response.failed":
					// 上游明确失败: 提取错误信息返回
					errMsg := "codex upstream failed"
					if respObj, ok := obj["response"].(map[string]any); ok {
						if e, ok := respObj["error"].(map[string]any); ok {
							if msg, ok := e["message"].(string); ok && msg != "" {
								errMsg = msg
							}
						}
					}
					return nil, fmt.Errorf("codex response failed: %s", errMsg)

				case "response.completed":
					completed = true
				}
			}
		}
		if err != nil {
			break
		}
	}

	if !completed {
		// 上游中断: 不能返回伪成功
		return nil, fmt.Errorf("codex stream ended without response.completed")
	}

	message := map[string]any{
		"role": "assistant",
	}
	if _, hasTool := toolCalls[0]; hasTool {
		// 有工具调用: message.tool_calls (按序号排序)
		message["content"] = nil
		arr := make([]any, 0, len(toolCalls))
		for i := 0; i < len(toolCalls); i++ {
			if tc := toolCalls[i]; tc != nil {
				arr = append(arr, tc)
			}
		}
		message["tool_calls"] = arr
	}
	if content := outText.String(); content != "" {
		message["content"] = content
	}
	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	}

	result := map[string]any{
		"id":      fmt.Sprintf("chatcmpl-%x", time.Now().UnixMilli()),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []any{map[string]any{
			"index":         0,
			"message":       message,
			"finish_reason": finishReason,
		}},
	}
	if usage != nil {
		result["usage"] = map[string]any{
			"prompt_tokens":     usage["input_tokens"],
			"completion_tokens": usage["output_tokens"],
			"total_tokens":      usage["total_tokens"],
		}
	}
	return result, nil
}

// toInt64 将 JSON 数值 (float64) 或其他类型安全转换为 int64
func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int:
		return int64(n), true
	case int64:
		return n, true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	}
	return 0, false
}

// ============ Codex 透传 (Responses SSE 直通) ============

// passthroughCodexResponsesSSE 将 Codex 上游的 Responses SSE 直接透传给客户端
// 用于 /v1/responses 入站 + codex 上游的场景
func passthroughCodexResponsesSSE(w http.ResponseWriter, upstream *http.Response) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Printf("  streaming not supported for client")
		return
	}

	reader := bufio.NewReader(upstream.Body)
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			w.Write([]byte(line))
			if strings.HasSuffix(line, "\n") {
				flusher.Flush()
			}
		}
		if err != nil {
			break
		}
	}
}

// ============ Codex 非流式 Responses 收集 ============

// collectCodexResponsesRaw 收集 Responses SSE 流, 返回完整的 response.completed 事件
func collectCodexResponsesRaw(upstream *http.Response) (map[string]any, error) {
	var lastCompleted map[string]any
	reader := bufio.NewReader(upstream.Body)
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			line = strings.TrimRight(line, "\r\n")
			if strings.HasPrefix(line, "data: ") {
				payload := strings.TrimSpace(line[6:])
				if payload == "" || payload == "[DONE]" {
					continue
				}
				var obj map[string]any
				if json.Unmarshal([]byte(payload), &obj) != nil {
					continue
				}
				if eventType, _ := obj["type"].(string); eventType == "response.completed" {
					lastCompleted = obj
				}
			}
		}
		if err != nil {
			break
		}
	}
	if lastCompleted == nil {
		return nil, fmt.Errorf("no response.completed event from codex upstream")
	}
	return lastCompleted, nil
}

// ============ 入站处理入口 ============

// handleCodexChat 处理 /v1/chat/completions 入站, 路由到 Codex 上游
func handleCodexChat(w http.ResponseWriter, params map[string]any, isStream bool) {
	// Codex 后端推荐 stream=true, 非流式入站时上游走流式再聚合
	upstreamStream := isStream
	if !isStream {
		upstreamStream = true
	}

	// 先找 custom 账号 (按模型匹配), 找不到回落 OAuth
	model, _ := params["model"].(string)
	var usedAcc *CodexAccount
	usedAcc = pickCodexCustomAccount(model)
	if usedAcc == nil {
		usedAcc = pickCodexAccount()
	}

	recordUsage := func(u map[string]any) {
		if usedAcc == nil {
			return
		}
		inTok, _ := toInt64(u["input_tokens"])
		outTok, _ := toInt64(u["output_tokens"])
		today := time.Now().Format("2006-01-02")
		codexPoolMu.Lock()
		defer codexPoolMu.Unlock()
		usedAcc.LastUsed = time.Now()
		usedAcc.UsageCount++
		usedAcc.TokensTotal += inTok + outTok
		if usedAcc.TokensDate != today {
			usedAcc.TokensDate = today
			usedAcc.TokensToday = 0
		}
		usedAcc.TokensToday += inTok + outTok
	}

	resp, err := callCodexAPIWithAccount(params, upstreamStream, usedAcc)
	if err != nil {
		log.Printf("  codex api error: %v", err)
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": err.Error(), "type": "api_error"},
		})
		return
	}
	defer resp.Body.Close()

	if isStream {
		// 流式: Responses SSE → Chat SSE
		responsesStreamToChat(w, resp, recordUsage)
		// 请求成功后落盘用量统计
		if usedAcc != nil {
			saveCodexPool()
		}
		return
	}

	// 非流式: 收集 Responses SSE → Chat JSON
	out, err := collectCodexResponsesStream(resp)
	if err != nil {
		log.Printf("  codex api error: %v", err)
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": err.Error(), "type": "api_error"},
		})
		return
	}
	// 非流式从结果对象里补一次用量
	if usage, ok := out["usage"].(map[string]any); ok {
		recordUsage(usage)
	}
	if usedAcc != nil {
		saveCodexPool()
	}
	out = normalizeOpenAIResponse(out)
	writeJSON(w, http.StatusOK, out)
}

// handleCodexResponses 处理 /v1/responses 入站, 路由到 Codex 上游 (透传)
func handleCodexResponses(w http.ResponseWriter, params map[string]any, isStream bool) {
	upstreamStream := isStream
	if !isStream {
		upstreamStream = true
	}

	// 先找 custom 账号 (按模型匹配), 找不到回落 OAuth
	model, _ := params["model"].(string)
	var usedAcc *CodexAccount
	usedAcc = pickCodexCustomAccount(model)
	if usedAcc == nil {
		usedAcc = pickCodexAccount()
	}

	resp, err := callCodexAPIWithAccount(params, upstreamStream, usedAcc)
	if err != nil {
		log.Printf("  codex api error: %v", err)
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{"message": err.Error(), "type": "api_error"},
		})
		return
	}
	defer resp.Body.Close()

	if isStream {
		// 流式: Responses SSE 直接透传
		passthroughCodexResponsesSSE(w, resp)
		return
	}

	// 非流式: 收集 Responses SSE → 返回完整 response 对象
	out, err := collectCodexResponsesRaw(resp)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]string{"message": err.Error(), "type": "parse_error"},
		})
		return
	}
	// 返回 response.completed 事件中的 response 对象
	if respObj, ok := out["response"].(map[string]any); ok {
		writeJSON(w, http.StatusOK, respObj)
	} else {
		writeJSON(w, http.StatusOK, out)
	}
}
