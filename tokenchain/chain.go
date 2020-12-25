package tokenchain

import (
	"bytes"
	"crypto/rand"
	"errors"
	"math/big"
	"time"

	"github.com/hectorchu/gonano/rpc"
	"github.com/hectorchu/gonano/util"
	"github.com/hectorchu/gonano/wallet"
)

// Chain represents a token chain.
type Chain struct {
	w        *wallet.Wallet
	a        *wallet.Account
	frontier rpc.BlockHash
	tokens   map[uint32]*Token
}

// NewChain initializes a new chain.
func NewChain(rpcURL string) (c *Chain, err error) {
	seed := make([]byte, 32)
	if _, err = rand.Read(seed); err != nil {
		return
	}
	if c, err = newFromSeed(seed, rpcURL); err != nil {
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
	if c, err = newFromSeed(seed, rpcURL); err != nil {
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

func newFromSeed(seed []byte, rpcURL string) (c *Chain, err error) {
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
		w:      w,
		a:      a,
		tokens: make(map[uint32]*Token),
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
		case balance.Cmp(big.NewInt(0)) > 0:
			return nil
		case pending.Cmp(big.NewInt(0)) > 0:
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
		switch m := m.(type) {
		case *genesisMessage:
			t := newToken(m.name, m.supply, m.decimals, hash)
			t.setBalance(info.BlockAccount, m.supply)
			c.tokens[height] = t
		case *transferMessage:
			t, ok := c.tokens[m.token]
			if !ok {
				break
			}
			prev, err := c.rpc().BlockInfo(info.Contents.Previous)
			if err != nil {
				return err
			}
			if prev.Subtype != "send" {
				break
			}
			if prev.Contents.Representative != info.Contents.Representative {
				break
			}
			if err = t.doTransfer(info.BlockAccount, prev.Contents.LinkAsAccount, m.amount); err != nil {
				break
			}
		}
		c.frontier = hash
	}
	return
}

func (c *Chain) getHeight(hash rpc.BlockHash) (height uint32, err error) {
	info, err := c.rpc().BlockInfo(hash)
	if err != nil {
		return
	}
	if info.BlockAccount == c.Address() {
		height = uint32(info.Height)
	} else {
		err = errors.New("block is not on this chain")
	}
	return
}

// Tokens gets the chain's tokens.
func (c *Chain) Tokens() (tokens map[string]*Token) {
	tokens = make(map[string]*Token)
	for _, token := range c.tokens {
		tokens[string(token.Hash())] = token
	}
	return
}

// Token gets the token at the specified block hash.
func (c *Chain) Token(hash rpc.BlockHash) (token *Token, err error) {
	height, err := c.getHeight(hash)
	if err != nil {
		return
	}
	token, ok := c.tokens[height]
	if !ok {
		err = errors.New("token not found")
	}
	return
}
