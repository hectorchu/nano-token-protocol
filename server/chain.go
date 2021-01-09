package main

import (
	"database/sql"
	"log"
	"sync"
	"time"

	"github.com/hectorchu/nano-token-protocol/tokenchain"
	_ "github.com/mattn/go-sqlite3"
)

type chainManager struct {
	c    *tokenchain.Chain
	quit chan bool
}

func newChainManager(c *tokenchain.Chain) (cm *chainManager) {
	cm = &chainManager{
		c:    c,
		quit: make(chan bool),
	}
	go cm.loop()
	return
}

func (cm *chainManager) loop() {
	if err := cm.c.Parse(); err != nil {
		log.Fatalln(err)
		return
	}
	parseTicker := time.NewTicker(10 * time.Second)
	defer parseTicker.Stop()
	dbTicker := time.NewTicker(30 * time.Second)
	defer dbTicker.Stop()
	for {
		select {
		case <-parseTicker.C:
			if err := cm.c.Parse(); err != nil {
				log.Fatalln(err)
				return
			}
		case <-dbTicker.C:
			if err := cm.saveState(); err != nil {
				log.Fatalln(err)
				return
			}
		case <-cm.quit:
			return
		}
	}
}

var dbLock sync.Mutex

func (cm *chainManager) saveState() (err error) {
	dbLock.Lock()
	defer dbLock.Unlock()
	db, err := sql.Open("sqlite3", "./chains.db")
	if err != nil {
		return
	}
	defer db.Close()
	return cm.c.SaveState(db)
}
