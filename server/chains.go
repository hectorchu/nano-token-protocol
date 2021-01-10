package main

import (
	"database/sql"
	"encoding/hex"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/hectorchu/gonano/rpc"
	"github.com/hectorchu/gonano/util"
	"github.com/hectorchu/nano-token-protocol/tokenchain"
	_ "github.com/mattn/go-sqlite3"
)

type chainManager struct {
	chains      map[string]*tokenchain.Chain
	lastUpdated time.Time
}

func newChainManager(rpcURL string) (cm *chainManager) {
	cm = &chainManager{
		chains: make(map[string]*tokenchain.Chain),
	}
	go cm.loop(rpcURL)
	return
}

func (cm *chainManager) loop(rpcURL string) {
	if err := cm.scanForChains(rpcURL); err != nil {
		log.Fatalln(err)
	}
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := cm.scanForChains(rpcURL); err != nil {
				log.Fatalln(err)
			}
		}
	}
}

func (cm *chainManager) scanForChains(rpcURL string) (err error) {
	client := rpc.Client{URL: rpcURL}
	account, err := util.PubkeyToAddress(make([]byte, 32))
	if err != nil {
		return
	}
	modifiedSince := cm.lastUpdated
	if modifiedSince.IsZero() {
		modifiedSince = time.Date(2020, 12, 25, 0, 0, 0, 0, time.UTC)
	}
	cm.lastUpdated = time.Now().UTC()
	for {
		const batchSize = 1e4
		accounts, err := client.Ledger(account, batchSize, modifiedSince)
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
			c, ok := cm.chains[address]
			if !ok {
				block := blocks[strings.ToUpper(hex.EncodeToString(info.OpenBlock))]
				seed, err := util.AddressToPubkey(block.Representative)
				if err != nil {
					return err
				}
				if c, err = tokenchain.NewChainFromSeed(seed, rpcURL); err != nil {
					return err
				}
				if c.Address() != address {
					continue
				}
				cm.chains[c.Address()] = c
			}
			if err = c.Parse(); err != nil {
				return err
			}
			if err = withDB(func(db *sql.DB) error { return c.SaveState(db) }); err != nil {
				return err
			}
		}
		if len(addresses) < batchSize {
			break
		}
		account = addresses[len(addresses)-1]
	}
	return withDB(func(db *sql.DB) error { return cm.saveState(db) })
}

func withDB(cb func(*sql.DB) error) (err error) {
	db, err := sql.Open("sqlite3", "./chains.db")
	if err != nil {
		return
	}
	defer db.Close()
	return cb(db)
}

func (cm *chainManager) saveState(db *sql.DB) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS chain_manager (id INTEGER PRIMARY KEY, lastUpdated INTEGER)`)
	if err != nil {
		tx.Rollback()
		return
	}
	_, err = tx.Exec("REPLACE INTO chain_manager (id, lastUpdated) VALUES (?, ?)", 1, cm.lastUpdated.Unix())
	if err != nil {
		tx.Rollback()
		return
	}
	return tx.Commit()
}
