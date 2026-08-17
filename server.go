package gomemcache

import (
	"encoding/json"
	"net/http"
	"time"
)

type CacheServer struct {
	cache *Cache
}

const defaultTTL = 5 * time.Minute

func NewCacheServer(c *Cache) *CacheServer {
	return &CacheServer{
		cache: c,
	}
}

func (cs *CacheServer) SetHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key        string `json:"key"`
		Value      string `json:"value"`
		TTLSeconds int    `json:"TTLSeconds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ttl := defaultTTL
	if req.TTLSeconds > 0 {
		ttl = time.Duration(req.TTLSeconds) * time.Second
	}

	cs.cache.Set(req.Key, req.Value, ttl)
	w.WriteHeader(http.StatusAccepted)
}

func (cs *CacheServer) GetHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	value, found := cs.cache.Get(key)

	if !found {
		http.NotFound(w, r)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"value": value})
}
