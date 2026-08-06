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
	UsageCount    int64     `json:"usageCount"`
	UsageCountToday int64   `json:"usageCountToday"`       // 今日使用次数
	UsageDate     string    `json:"usageDate"`             // 计数日期 YYYY-MM-DD，跨日自动重置
	CreatedAt     time.Time `json:"createdAt"`
	CooldownUntil time.Time `json:"cooldownUntil,omitempty"` // 预计冷却结束时间
	LastReason    string    `json:"lastReason,omitempty"`    // 最后一次进入冷却/失效的原因
}

type AccountPool struct {
	Accounts   []*Account `json:"accounts"`
	CurrentIdx int        `json:"currentIdx"`
	Keys       []string   `json:"keys,omitempty"`
}

type LoginMethod int

const (
	MethodDeviceOAuth LoginMethod = iota
	MethodRefreshToken
	MethodSSOCookie
)
