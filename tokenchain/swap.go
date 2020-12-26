package tokenchain

import (
	"errors"
	"math/big"

	"github.com/hectorchu/gonano/rpc"
	"github.com/hectorchu/gonano/wallet"
)

// Swap represents a token swap.
type Swap struct {
	c           *Chain
	hash        rpc.BlockHash
	left, right SwapSide
	inactive    bool
}

// SwapSide represents a side of the swap.
type SwapSide struct {
	Account string
	Token   *Token
	Amount  *big.Int
}

// Hash returns the block hash of the swap.
func (s *Swap) Hash() rpc.BlockHash {
	return s.hash
}

// Left returns the left side of the swap.
func (s *Swap) Left() (ss SwapSide) {
	ss = s.left
	if ss.Amount != nil {
		ss.Amount = new(big.Int).Set(ss.Amount)
	}
	return
}

// Right returns the right side of the swap.
func (s *Swap) Right() (ss SwapSide) {
	ss = s.right
	if ss.Amount != nil {
		ss.Amount = new(big.Int).Set(ss.Amount)
	}
	return
}

// Active returns whether the swap is active.
func (s *Swap) Active() bool {
	return !s.inactive
}

// ProposeSwap proposes a swap on-chain.
func ProposeSwap(c *Chain, a *wallet.Account, counterparty string, t *Token, amount *big.Int) (s *Swap, err error) {
	if err = c.Parse(); err != nil {
		return
	}
	if err = t.checkBalance(a.Address(), amount); err != nil {
		return
	}
	height, err := c.getHeight(t.hash)
	if err != nil {
		return
	}
	hash, err := c.send(a, counterparty, &swapProposeMessage{
		token:  height,
		amount: amount,
	})
	if err != nil {
		return
	}
	return c.Swap(hash)
}

func (m *swapProposeMessage) process(c *Chain, hash rpc.BlockHash, height uint32, info rpc.BlockInfo) (valid bool, err error) {
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
	c.swaps[height] = &Swap{
		c:    c,
		hash: hash,
		left: SwapSide{
			Account: info.BlockAccount,
			Token:   t,
			Amount:  m.amount,
		},
		right: SwapSide{
			Account: destination,
		},
	}
	return
}

// Accept accepts a swap proposal.
func (s *Swap) Accept(a *wallet.Account, t *Token, amount *big.Int) (hash rpc.BlockHash, err error) {
	if err = s.c.Parse(); err != nil {
		return
	}
	if err = s.checkAccept(a.Address(), t, amount); err != nil {
		return
	}
	swap, err := s.c.getHeight(s.hash)
	if err != nil {
		return
	}
	token, err := t.c.getHeight(t.hash)
	if err != nil {
		return
	}
	return s.c.send(a, "", &swapAcceptMessage{
		swap:   swap,
		token:  token,
		amount: amount,
	})
}

func (s *Swap) checkAccept(account string, t *Token, amount *big.Int) (err error) {
	if s.inactive {
		return errors.New("Swap is inactive")
	}
	if s.right.Token != nil {
		return errors.New("Swap already accepted")
	}
	if account != s.right.Account {
		return errors.New("Must accept swap with right account")
	}
	if s.c != t.c {
		return errors.New("Chain mismatch")
	}
	return t.checkBalance(account, amount)
}

func (m *swapAcceptMessage) process(c *Chain, hash rpc.BlockHash, height uint32, info rpc.BlockInfo) (valid bool, err error) {
	s, ok := c.swaps[m.swap]
	if !ok {
		return
	}
	t, ok := c.tokens[m.token]
	if !ok {
		return
	}
	if s.checkAccept(info.BlockAccount, t, m.amount) != nil {
		return
	}
	s.right = SwapSide{
		Account: info.BlockAccount,
		Token:   t,
		Amount:  m.amount,
	}
	return true, nil
}

// Confirm confirms a swap proposal.
func (s *Swap) Confirm(a *wallet.Account) (hash rpc.BlockHash, err error) {
	if err = s.c.Parse(); err != nil {
		return
	}
	if err = s.checkConfirm(a.Address()); err != nil {
		return
	}
	height, err := s.c.getHeight(s.hash)
	if err != nil {
		return
	}
	return s.c.send(a, "", &swapConfirmMessage{swap: height})
}

func (s *Swap) checkConfirm(account string) (err error) {
	if s.inactive {
		return errors.New("Swap is inactive")
	}
	if s.right.Token == nil {
		return errors.New("Swap not accepted")
	}
	if account != s.left.Account {
		return errors.New("Must confirm swap with left account")
	}
	if err = s.left.Token.checkBalance(s.left.Account, s.left.Amount); err != nil {
		return
	}
	if err = s.right.Token.checkBalance(s.right.Account, s.right.Amount); err != nil {
		return
	}
	return
}

func (m *swapConfirmMessage) process(c *Chain, hash rpc.BlockHash, height uint32, info rpc.BlockInfo) (valid bool, err error) {
	s, ok := c.swaps[m.swap]
	if !ok {
		return
	}
	if s.checkConfirm(info.BlockAccount) != nil {
		return
	}
	balance := s.left.Token.Balance(s.left.Account)
	s.left.Token.setBalance(s.left.Account, balance.Sub(balance, s.left.Amount))
	balance = s.left.Token.Balance(s.right.Account)
	s.left.Token.setBalance(s.right.Account, balance.Add(balance, s.left.Amount))
	balance = s.right.Token.Balance(s.right.Account)
	s.right.Token.setBalance(s.right.Account, balance.Sub(balance, s.right.Amount))
	balance = s.right.Token.Balance(s.left.Account)
	s.right.Token.setBalance(s.left.Account, balance.Add(balance, s.right.Amount))
	s.inactive = true
	delete(c.swaps, m.swap)
	return true, nil
}

// Cancel cancels a swap proposal.
func (s *Swap) Cancel(a *wallet.Account) (hash rpc.BlockHash, err error) {
	if err = s.c.Parse(); err != nil {
		return
	}
	if err = s.checkCancel(a.Address()); err != nil {
		return
	}
	height, err := s.c.getHeight(s.hash)
	if err != nil {
		return
	}
	return s.c.send(a, "", &swapCancelMessage{swap: height})
}

func (s *Swap) checkCancel(account string) (err error) {
	if s.inactive {
		return errors.New("Swap is inactive")
	}
	if account != s.left.Account && account != s.right.Account {
		return errors.New("Must cancel swap with left or right account")
	}
	return
}

func (m *swapCancelMessage) process(c *Chain, hash rpc.BlockHash, height uint32, info rpc.BlockInfo) (valid bool, err error) {
	s, ok := c.swaps[m.swap]
	if !ok {
		return
	}
	if s.checkCancel(info.BlockAccount) != nil {
		return
	}
	s.inactive = true
	delete(c.swaps, m.swap)
	return true, nil
}
