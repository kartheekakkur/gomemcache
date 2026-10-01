package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	gomemcache "github.com/kartheekakkur/gomemcache"
)

var port string
var peers string

func main() {
	flag.StringVar(&port, "port", ":8080", "HTTP Server Port")
	flag.StringVar(&peers, "peers", "", "Comma-separated list of peer addresses")
	flag.Parse()

	peerList := strings.Split(peers, ",")
	selfID := "http://localhost" + port
	cs := gomemcache.NewCacheServer(peerList, selfID)

	http.HandleFunc("/set", cs.SetHandler)
	http.HandleFunc("/get", cs.GetHandler)

	fmt.Printf("Starting gomemcache server on %s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}
