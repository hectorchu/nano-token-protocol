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
)

type chainsManager struct {
	chains      map[string]*chainManager
	lastUpdated time.Time
	quit        chan bool
}

func newChainsManager(rpcURL string) (cm *chainsManager) {
	cm = &chainsManager{
		chains: make(map[string]*chainManager),
		quit:   make(chan bool),
	}
	go cm.loop(rpcURL)
	return
}

func (cm *chainsManager) loop(rpcURL string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := cm.scanForChains(rpcURL); err != nil {
				log.Fatalln(err)
			}
		case <-cm.quit:
			return
		}
	}
}

func (cm *chainsManager) scanForChains(rpcURL string) (err error) {
	log.Println("Scanning for chains...")
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
	for count := 0; ; {
		const batchSize = 1e4
		accounts, err := client.Ledger(account, batchSize, modifiedSince)
		if err != nil {
			return err
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
			block := blocks[strings.ToUpper(hex.EncodeToString(info.OpenBlock))]
			seed, err := util.AddressToPubkey(block.Representative)
			if err != nil {
				return err
			}
			c, err := tokenchain.NewChainFromSeed(seed, rpcURL)
			if err != nil {
				return err
			}
			if c.Address() != address {
				continue
			}
			if err = cm.addChain(c); err != nil {
				return err
			}
		}
		count += len(addresses)
		log.Println("Processed", count, "accounts")
		if len(addresses) < batchSize {
			break
		}
		account = addresses[len(addresses)-1]
	}
	return withDB(func(db *sql.DB) error {
		return cm.saveState(db)
	})
}

func (cm *chainsManager) addChain(c *tokenchain.Chain) (err error) {
	if _, ok := cm.chains[c.Address()]; ok {
		return
	}
	if err = c.Parse(); err != nil {
		return
	}
	if err = withDB(func(db *sql.DB) error {
		return c.SaveState(db)
	}); err != nil {
		return
	}
	cm.chains[c.Address()] = newChainManager(c)
	return
}

func (cm *chainsManager) saveState(db *sql.DB) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS manager (id INTEGER PRIMARY KEY, lastUpdated INTEGER)`)
	if err != nil {
		tx.Rollback()
		return
	}
	_, err = tx.Exec("REPLACE INTO manager (id, lastUpdated) VALUES (?, ?)", 1, cm.lastUpdated.Unix())
	if err != nil {
		tx.Rollback()
		return
	}
	return tx.Commit()
}
