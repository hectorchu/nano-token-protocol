package main

import (
	"log"
	"time"
)

const rpcURL = "http://[::1]:7076"

func main() {
	cm := newChainsManager()
	modifiedSince := time.Date(2020, 12, 25, 0, 0, 0, 0, time.UTC)
	if err := cm.scanForChains(modifiedSince, rpcURL); err != nil {
		log.Fatalln(err)
	}
}
