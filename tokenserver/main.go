package main

import (
	"log"
	"net/http"
)

var chainMan *chainManager

func main() {
	var err error
	if chainMan, err = newChainManager("http://[::1]:7076"); err != nil {
		log.Fatalln(err)
	}
	http.HandleFunc("/", rpcHandler)
	if err := http.ListenAndServe("[::1]:7080", nil); err != nil {
		log.Fatalln(err)
	}
}
