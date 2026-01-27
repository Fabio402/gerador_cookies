package scraper

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var providerCacheClearOnce sync.Once

type providerCacheEntry struct {
	ScriptURL  string    `json:"scriptUrl"`
	Dynamic    string    `json:"dynamic"`
	ExpiresAt  time.Time `json:"expiresAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	Domain     string    `json:"domain"`
	Provider   string    `json:"provider"`
	Mode       string    `json:"mode"`
}

type ProviderCache struct {
	mu      sync.Mutex
	path    string
	entries map[string]providerCacheEntry
}

func defaultProviderCachePath() string {
	if dir, err := os.UserCacheDir(); err == nil && dir != "" {
		p := filepath.Join(dir, "reqs")
		_ = os.MkdirAll(p, 0o700)
		return filepath.Join(p, "provider-cache.json")
	}
	return filepath.Join(os.TempDir(), "reqs-provider-cache.json")
}

func cacheKey(domain, provider, mode string) string {
	return fmt.Sprintf("%s|%s|%s", domain, provider, mode)
}

func LoadProviderCache(path string) (*ProviderCache, error) {
	pc := &ProviderCache{path: path, entries: map[string]providerCacheEntry{}}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return pc, nil
		}
		return pc, err
	}
	if len(b) == 0 {
		return pc, nil
	}
	var raw map[string]providerCacheEntry
	if err := json.Unmarshal(b, &raw); err != nil {
		return pc, nil
	}
	pc.entries = raw
	return pc, nil
}

func LoadProviderCacheDefault() (*ProviderCache, error) {
	if os.Getenv("REQS_PROVIDER_CACHE_ENABLE") != "1" || os.Getenv("REQS_PROVIDER_CACHE_DISABLE") == "1" {
		return &ProviderCache{path: "", entries: map[string]providerCacheEntry{}}, nil
	}
	path := defaultProviderCachePath()
	if os.Getenv("REQS_PROVIDER_CACHE_CLEAR_ON_START") == "1" {
		providerCacheClearOnce.Do(func() {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				log.Printf("failed to clear provider cache: %v", err)
			}
		})
	}
	return LoadProviderCache(path)
}

func (pc *ProviderCache) Save() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.saveLocked()
}

func (pc *ProviderCache) saveLocked() error {
	if pc.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(pc.path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(pc.entries, "", "  ")
	if err != nil {
		return err
	}
	tmp := pc.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, pc.path)
}

func (pc *ProviderCache) Get(domain, provider, mode string) (providerCacheEntry, bool) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	k := cacheKey(domain, provider, mode)
	v, ok := pc.entries[k]
	if !ok {
		return providerCacheEntry{}, false
	}
	if !v.ExpiresAt.IsZero() && time.Now().After(v.ExpiresAt) {
		delete(pc.entries, k)
		_ = pc.saveLocked()
		return providerCacheEntry{}, false
	}
	return v, true
}

func (pc *ProviderCache) Upsert(domain, provider, mode string, scriptURL *string, dynamic *string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	k := cacheKey(domain, provider, mode)
	cur, _ := pc.entries[k]
	cur.Domain = domain
	cur.Provider = provider
	cur.Mode = mode
	if scriptURL != nil && *scriptURL != "" {
		cur.ScriptURL = *scriptURL
	}
	if dynamic != nil && *dynamic != "" {
		cur.Dynamic = *dynamic
	}
	cur.UpdatedAt = time.Now()
	cur.ExpiresAt = time.Now().Add(24 * time.Hour)
	pc.entries[k] = cur
	_ = pc.saveLocked()
}
