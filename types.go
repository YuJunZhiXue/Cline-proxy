package main

import "time"

type Account struct {
	AccountID     string    `json:"accountId"`
	Email         string    `json:"email"`
	RefreshToken  string    `json:"refreshToken"`
	AccessToken  string    `json:"-"`
	ExpiresAt     int64     `json:"-"`
	Status        string    `json:"status"` // active, cooldown, expired
	LastUsed      time.Time `json:"lastUsed"`
	UsageCount    int64     `json:"usageCount"`             // 本地累计成功调用次数
	UsageCountToday int64   `json:"usageCountToday"`       // 本地今日成功调用次数
	UsageDate     string    `json:"usageDate"`             // 本地计数日期 YYYY-MM-DD，跨日自动重置
	CreatedAt     time.Time `json:"createdAt"`
	CooldownUntil time.Time `json:"cooldownUntil,omitempty"` // 预计冷却结束时间
	LastReason    string    `json:"lastReason,omitempty"`    // 最后一次进入冷却/失效的原因
}

type AccountPool struct {
	Accounts      []*Account `json:"accounts"`
	CurrentIdx    int        `json:"currentIdx"`
	Keys          []string   `json:"keys,omitempty"`
	DefaultModel  string     `json:"defaultModel,omitempty"` // 用户自定义默认模型，持久化
}

type LoginMethod int

const (
	MethodDeviceOAuth LoginMethod = iota
	MethodRefreshToken
	MethodSSOCookie
)
