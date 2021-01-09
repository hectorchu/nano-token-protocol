package main

import (
	"database/sql"
	"log"
	"time"

	"github.com/hectorchu/nano-token-protocol/tokenchain"
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
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := cm.c.Parse(); err != nil {
				log.Fatalln(err)
			}
			if err := withDB(func(db *sql.DB) error {
				return cm.c.SaveState(db)
			}); err != nil {
				log.Fatalln(err)
			}
		case <-cm.quit:
			return
		}
	}
}
