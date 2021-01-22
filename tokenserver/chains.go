package main

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hectorchu/gonano/rpc"
	"github.com/hectorchu/gonano/util"
	"github.com/hectorchu/gonano/websocket"
	"github.com/hectorchu/nano-token-protocol/tokenchain"
	_ "github.com/mattn/go-sqlite3"
)

type chainManager struct {
	m           sync.Mutex
	chains      map[string]*tokenchain.Chain
	lastUpdated time.Time
}

func newChainManager(rpcURL, wsURL string) (cm *chainManager, err error) {
	cm = &chainManager{
		chains:      make(map[string]*tokenchain.Chain),
		lastUpdated: time.Date(2020, 12, 25, 0, 0, 0, 0, time.UTC),
	}
	if _, err := os.Stat("./chains.db"); err == nil {
		if err = withDB(func(db *sql.DB) error { return cm.loadState(db, rpcURL) }); err != nil {
			return nil, err
		}
	}
	log.Println("Catching up...")
	for {
		lastUpdated := time.Now().UTC()
		if lastUpdated.Sub(cm.lastUpdated) < 5*time.Minute {
			break
		}
		if err = cm.scanForChains(rpcURL); err != nil {
			return
		}
		cm.lastUpdated = lastUpdated
		if err = withDB(cm.saveState); err != nil {
			return
		}
	}
	ws := &websocket.Client{URL: wsURL}
	if err = ws.Connect(); err != nil {
		return
	}
	messages := make(chan interface{}, 1e4)
	go func() {
		for {
			messages <- <-ws.Messages
		}
	}()
	if err = cm.scanForChains(rpcURL); err != nil {
		return
	}
	log.Println("...done")
	go cm.loop(messages, rpcURL)
	return
}

func (cm *chainManager) withLock(cb func(*chainManager)) {
	cm.m.Lock()
	cb(cm)
	cm.m.Unlock()
}

func (cm *chainManager) loop(messages <-chan interface{}, rpcURL string) {
	for {
		switch m := (<-messages).(type) {
		case *websocket.Confirmation:
			if err := cm.scanForChain(m.Block, rpcURL); err != nil {
				log.Fatalln(err)
			}
			cm.lastUpdated = m.Time
		}
		if err := withDB(cm.saveState); err != nil {
			log.Fatalln(err)
		}
	}
}

func (cm *chainManager) scanForChains(rpcURL string) (err error) {
	client := rpc.Client{URL: rpcURL}
	account, err := util.PubkeyToAddress(make([]byte, 32))
	if err != nil {
		return
	}
	for {
		const batchSize = 1e4
		accounts, err := client.Ledger(account, batchSize, cm.lastUpdated)
		if err != nil {
			return err
		}
		if len(accounts) == 0 {
			break
		}
		var (
			addresses = make([]string, 0, len(accounts))
			hashes    = make([]rpc.BlockHash, 0, len(accounts))
		)
		for address, info := range accounts {
			addresses = append(addresses, address)
			hashes = append(hashes, info.OpenBlock)
		}
		sort.Slice(addresses, func(i, j int) bool {
			return strings.Compare(addresses[i], addresses[j]) < 0
		})
		blocks, err := client.Blocks(hashes)
		if err != nil {
			return err
		}
		for address, info := range accounts {
			if address == account {
				continue
			}
			if err = cm.scanForChain(blocks[info.OpenBlock.String()], rpcURL); err != nil {
				return err
			}
		}
		if len(addresses) < batchSize {
			break
		}
		account = addresses[len(addresses)-1]
	}
	return
}

func (cm *chainManager) scanForChain(block *rpc.Block, rpcURL string) (err error) {
	c, ok := cm.chains[block.Account]
	if !ok {
		if bytes.Count(block.Previous, []byte{0}) != len(block.Previous) {
			return
		}
		seed, err := util.AddressToPubkey(block.Representative)
		if err != nil {
			return err
		}
		if c, err = tokenchain.NewChainFromSeed(seed, rpcURL); err != nil {
			return err
		}
		if c.Address() != block.Account {
			return err
		}
		cm.m.Lock()
		cm.chains[c.Address()] = c
	} else {
		cm.m.Lock()
	}
	err = c.Parse()
	cm.m.Unlock()
	if err != nil {
		return
	}
	return withDB(c.SaveState)
}

func withDB(cb func(*sql.DB) error) (err error) {
	db, err := sql.Open("sqlite3", "./chains.db")
	if err != nil {
		return
	}
	defer db.Close()
	return cb(db)
}

func (cm *chainManager) loadState(db *sql.DB, rpcURL string) (err error) {
	rows, err := db.Query("SELECT seed FROM chains")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var seedStr string
		if err := rows.Scan(&seedStr); err != nil {
			return err
		}
		seed, err := hex.DecodeString(seedStr)
		if err != nil {
			return err
		}
		c, err := tokenchain.NewChainFromSeed(seed, rpcURL)
		if err != nil {
			return err
		}
		if err = c.LoadState(db); err != nil {
			return err
		}
		cm.chains[c.Address()] = c
	}
	if err = rows.Err(); err != nil {
		return
	}
	var lastUpdated int64
	err = db.QueryRow("SELECT lastUpdated FROM chain_manager WHERE id = 1").Scan(&lastUpdated)
	if err != nil {
		return
	}
	cm.lastUpdated = time.Unix(lastUpdated, 0).UTC()
	return
}

func (cm *chainManager) saveState(db *sql.DB) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	_, err = tx.Exec("CREATE TABLE IF NOT EXISTS chain_manager (id INTEGER PRIMARY KEY, lastUpdated INTEGER)")
	if err != nil {
		tx.Rollback()
		return
	}
	_, err = tx.Exec("REPLACE INTO chain_manager (id, lastUpdated) VALUES (1, ?)", cm.lastUpdated.Unix())
	if err != nil {
		tx.Rollback()
		return
	}
	return tx.Commit()
}
