package repository

import "sync"

// CacheRepo simula um repositório de cache (Redis)
type CacheRepo struct {
    mu    sync.RWMutex
    store map[string][]byte
}

func NewCacheRepo() *CacheRepo {
    return &CacheRepo{store: map[string][]byte{}}
}

func (r *CacheRepo) Get(key string) ([]byte, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    b, ok := r.store[key]
    return b, ok
}

func (r *CacheRepo) Set(key string, val []byte) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.store[key] = val
}

func (r *CacheRepo) Clear() {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.store = map[string][]byte{}
}
