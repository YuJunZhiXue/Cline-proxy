package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type clinePassAccountView struct {
	Index      int    `json:"index"`
	Name       string `json:"name"`
	APIKey     string `json:"apiKey"`
	Enabled    bool   `json:"enabled"`
	Current    bool   `json:"current"`
	UsageCount int64  `json:"usageCount"`
	LastUsed   any    `json:"lastUsed,omitempty"`
}

func maskClinePassKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return key[:2] + "****"
	}
	end := 7
	if end > len(key)-3 {
		end = len(key) - 3
	}
	return key[:end] + "****" + key[len(key)-3:]
}

func clinePassAccountViews(cfg *ClinePassConfig) []clinePassAccountView {
	out := make([]clinePassAccountView, 0, len(cfg.Accounts))
	for i, account := range cfg.Accounts {
		out = append(out, clinePassAccountView{
			Index:      i,
			Name:       account.Name,
			APIKey:     maskClinePassKey(account.APIKey),
			Enabled:    account.Enabled,
			Current:    i == cfg.CurrentIdx,
			UsageCount: account.UsageCount,
			LastUsed:   account.LastUsed,
		})
	}
	return out
}

func handleClinePassConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	cfg := getClinePassConfig()
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Data: map[string]any{
		"enabled":                     cfg.Enabled,
		"baseURL":                     cfg.BaseURL,
		"accountMode":                 cfg.AccountMode,
		"currentIdx":                  cfg.CurrentIdx,
		"allowClientProviderOverride": cfg.AllowClientProviderOverride,
		"accountCount":                len(cfg.Accounts),
		"perModel":                    cfg.PerModel,
	}})
}

func handleClinePassConfigUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	var patch struct {
		Enabled                     *bool   `json:"enabled"`
		BaseURL                     *string `json:"baseURL"`
		AccountMode                 *string `json:"accountMode"`
		CurrentIdx                  *int    `json:"currentIdx"`
		AllowClientProviderOverride *bool   `json:"allowClientProviderOverride"`
	}
	if err := decodeAdminJSON(r, &patch); err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	cfg := getClinePassConfig()
	if patch.Enabled != nil {
		cfg.Enabled = *patch.Enabled
	}
	if patch.BaseURL != nil && strings.TrimSpace(*patch.BaseURL) != "" {
		cfg.BaseURL = strings.TrimRight(strings.TrimSpace(*patch.BaseURL), "/")
	}
	if patch.AccountMode != nil {
		if *patch.AccountMode != "single" && *patch.AccountMode != "round_robin" {
			writeAPI(w, http.StatusBadRequest, apiResponse{Error: "accountMode must be single or round_robin"})
			return
		}
		cfg.AccountMode = *patch.AccountMode
	}
	if patch.CurrentIdx != nil {
		if *patch.CurrentIdx < 0 || *patch.CurrentIdx >= len(cfg.Accounts) {
			writeAPI(w, http.StatusBadRequest, apiResponse{Error: "currentIdx is out of range"})
			return
		}
		cfg.CurrentIdx = *patch.CurrentIdx
	}
	if patch.AllowClientProviderOverride != nil {
		cfg.AllowClientProviderOverride = *patch.AllowClientProviderOverride
	}
	setClinePassConfig(cfg)
	handleClinePassConfig(w, &http.Request{Method: http.MethodGet})
}

func handleClinePassAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	cfg := getClinePassConfig()
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Data: clinePassAccountViews(cfg)})
}

func handleClinePassAccountAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	var req struct {
		Name    string `json:"name"`
		APIKey  string `json:"apiKey"`
		Enabled *bool  `json:"enabled"`
	}
	if err := decodeAdminJSON(r, &req); err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.APIKey = strings.TrimSpace(req.APIKey)
	if req.Name == "" || req.APIKey == "" {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "name and apiKey are required"})
		return
	}
	cfg := getClinePassConfig()
	for _, account := range cfg.Accounts {
		if account.Name == req.Name {
			writeAPI(w, http.StatusConflict, apiResponse{Error: "account name already exists"})
			return
		}
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	cfg.Accounts = append(cfg.Accounts, ClinePassAccount{Name: req.Name, APIKey: req.APIKey, Enabled: enabled})
	setClinePassConfig(cfg)
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Data: map[string]any{"name": req.Name, "apiKey": maskClinePassKey(req.APIKey), "enabled": enabled}})
}

func handleClinePassAccountUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	var req struct {
		Index   *int   `json:"index"`
		Name    string `json:"name"`
		NewName string `json:"newName"`
		APIKey  string `json:"apiKey"`
		Enabled *bool  `json:"enabled"`
	}
	if err := decodeAdminJSON(r, &req); err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	cfg := getClinePassConfig()
	idx := findClinePassAccount(cfg, req.Index, req.Name)
	if idx < 0 {
		writeAPI(w, http.StatusNotFound, apiResponse{Error: "account not found"})
		return
	}
	if strings.TrimSpace(req.NewName) != "" {
		cfg.Accounts[idx].Name = strings.TrimSpace(req.NewName)
	}
	if strings.TrimSpace(req.APIKey) != "" {
		cfg.Accounts[idx].APIKey = strings.TrimSpace(req.APIKey)
	}
	if req.Enabled != nil {
		cfg.Accounts[idx].Enabled = *req.Enabled
	}
	setClinePassConfig(cfg)
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Data: clinePassAccountViews(cfg)[idx]})
}

func handleClinePassAccountDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	var req struct {
		Index *int   `json:"index"`
		Name  string `json:"name"`
	}
	if err := decodeAdminJSON(r, &req); err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	cfg := getClinePassConfig()
	idx := findClinePassAccount(cfg, req.Index, req.Name)
	if idx < 0 {
		writeAPI(w, http.StatusNotFound, apiResponse{Error: "account not found"})
		return
	}
	cfg.Accounts = append(cfg.Accounts[:idx], cfg.Accounts[idx+1:]...)
	if idx < cfg.CurrentIdx {
		cfg.CurrentIdx--
	}
	if cfg.CurrentIdx >= len(cfg.Accounts) {
		cfg.CurrentIdx = 0
	}
	setClinePassConfig(cfg)
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Message: "account deleted"})
}

func handleClinePassAccountTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	var req struct {
		Index *int   `json:"index"`
		Name  string `json:"name"`
		Model string `json:"model"`
	}
	if err := decodeAdminJSON(r, &req); err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	cfg := getClinePassConfig()
	idx := findClinePassAccount(cfg, req.Index, req.Name)
	if idx < 0 {
		writeAPI(w, http.StatusNotFound, apiResponse{Error: "account not found"})
		return
	}
	account := cfg.Accounts[idx]
	model := strings.TrimSpace(req.Model)
	if model == "" {
		for configuredModel := range cfg.PerModel {
			if !strings.HasSuffix(configuredModel, "*") {
				model = configuredModel
				break
			}
		}
	}
	if model == "" {
		model = clinePassDefaultModel
	}
	resp, raw, err := doClinePassHTTP(account, cfg, buildClinePassBody(map[string]any{
		"model":      model,
		"messages":   []any{map[string]any{"role": "user", "content": "hi"}},
		"max_tokens": float64(1),
	}, false))
	if err != nil {
		writeAPI(w, http.StatusBadGateway, apiResponse{Error: "network_error: " + err.Error()})
		return
	}
	status := classifyClinePassProviderValidation(resp.StatusCode, raw)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 && status == "ok" {
		writeAPI(w, http.StatusOK, apiResponse{Success: true, Data: map[string]any{"status": "ok", "httpStatus": resp.StatusCode, "model": model}})
		return
	}
	writeAPI(w, resp.StatusCode, apiResponse{Error: fmt.Sprintf("%s: HTTP %d", status, resp.StatusCode), Data: map[string]any{"status": status, "httpStatus": resp.StatusCode}})
}

func handleClinePassModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Data: map[string]any{"models": clinePassModels()}})
}

func handleClinePassModelProbe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	var req struct {
		Model    string `json:"model"`
		Pipeline string `json:"pipeline"`
	}
	if err := decodeAdminJSON(r, &req); err != nil || strings.TrimSpace(req.Model) == "" {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "model is required"})
		return
	}
	providers, pipeline, err := probeClinePassProviders(strings.TrimSpace(req.Model), req.Pipeline)
	if err != nil && len(providers) == 0 {
		writeAPI(w, http.StatusBadGateway, apiResponse{Error: err.Error(), Data: map[string]any{"providers": []string{}, "pipeline": pipeline}})
		return
	}
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Data: map[string]any{"model": req.Model, "pipeline": pipeline, "providers": providers}})
}

func handleClinePassModelValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	var req struct {
		Model     string   `json:"model"`
		Providers []string `json:"providers"`
	}
	if err := decodeAdminJSON(r, &req); err != nil || strings.TrimSpace(req.Model) == "" {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "model is required"})
		return
	}
	statuses, err := validateClinePassProviders(strings.TrimSpace(req.Model), req.Providers)
	if err != nil {
		writeAPI(w, http.StatusBadGateway, apiResponse{Error: err.Error()})
		return
	}
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Data: map[string]any{"model": req.Model, "statuses": statuses}})
}

func handleClinePassModelPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	var req struct {
		Model  string               `json:"model"`
		Policy ClinePassModelPolicy `json:"policy"`
	}
	if err := decodeAdminJSON(r, &req); err != nil || strings.TrimSpace(req.Model) == "" {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "model and policy are required"})
		return
	}
	policy := normalizeClinePassPolicy(req.Policy)
	cfg := getClinePassConfig()
	cfg.PerModel[strings.TrimSpace(req.Model)] = policy
	setClinePassConfig(cfg)
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Data: map[string]any{"model": req.Model, "policy": policy}})
}

func findClinePassAccount(cfg *ClinePassConfig, index *int, name string) int {
	if index != nil {
		if *index >= 0 && *index < len(cfg.Accounts) {
			return *index
		}
		return -1
	}
	for i, account := range cfg.Accounts {
		if account.Name == name {
			return i
		}
	}
	return -1
}

func decodeAdminJSON(r *http.Request, target any) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}
