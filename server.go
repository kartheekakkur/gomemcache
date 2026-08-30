package gomemcache

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

type CacheServer struct {
	cache *Cache
	peers []string
	mu    sync.Mutex
}

const defaultTTL = 5 * time.Minute

func NewCacheServer(c *Cache,peers []string) *CacheServer {
	return &CacheServer{
		cache: c,
		peers: peers,
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

	if r.Header.Get("replicationHeader") == ""{
		go cs.replicaSet(req.Key,req.Value)
	}
	w.WriteHeader(http.StatusOK)
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

func (cs *CacheServer) replicaSet(key, value string) {

	cs.mu.Lock()

	defer cs.mu.Unlock()

	req := struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}{
		Key:   key,
		Value: value,
	}

	data, _ := json.Marshal(req)

	for _, peer := range cs.peers {
		go func(peer string) {
			client := &http.Client{}

			req, err := http.NewRequest("POST", peer+"/set", bytes.NewReader(data))
			if err != nil {
				log.Printf("Failed to create replication request: %v", err)
				return
			}

			req.Header.Set("Content-Type","application/json")
			req.Header.Set("replicationHeader", "true")

			_,err = client.Do(req)

          if err != nil {
             log.Printf("Failed to replicate to peer %s: %v", peer, err)
          }
          log.Println("replication successful to", peer)

		}(peer)
	}
}
