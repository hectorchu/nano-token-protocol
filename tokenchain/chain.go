package tokenchain

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/hectorchu/gonano/rpc"
	"github.com/hectorchu/gonano/util"
	"github.com/hectorchu/gonano/wallet"
)

// Chain represents a token chain.
type Chain struct {
	seed     []byte
	w        *wallet.Wallet
	a        *wallet.Account
	frontier rpc.BlockHash
	tokens   map[uint32]*Token
	swaps    map[uint32]*Swap
}

// NewChain initializes a new chain.
func NewChain(rpcURL string) (c *Chain, err error) {
	seed := make([]byte, 32)
	if _, err = rand.Read(seed); err != nil {
		return
	}
	if c, err = NewChainFromSeed(seed, rpcURL); err != nil {
		return
	}
	return c, setData(c.a, seed)
}

// LoadChain loads a chain at an address.
func LoadChain(address, rpcURL string) (c *Chain, err error) {
	client := rpc.Client{URL: rpcURL}
	info, err := client.AccountInfo(address)
	if err != nil {
		return
	}
	block, err := client.BlockInfo(info.OpenBlock)
	if err != nil {
		return
	}
	seed, err := util.AddressToPubkey(block.Contents.Representative)
	if err != nil {
		return
	}
	if c, err = NewChainFromSeed(seed, rpcURL); err != nil {
		return
	}
	if c.Address() != address {
		err = errors.New("Address does not match")
	}
	return
}

// Address returns the address of the chain.
func (c *Chain) Address() string {
	return c.a.Address()
}

// NewChainFromSeed initializes a new chain from a seed.
func NewChainFromSeed(seed []byte, rpcURL string) (c *Chain, err error) {
	w, err := wallet.NewWallet(seed)
	if err != nil {
		return
	}
	w.RPC.URL = rpcURL
	a, err := w.NewAccount(nil)
	if err != nil {
		return
	}
	c = &Chain{
		seed:   seed,
		w:      w,
		a:      a,
		tokens: make(map[uint32]*Token),
		swaps:  make(map[uint32]*Swap),
	}
	return
}

// WaitForOpen waits for the open block.
func (c *Chain) WaitForOpen() (err error) {
	for {
		balance, pending, err := c.a.Balance()
		switch {
		case err != nil:
			return err
		case balance.Sign() > 0:
			return nil
		case pending.Sign() > 0:
			if err = c.a.ReceivePendings(); err != nil {
				return err
			}
		default:
			time.Sleep(5 * time.Second)
		}
	}
}

func (c *Chain) rpc() *rpc.Client {
	return &c.w.RPC
}

func (c *Chain) send(a *wallet.Account, destination *string, m message) (hash rpc.BlockHash, err error) {
	if err = setData(a, m.serialize()); err != nil {
		return
	}
	if destination != nil {
		if _, err = a.Send(*destination, big.NewInt(1)); err != nil {
			return
		}
	}
	if hash, err = a.Send(c.Address(), big.NewInt(1)); err != nil {
		return
	}
	if hash, err = c.confirm(hash); err != nil {
		return
	}
	return hash, c.Parse()
}

func (c *Chain) confirm(link rpc.BlockHash) (hash rpc.BlockHash, err error) {
	for {
		if hash, err = c.a.ReceivePending(link); err != nil {
			switch err.Error() {
			case "Fork":
				continue
			case "Unreceivable":
				var hashes []rpc.BlockHash
				if hashes, err = c.rpc().Successors(c.frontier, -1); err != nil {
					return
				}
				for _, hash = range hashes[1:] {
					var block rpc.BlockInfo
					if block, err = c.rpc().BlockInfo(hash); err != nil {
						return
					}
					if bytes.Equal(block.Contents.Link, link) {
						break
					}
				}
			}
		}
		return
	}
}

// Parse parses the chain for tokens.
func (c *Chain) Parse() (err error) {
	if c.frontier == nil {
		info, err := c.rpc().AccountInfo(c.Address())
		if err != nil {
			return err
		}
		c.frontier = info.OpenBlock
	}
	hashes, err := c.rpc().Successors(c.frontier, -1)
	if err != nil {
		return
	}
	for _, hash := range hashes[1:] {
		info, err := c.rpc().BlockInfo(hash)
		if err != nil {
			return err
		}
		if info.Subtype != "receive" {
			c.frontier = hash
			continue
		}
		height := uint32(info.Height)
		info, err = c.rpc().BlockInfo(info.Contents.Link)
		if err != nil {
			return err
		}
		data, err := util.AddressToPubkey(info.Contents.Representative)
		if err != nil {
			return err
		}
		m, err := parseMessage(data)
		if err != nil {
			c.frontier = hash
			continue
		}
		if _, err = m.process(c, hash, height, info); err != nil {
			return err
		}
		c.frontier = hash
	}
	return
}

func (c *Chain) getDestination(block *rpc.Block) (account string, valid bool, err error) {
	info, err := c.rpc().BlockInfo(block.Previous)
	if err != nil {
		return
	}
	if info.Subtype != "send" {
		return
	}
	if info.Contents.Representative != block.Representative {
		return
	}
	return info.Contents.LinkAsAccount, true, nil
}

func (c *Chain) getHeight(hash rpc.BlockHash) (height uint32, err error) {
	info, err := c.rpc().BlockInfo(hash)
	if err != nil {
		return
	}
	if info.BlockAccount == c.Address() {
		height = uint32(info.Height)
	} else {
		err = errors.New("Block is not on this chain")
	}
	return
}

// Tokens gets the chain's tokens.
func (c *Chain) Tokens() (tokens map[string]*Token) {
	tokens = make(map[string]*Token)
	for _, t := range c.tokens {
		tokens[string(t.Hash())] = t
	}
	return
}

// Token gets the token at the specified block hash.
func (c *Chain) Token(hash rpc.BlockHash) (t *Token, err error) {
	height, err := c.getHeight(hash)
	if err != nil {
		return
	}
	t, ok := c.tokens[height]
	if !ok {
		err = errors.New("Token not found")
	}
	return
}

// Swap gets the swap at the specified block hash.
func (c *Chain) Swap(hash rpc.BlockHash) (s *Swap, err error) {
	height, err := c.getHeight(hash)
	if err != nil {
		return
	}
	s, ok := c.swaps[height]
	if !ok {
		err = errors.New("Swap not found")
	}
	return
}

// SaveState saves the chain state to the DB.
func (c *Chain) SaveState(db *sql.DB) (err error) {
	var (
		seed         = strings.ToUpper(hex.EncodeToString(c.seed))
		frontier     = strings.ToUpper(hex.EncodeToString(c.frontier))
		prevFrontier string
	)
	err = db.QueryRow("SELECT frontier FROM chains WHERE seed = ?", seed).Scan(&prevFrontier)
	if err == nil && frontier == prevFrontier {
		return
	}
	tx, err := db.Begin()
	if err != nil {
		return
	}
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS chains (seed TEXT PRIMARY KEY, frontier TEXT)`)
	if err != nil {
		tx.Rollback()
		return
	}
	_, err = tx.Exec("REPLACE INTO chains (seed, frontier) VALUES (?, ?)", seed, frontier)
	if err != nil {
		tx.Rollback()
		return
	}
	for _, t := range c.tokens {
		if err = t.saveState(tx); err != nil {
			tx.Rollback()
			return
		}
	}
	return tx.Commit()
}
