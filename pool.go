package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	pool       *AccountPool
	poolMu     sync.Mutex
	poolSaveMu sync.Mutex
	poolPath   string
)

func init() {
	poolPath = resolveDataPath(".cline-accounts.json")
}

// resolveDataPath 数据文件路径解析：优先可执行文件目录，其次当前工作目录。
// go run 运行时编译产物在临时目录，此时应回退到工作目录（项目根）查找数据文件。
func resolveDataPath(filename string) string {
	candidates := []string{}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), filename))
	}
	if pwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(pwd, filename))
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	// 都不存在：默认写到当前工作目录（go run 场景），保证数据落盘位置稳定
	if len(candidates) > 1 {
		return candidates[1]
	}
	return candidates[0]
}

func loadPool() *AccountPool {
	poolMu.Lock()
	defer poolMu.Unlock()

	if pool != nil {
		return pool
	}

	data, err := os.ReadFile(poolPath)
	if err != nil {
		pool = &AccountPool{Accounts: []*Account{}, Keys: []string{}}
		return pool
	}

	var p AccountPool
	if err := json.Unmarshal(data, &p); err != nil {
		pool = &AccountPool{Accounts: []*Account{}, Keys: []string{}}
		return pool
	}

	if p.Accounts == nil {
		p.Accounts = []*Account{}
	}
	if p.Keys == nil {
		p.Keys = []string{}
	}
	pool = &p
	if pool.DefaultModel != "" {
		defaultModel = pool.DefaultModel
	}
	return pool
}

// setDefaultModel 持久化默认模型：更新内存全局并写入账号池文件
func setDefaultModel(modelID string) {
	initModelsCache()
	modelsMu.Lock()
	_, ok := modelsCache[modelID]
	modelsMu.Unlock()
	if !ok {
		return
	}
	defaultModel = modelID
	p := loadPool()
	poolMu.Lock()
	p.DefaultModel = modelID
	poolMu.Unlock()
	savePool()
}

func savePool() {
	poolMu.Lock()
	defer poolMu.Unlock()
	savePoolLocked()
}

// savePoolLocked 持久化账号池；调用方必须已经持有 poolMu。
func savePoolLocked() {
	poolSaveMu.Lock()
	defer poolSaveMu.Unlock()

	data, _ := json.MarshalIndent(pool, "", "  ")
	if err := os.WriteFile(poolPath, data, 0600); err != nil {
		log.Printf("Failed to save accounts: %v", err)
	}
}

func addAccount(acc *Account) {
	p := loadPool()
	poolMu.Lock()
	p.Accounts = append(p.Accounts, acc)
	poolMu.Unlock()
	savePool()
}

func removeAccount(accountID string) bool {
	p := loadPool()
	poolMu.Lock()

	for i, a := range p.Accounts {
		if a.AccountID == accountID {
			p.Accounts = append(p.Accounts[:i], p.Accounts[i+1:]...)
			savePoolLocked()
			poolMu.Unlock()
			return true
		}
	}
	poolMu.Unlock()
	return false
}

func getAccountByID(accountID string) *Account {
	p := loadPool()
	poolMu.Lock()
	defer poolMu.Unlock()

	for _, a := range p.Accounts {
		if a.AccountID == accountID {
			return a
		}
	}
	return nil
}

func refreshAccountToken(acc *Account) error {
	resp, err := refreshClineToken(acc.RefreshToken)
	if err != nil {
		poolMu.Lock()
		acc.Status = "expired"
		savePoolLocked()
		poolMu.Unlock()
		return fmt.Errorf("token refresh failed: %w", err)
	}

	poolMu.Lock()
	acc.AccessToken = "workos:" + resp.Data.AccessToken
	if resp.Data.RefreshToken != "" {
		acc.RefreshToken = resp.Data.RefreshToken
	}
	acc.ExpiresAt = parseExpiry(resp.Data.ExpiresAt) - 60000
	acc.Status = "active"
	savePoolLocked()
	poolMu.Unlock()
	return nil
}

func pickAccount() *Account {
	p := loadPool()
	poolMu.Lock()

	active := make([]*Account, 0)
	for _, a := range p.Accounts {
		// 自动解除已到期的冷却
		if a.Status == "cooldown" && !a.CooldownUntil.IsZero() && time.Now().After(a.CooldownUntil) {
			a.Status = "active"
			a.CooldownUntil = time.Time{}
			a.LastReason = ""
		}
		if a.Status == "active" {
			active = append(active, a)
		}
	}

	if len(active) == 0 {
		poolMu.Unlock()
		return nil
	}

	cfg := getProxyConfig()

	var acc *Account
	switch cfg.Strategy {
	case "fill":
		// Always pick the first available (fill)
		acc = active[0]
	case "random":
		// Random selection
		n := time.Now().UnixNano() % int64(len(active))
		acc = active[n]
	default: // round_robin
		if p.CurrentIdx >= len(active) {
			p.CurrentIdx = 0
		}
		acc = active[p.CurrentIdx]
		p.CurrentIdx = (p.CurrentIdx + 1) % len(active)
	}

	savePoolLocked()
	poolMu.Unlock()
	return acc
}

func ensureAccountToken(acc *Account) (string, error) {
	if acc.AccessToken != "" && time.Now().UnixMilli() < acc.ExpiresAt {
		return acc.AccessToken, nil
	}

	if err := refreshAccountToken(acc); err != nil {
		return "", err
	}

	return acc.AccessToken, nil
}

func listAccounts() []*Account {
	p := loadPool()
	poolMu.Lock()

	// 自动解除已到期的冷却，确保返回的列表是最新状态
	usageDate := time.Now().Format("2006-01-02")
	for _, a := range p.Accounts {
		if a.Status == "cooldown" && !a.CooldownUntil.IsZero() && time.Now().After(a.CooldownUntil) {
			a.Status = "active"
			a.CooldownUntil = time.Time{}
			a.LastReason = ""
		}
		if a.UsageDate != usageDate {
			a.UsageDate = usageDate
			a.UsageCountToday = 0
		}
	}
	result := make([]*Account, len(p.Accounts))
	for i, a := range p.Accounts {
		// Don't expose tokens
		result[i] = &Account{
			AccountID:       a.AccountID,
			Email:           a.Email,
			Status:          a.Status,
			LastUsed:        a.LastUsed,
			UsageCount:      a.UsageCount,
			UsageCountToday: a.UsageCountToday,
			UsageDate:       a.UsageDate,
			CreatedAt:       a.CreatedAt,
			CooldownUntil:   a.CooldownUntil,
			LastReason:     a.LastReason,
		}
	}
	savePoolLocked()
	poolMu.Unlock()
	return result
}

// markAccountCooldown 将账号置为冷却状态，并记录预计恢复时间。
// duration 为冷却时长；duration<=0 时使用默认冷却。
func markAccountCooldown(acc *Account, reason string, duration time.Duration) {
	if acc == nil {
		return
	}
	if duration <= 0 {
		duration = 18 * time.Hour // 默认 18 小时（Cline 免费额度每日重置）
	}
	poolMu.Lock()
	acc.Status = "cooldown"
	acc.CooldownUntil = time.Now().Add(duration)
	acc.LastReason = reason
	savePoolLocked()
	poolMu.Unlock()
}

// bumpUsage 递增本地成功调用计数（含今日计数），自动处理跨日重置。
func bumpUsage(acc *Account) {
	if acc == nil {
		return
	}

	poolMu.Lock()
	now := time.Now()
	today := now.Format("2006-01-02")
	if acc.UsageDate != today {
		acc.UsageDate = today
		acc.UsageCountToday = 0
	}
	acc.UsageCountToday++
	acc.UsageCount++
	acc.LastUsed = now
	savePoolLocked()
	poolMu.Unlock()
}

// resetTodayUsage 仅重置本地今日调用计数，不影响累计调用次数。
func resetTodayUsage(acc *Account) {
	if acc == nil {
		return
	}

	poolMu.Lock()
	acc.UsageDate = time.Now().Format("2006-01-02")
	acc.UsageCountToday = 0
	savePoolLocked()
	poolMu.Unlock()
}

// describePoolStatus 汇总当前账号池状态，用于错误诊断。
func describePoolStatus() string {
	p := loadPool()
	poolMu.Lock()
	defer poolMu.Unlock()

	total := len(p.Accounts)
	if total == 0 {
		return "pool is empty, use --add-account or admin API to add accounts"
	}

	active, cooldown, expired := 0, 0, 0
	var nextRecover *time.Time
	for _, a := range p.Accounts {
		switch a.Status {
		case "active":
			active++
		case "cooldown":
			cooldown++
			if !a.CooldownUntil.IsZero() {
				if nextRecover == nil || a.CooldownUntil.Before(*nextRecover) {
					t := a.CooldownUntil
					nextRecover = &t
				}
			}
		case "expired":
			expired++
		}
	}

	s := fmt.Sprintf("total=%d active=%d cooldown=%d expired=%d", total, active, cooldown, expired)
	if cooldown > 0 && nextRecover != nil {
		s += fmt.Sprintf(", earliest recover at %s", nextRecover.Format("2006-01-02 15:04:05"))
	}
	return s
}

func addAccountFromDeviceAuth() (*Account, error) {
	fmt.Println("\n=== Add New Cline Account (OAuth) ===")

	device, err := workosDeviceAuth()
	if err != nil {
		return nil, err
	}

	authURL := device.VerificationURIComplete
	if authURL == "" {
		authURL = device.VerificationURI
	}

	fmt.Println("  1. Open this URL in your browser:")
	fmt.Println("     " + authURL)
	fmt.Println("  2. Enter code: " + device.UserCode)
	fmt.Println("  3. Log in with Google, GitHub, or email")

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
		return nil, err
	}

	fmt.Println("  WorkOS authorized. Registering with Cline...")

	cline, err := registerWithCline(workosTok.AccessToken, workosTok.RefreshToken)
	if err != nil {
		return nil, err
	}

	if cline.Data.RefreshToken == "" {
		return nil, fmt.Errorf("cline registration missing refresh token")
	}

	email := "unknown"
	if cline.Data.UserInfo != nil && cline.Data.UserInfo.Email != "" {
		email = cline.Data.UserInfo.Email
	}

	acc := &Account{
		AccountID:    fmt.Sprintf("acc_%d", time.Now().UnixMilli()),
		Email:        email,
		RefreshToken: cline.Data.RefreshToken,
		AccessToken:  "workos:" + cline.Data.AccessToken,
		ExpiresAt:    parseExpiry(cline.Data.ExpiresAt) - 60000,
		Status:       "active",
		CreatedAt:    time.Now(),
	}

	addAccount(acc)
	fmt.Printf("  Account added! Email: %s\n", email)
	return acc, nil
}
