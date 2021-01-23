package main

import (
	"log"
	"net/http"
)

func main() {
	const (
		rpcURL = "http://[::1]:7076"
		wsURL  = "ws://[::1]:7078"
	)
	cm, err := newChainManager(rpcURL, wsURL)
	if err != nil {
		log.Fatalln(err)
	}
	http.HandleFunc("/", rpcHandler(cm))
	log.Fatalln(http.ListenAndServe("[::1]:7080", nil))
}
