package gomemcache

import "sync"


type CacheItem struct{

	Value string
}

type Cache struct{
	mu sync.RWMutex
	items map[string]CacheItem
}

func NewCache() *Cache{
	return &Cache{
		items: map[string]CacheItem{},
	}
}

func (c *Cache) Set(key,value string){
    c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = CacheItem{Value: value}
}

func (c *Cache) Get(key string)(string,bool){
    c.mu.RLock()
	defer c.mu.RUnlock()
	item,found := c.items[key]

	if !found{
		return "",false
	}
	return item.Value,true
}