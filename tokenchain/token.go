package tokenchain

import (
	"errors"
	"math/big"

	"github.com/hectorchu/gonano/rpc"
	"github.com/hectorchu/gonano/wallet"
)

// Token represents a token.
type Token struct {
	name     string
	supply   *big.Int
	decimals byte
	hash     rpc.BlockHash
	balances map[string]*big.Int
}

func newToken(name string, supply *big.Int, decimals byte, hash rpc.BlockHash) *Token {
	return &Token{
		name:     name,
		supply:   supply,
		decimals: decimals,
		hash:     hash,
		balances: make(map[string]*big.Int),
	}
}

// Name returns the token name.
func (t *Token) Name() string {
	return t.name
}

// Supply returns the token supply.
func (t *Token) Supply() *big.Int {
	return t.supply
}

// Decimals returns the token decimals.
func (t *Token) Decimals() byte {
	return t.decimals
}

// Hash returns the block hash for the token.
func (t *Token) Hash() rpc.BlockHash {
	return t.hash
}

// Balances gets the token balances.
func (t *Token) Balances() (balances map[string]*big.Int) {
	balances = make(map[string]*big.Int)
	for account, balance := range t.balances {
		balances[account] = balance
	}
	return
}

// Balance gets the balance for account.
func (t *Token) Balance(account string) (balance *big.Int) {
	balance, ok := t.balances[account]
	if !ok {
		balance = &big.Int{}
	}
	return
}

func (t *Token) setBalance(account string, balance *big.Int) {
	t.balances[account] = balance
}

// Genesis initializes a new token on a chain.
func Genesis(c *Chain, a *wallet.Account, name string, supply *big.Int, decimals byte) (t *Token, err error) {
	if err = c.Parse(); err != nil {
		return
	}
	m := &genesisMessage{
		decimals: decimals,
		name:     name,
		supply:   supply,
	}
	if err = setData(a, m.serialize()); err != nil {
		return
	}
	hash, err := a.Send(c.a.Address(), big.NewInt(1))
	if err != nil {
		return
	}
	if hash, err = c.confirm(hash); err != nil {
		return
	}
	if err = c.Parse(); err != nil {
		return
	}
	return c.Token(hash)
}

// Transfer transfers an amount of tokens to another account.
func (t *Token) Transfer(c *Chain, a *wallet.Account, account string, amount *big.Int) (err error) {
	if err = c.Parse(); err != nil {
		return
	}
	balance := t.Balance(a.Address())
	if balance.Cmp(amount) < 0 {
		return errors.New("insufficient balance")
	}
	height, err := c.getHeight(t.hash)
	if err != nil {
		return
	}
	m := &transferMessage{
		token:  height,
		amount: amount,
	}
	if err = setData(a, m.serialize()); err != nil {
		return
	}
	if _, err = a.Send(account, big.NewInt(1)); err != nil {
		return
	}
	hash, err := a.Send(c.a.Address(), big.NewInt(1))
	if err != nil {
		return
	}
	if _, err = c.confirm(hash); err != nil {
		return
	}
	return c.Parse()
}

func (t *Token) doTransfer(src, dest string, amount *big.Int) (err error) {
	balance := t.Balance(src)
	if balance.Cmp(amount) < 0 {
		return errors.New("insufficient balance")
	}
	t.setBalance(src, balance.Sub(balance, amount))
	balance = t.Balance(dest)
	t.setBalance(dest, balance.Add(balance, amount))
	return
}
