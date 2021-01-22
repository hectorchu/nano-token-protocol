package main

import (
	"log"
	"net/http"
)

var chainMan *chainManager

const (
	rpcURL = "http://[::1]:7076"
	wsURL  = "ws://[::1]:7078"
)

func main() {
	var err error
	if chainMan, err = newChainManager(rpcURL, wsURL); err != nil {
		log.Fatalln(err)
	}
	http.HandleFunc("/", rpcHandler)
	log.Fatalln(http.ListenAndServe("[::1]:7080", nil))
}
