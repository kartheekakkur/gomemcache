package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	gomemcache "github.com/kartheekakkur/gomemcache"
)

func main() {
	cache := gomemcache.NewCache()
	cache.StartEvictionTicker(1 * time.Minute)
	cs := gomemcache.NewCacheServer(cache)

	http.HandleFunc("/set", cs.SetHandler)
	http.HandleFunc("/get", cs.GetHandler)

	fmt.Println("Starting gomemcache server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
