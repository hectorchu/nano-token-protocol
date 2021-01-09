package main

import (
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
	chains map[string]*chainManager
}

func newChainsManager() *chainsManager {
	return &chainsManager{
		chains: make(map[string]*chainManager),
	}
}

func (cm *chainsManager) scanForChains(modifiedSince time.Time, rpcURL string) (err error) {
	log.Println("Scanning for chains...")
	client := rpc.Client{URL: rpcURL}
	account, err := util.PubkeyToAddress(make([]byte, 32))
	if err != nil {
		return
	}
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
			cm.addChain(c)
		}
		count += len(addresses)
		log.Println("Processed", count, "accounts")
		if len(addresses) < batchSize {
			break
		}
		account = addresses[len(addresses)-1]
	}
	return
}

func (cm *chainsManager) addChain(c *tokenchain.Chain) {
	if _, ok := cm.chains[c.Address()]; ok {
		return
	}
	cm.chains[c.Address()] = newChainManager(c)
}
