package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"cache"
)

var db = map[string]string{
	"tlf": "630",
	"pxy": "589",
	"xm":  "567",
}

func main() {
	var port int
	var api bool
	flag.IntVar(&port, "port", 9001, "Cache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		9001: "http://localhost:9001",
		9002: "http://localhost:9002",
		9003: "http://localhost:9003",
	}

	var addrList []string
	for _, v := range addrMap {
		addrList = append(addrList, v)
	}

	distrCache := createGroup()
	if api {
		go startApiServer(apiAddr, distrCache)
	}
	startCacheServer(addrMap[port], addrList, distrCache)
}

func createGroup() *cache.Group {
	return cache.NewGroup("test", 2<<10, cache.GetterFunc(func(key string) ([]byte, error) {
		log.Println("from db load ", key)
		if v, ok := db[key]; ok {
			return []byte(v), nil
		}
		return nil, fmt.Errorf("not found this data %s", key)
	}))
}

func startCacheServer(addr string, addrList []string, cacheGroup *cache.Group) {
	peers := cache.NewHttpPool(addr)
	peers.Set(addrList...)
	cacheGroup.RegisterPeers(peers)
	log.Printf("cache server running %v", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

func startApiServer(addr string, cacheGroup *cache.Group) {
	http.HandleFunc("/api", func(writer http.ResponseWriter, request *http.Request) {
		key := request.URL.Query().Get("key")
		view, err := cacheGroup.Get(key)
		if err != nil {
			http.Error(writer, "bad request", http.StatusInternalServerError)
		}

		writer.Header().Set("Content-Type", "application/octet-stream")
		writer.Write(view.ByteSlice())
	})

	log.Println("get cache server running")
	log.Fatal(http.ListenAndServe(addr[7:], nil))
}
