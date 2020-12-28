package tokenchain

import (
	"errors"
	"math/big"

	"github.com/hectorchu/gonano/rpc"
	"github.com/hectorchu/gonano/wallet"
)

// Token represents a token.
type Token struct {
	c        *Chain
	hash     rpc.BlockHash
	name     string
	supply   *big.Int
	decimals byte
	balances map[string]*big.Int
}

// Hash returns the block hash of the token.
func (t *Token) Hash() rpc.BlockHash {
	return t.hash
}

// Name returns the token name.
func (t *Token) Name() string {
	return t.name
}

// Supply returns the token supply.
func (t *Token) Supply() *big.Int {
	return new(big.Int).Set(t.supply)
}

// Decimals returns the token decimals.
func (t *Token) Decimals() byte {
	return t.decimals
}

// Balances gets the token balances.
func (t *Token) Balances() (balances map[string]*big.Int) {
	balances = make(map[string]*big.Int)
	for account, balance := range t.balances {
		balances[account] = new(big.Int).Set(balance)
	}
	return
}

// Balance gets the balance for account.
func (t *Token) Balance(account string) (balance *big.Int) {
	balance, ok := t.balances[account]
	if !ok {
		return new(big.Int)
	}
	return new(big.Int).Set(balance)
}

func (t *Token) setBalance(account string, balance *big.Int) {
	t.balances[account] = balance
}

func (t *Token) checkBalance(account string, amount *big.Int) (err error) {
	if err = checkPositive(amount); err != nil {
		return
	}
	if t.Balance(account).Cmp(amount) < 0 {
		err = errors.New("Insufficient balance")
	}
	return
}

// TokenGenesis initializes a new token on a chain.
func TokenGenesis(c *Chain, a *wallet.Account, name string, supply *big.Int, decimals byte) (t *Token, err error) {
	if err = c.Parse(); err != nil {
		return
	}
	if err = checkPositive(supply); err != nil {
		return
	}
	hash, err := c.send(a, "", &genesisMessage{
		decimals: decimals,
		name:     name,
		supply:   supply,
	})
	if err != nil {
		return
	}
	return c.Token(hash)
}

func (m *genesisMessage) process(c *Chain, hash rpc.BlockHash, height uint32, info rpc.BlockInfo) (valid bool, err error) {
	if err = checkPositive(m.supply); err != nil {
		return
	}
	t := &Token{
		c:        c,
		hash:     hash,
		name:     m.name,
		supply:   m.supply,
		decimals: m.decimals,
		balances: make(map[string]*big.Int),
	}
	t.setBalance(info.BlockAccount, m.supply)
	c.tokens[height] = t
	return true, nil
}

// Transfer transfers an amount of tokens to another account.
func (t *Token) Transfer(a *wallet.Account, account string, amount *big.Int) (hash rpc.BlockHash, err error) {
	if err = t.c.Parse(); err != nil {
		return
	}
	if err = t.checkBalance(a.Address(), amount); err != nil {
		return
	}
	height, err := t.c.getHeight(t.hash)
	if err != nil {
		return
	}
	return t.c.send(a, account, &transferMessage{
		token:  height,
		amount: amount,
	})
}

func (m *transferMessage) process(c *Chain, hash rpc.BlockHash, height uint32, info rpc.BlockInfo) (valid bool, err error) {
	t, ok := c.tokens[m.token]
	if !ok {
		return
	}
	if t.checkBalance(info.BlockAccount, m.amount) != nil {
		return
	}
	destination, valid, err := c.getDestination(info.Contents)
	if !valid {
		return
	}
	balance := t.Balance(info.BlockAccount)
	t.setBalance(info.BlockAccount, balance.Sub(balance, m.amount))
	balance = t.Balance(destination)
	t.setBalance(destination, balance.Add(balance, m.amount))
	return
}
