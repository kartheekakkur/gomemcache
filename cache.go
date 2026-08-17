package gomemcache


type CacheItem struct{

	Value string
}

type Cache struct{

	items map[string]CacheItem
}

func NewCache() *Cache{
	return &Cache{
		items: map[string]CacheItem{},
	}
}

func (c *Cache) Set(key,value string){

	c.items[key] = CacheItem{Value: value}
}

func (c *Cache) Get(key string)(string,bool){

	item,found := c.items[key]

	if !found{
		return "",false
	}
	return item.Value,true
}