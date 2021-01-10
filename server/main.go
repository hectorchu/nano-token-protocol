package main

import (
	"log"
	"net/http"
)

func main() {
	newChainManager("http://[::1]:7076")
	http.HandleFunc("/", rpcHandler)
	if err := http.ListenAndServe("[::1]:7080", nil); err != nil {
		log.Fatalln(err)
	}
}
