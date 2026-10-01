package gomemcache

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type CacheServer struct {
	cache    *Cache
	peers    []string
	HashRing *HashRing
	selfID   string
	mu       sync.Mutex
}

const defaultTTL = 5 * time.Minute
const replicationHeader = "X-Replication-Request"

func NewCacheServer(peers []string, selfID string) *CacheServer {
	cs := &CacheServer{
		cache:    NewCache(10),
		peers:    peers,
		HashRing: NewHashRing(),
		selfID:   selfID,
	}

	for _, peer := range peers {
		if peer == "" {
			continue
		}
		cs.HashRing.AddNode(Node{ID: peer, Addr: peer})
	}

	cs.HashRing.AddNode(Node{ID: selfID, Addr: "self"})

	return cs
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
	targetNode := cs.HashRing.GetNode(req.Key)
	if targetNode.Addr == "self" {
		cs.cache.Set(req.Key, req.Value, ttl)
		if r.Header.Get(replicationHeader) == "" {
			go cs.replicaSet(req.Key, req.Value, ttl)
		}
		w.WriteHeader(http.StatusOK)
	} else {
		data, err := json.Marshal(req)
		if err != nil {
			http.Error(w, "Failed to encode forwarded request", http.StatusInternalServerError)
			return
		}
		forwarded := r.Clone(r.Context())
		forwarded.Header = r.Header.Clone()
		forwarded.Body = io.NopCloser(bytes.NewReader(data))
		forwarded.ContentLength = int64(len(data))
		cs.forwardRequest(w, targetNode, forwarded)
	}
}

func (cs *CacheServer) GetHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")

	targetNode := cs.HashRing.GetNode(key)

	if targetNode.Addr == "self" {
		value, found := cs.cache.Get(key)

		if !found {
			http.NotFound(w, r)
			return
		}
		if err := json.NewEncoder(w).Encode(map[string]string{"value": value}); err != nil {
			log.Printf("Failed to encode cache response: %v", err)
		}

	} else {
		if r.Header.Get("X-Forwarded-For") == cs.selfID {
			http.Error(w, "Loop detected", http.StatusBadRequest)
			return
		}

		forwarded := r.Clone(r.Context())
		forwarded.Header = r.Header.Clone()
		forwarded.Header.Set("X-Forwarded-For", cs.selfID)
		cs.forwardRequest(w, targetNode, forwarded)
	}
}

func (cs *CacheServer) replicaSet(key, value string, ttl time.Duration) {

	cs.mu.Lock()

	defer cs.mu.Unlock()

	req := struct {
		Key        string `json:"key"`
		Value      string `json:"value"`
		TTLSeconds int    `json:"TTLSeconds"`
	}{
		Key:        key,
		Value:      value,
		TTLSeconds: int(ttl / time.Second),
	}

	data, err := json.Marshal(req)
	if err != nil {
		log.Printf("Failed to encode replication request: %v", err)
		return
	}

	for _, peer := range cs.peers {
		if peer != cs.selfID {
			go func(peer string) {
				client := &http.Client{}

				req, err := http.NewRequest("POST", peer+"/set", bytes.NewReader(data))
				if err != nil {
					log.Printf("Failed to create replication request: %v", err)
					return
				}

				req.Header.Set("Content-Type", "application/json")
				req.Header.Set(replicationHeader, "true")

				resp, err := client.Do(req)

				if err != nil {
					log.Printf("Failed to replicate to peer %s: %v", peer, err)
					return
				}
				defer resp.Body.Close()
				if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
					log.Printf("Replication to peer %s returned status %s", peer, resp.Status)
					return
				}
				log.Println("replication successful to", peer)

			}(peer)
		}
	}
}

func (cs *CacheServer) forwardRequest(w http.ResponseWriter, targetNode Node, r *http.Request) {
	base, err := url.Parse(targetNode.Addr)
	if err != nil || base.Scheme == "" || base.Host == "" {
		http.Error(w, "Invalid peer address", http.StatusBadGateway)
		return
	}

	base.Path = strings.TrimRight(base.Path, "/") + r.URL.Path
	base.RawQuery = r.URL.RawQuery
	forwarded, err := http.NewRequestWithContext(r.Context(), r.Method, base.String(), r.Body)
	if err != nil {
		http.Error(w, "Failed to create forwarded request", http.StatusBadGateway)
		return
	}
	forwarded.Header = r.Header.Clone()

	resp, err := http.DefaultClient.Do(forwarded)
	if err != nil {
		log.Printf("Failed to forward request to %s: %v", targetNode.Addr, err)
		http.Error(w, "Peer unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("Failed to copy response from %s: %v", targetNode.Addr, err)
	}
}
