package session

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/session"
	"github.com/gotd/td/tg"

	cfg "github.com/madtoby2/tgtool/internal/config"
)

type AccountInfo struct {
	Phone       string `json:"phone"`
	SessionFile string `json:"session_file"`
	Username    string `json:"username"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	IsPremium   bool   `json:"is_premium"`
	IsActive    bool   `json:"is_active"`
	LastLogin   string `json:"last_login"`
}

type Manager struct {
	mu       sync.RWMutex
	accounts map[string]*AccountInfo
	clients  map[string]*telegram.Client
}

func NewManager() *Manager {
	return &Manager{
		accounts: make(map[string]*AccountInfo),
		clients:  make(map[string]*telegram.Client),
	}
}

func (m *Manager) List() []AccountInfo {
	m.mu.RLock(); defer m.mu.RUnlock()
	out := make([]AccountInfo, 0, len(m.accounts))
	for _, a := range m.accounts { out = append(out, *a) }
	return out
}

func (m *Manager) Get(phone string) *AccountInfo {
	m.mu.RLock(); defer m.mu.RUnlock()
	a, _ := m.accounts[phone]; return a
}

func (m *Manager) Remove(phone string) bool {
	m.mu.Lock(); defer m.mu.Unlock()
	if _, ok := m.accounts[phone]; !ok { return false }
	delete(m.accounts, phone); delete(m.clients, phone)
	os.Remove(filepath.Join(cfg.SessionsDir, "account_"+phone))
	return true
}

func (m *Manager) GetClient(ctx context.Context, phone string) (*telegram.Client, error) {
	m.mu.RLock()
	c, ok := m.clients[phone]
	m.mu.RUnlock()
	if ok { return c, nil }

	conf := cfg.Load()
	client := telegram.NewClient(conf.APIID, conf.APIHash, telegram.Options{
		SessionStorage: &session.FileStorage{Path: filepath.Join(cfg.SessionsDir, "account_"+phone)},
	})
	err := client.Run(ctx, func(ctx context.Context) error {
		status, err := client.Auth().Status(ctx)
		if err != nil || !status.Authorized { return err }
		self, _ := client.Self(ctx)
		info := &AccountInfo{Phone: phone, FirstName: self.FirstName,
			LastName: self.LastName, Username: self.Username,
			IsPremium: self.Premium, IsActive: true}
		m.mu.Lock()
		m.accounts[phone] = info
		m.clients[phone] = client
		m.mu.Unlock()
		return nil
	})
	return client, err
}

type codeAuth struct {
	phone  string
	codeFn func() string
	passFn func() string
}

func (c codeAuth) Phone(_ context.Context) (string, error)               { return c.phone, nil }
func (c codeAuth) Password(_ context.Context) (string, error)             { if c.passFn != nil { return c.passFn(), nil }; return "", nil }
func (c codeAuth) Code(_ context.Context, _ *tg.AuthSentCode) (string, error) { return c.codeFn(), nil }
func (c codeAuth) AcceptTermsOfService(_ context.Context, _ tg.HelpTermsOfService) error { return nil }
func (c codeAuth) SignUp(_ context.Context) (auth.UserInfo, error)       { return auth.UserInfo{}, nil }

func Login(ctx context.Context, phone string, codeFn func() string, passFn func() string) (*telegram.Client, error) {
	conf := cfg.Load()
	client := telegram.NewClient(conf.APIID, conf.APIHash, telegram.Options{
		SessionStorage: &session.FileStorage{Path: filepath.Join(cfg.SessionsDir, "account_"+phone)},
	})
	flow := auth.NewFlow(codeAuth{phone: phone, codeFn: codeFn, passFn: passFn}, auth.SendCodeOptions{})
	return client, client.Run(ctx, func(ctx context.Context) error {
		return client.Auth().IfNecessary(ctx, flow)
	})
}
