package gomemcache

import (
	"crypto/sha1"
	"encoding/binary"
	"slices"
	"sort"
	"sync"
)

type Node struct {
	ID   string
	Addr string
}

type ringEntry struct {
	hash uint32
	node Node
}

type HashRing struct {
	entries []ringEntry
	lock    sync.RWMutex
}

func NewHashRing() *HashRing {
	return &HashRing{}
}

func (h *HashRing) hash(key string) uint32 {
	hsh := sha1.New()
	hsh.Write([]byte(key))
	return h.bytesToUint32(hsh.Sum(nil))

}

func (h *HashRing) bytesToUint32(b []byte) uint32 {

	return binary.BigEndian.Uint32(b)
}

func (h *HashRing) AddNode(node Node) {
	h.lock.Lock()
	defer h.lock.Unlock()

	hash := h.hash(node.ID)
	index := sort.Search(len(h.entries), func(i int) bool {
		return h.entries[i].hash >= hash
	})
	h.entries = slices.Insert(h.entries, index, ringEntry{hash: hash, node: node})
}

func (h *HashRing) RemoveNode(nodeID string) {
	h.lock.Lock()
	defer h.lock.Unlock()

	index := -1
	for i, entry := range h.entries {
		if entry.node.ID == nodeID {
			index = i
			break
		}
	}
	if index == -1 {
		return
	}

	h.entries = slices.Delete(h.entries, index, index+1)
}

func (h *HashRing) GetNode(key string) Node {
	h.lock.RLock()
	defer h.lock.RUnlock()

	if len(h.entries) == 0 {
		return Node{}
	}

	hash := h.hash(key)

	index := sort.Search(len(h.entries), func(i int) bool {
		return h.entries[i].hash >= hash
	})

	return h.entries[index].node

}
