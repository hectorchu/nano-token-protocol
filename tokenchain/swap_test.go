package tokenchain_test

import (
	"math/big"
	"testing"

	"github.com/hectorchu/nano-token-protocol/tokenchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwap(t *testing.T) {
	var (
		chain   = newChain(t)
		token1  = genesis(t, chain, getAccount(0))
		token2  = genesis(t, chain, getAccount(1))
		amount1 = big.NewInt(1000)
		amount2 = big.NewInt(2000)
	)
	swap, err := tokenchain.ProposeSwap(chain, getAccount(0), getAccount(1).Address(), token1, amount1)
	require.Nil(t, err)
	assert.Equal(t, getAccount(0).Address(), swap.Left().Account)
	assert.Equal(t, getAccount(1).Address(), swap.Right().Account)
	assert.Equal(t, token1, swap.Left().Token)
	assert.Equal(t, amount1, swap.Left().Amount)
	_, err = swap.Accept(getAccount(1), token2, amount2)
	require.Nil(t, err)
	assert.Equal(t, getAccount(1).Address(), swap.Right().Account)
	assert.Equal(t, token2, swap.Right().Token)
	assert.Equal(t, amount2, swap.Right().Amount)
	_, err = swap.Confirm(getAccount(1))
	assert.NotNil(t, err)
	_, err = swap.Confirm(getAccount(0))
	require.Nil(t, err)
	assert.False(t, swap.Active())
	_, err = chain.Swap(swap.Hash())
	assert.NotNil(t, err)
	assert.Equal(t, new(big.Int).Sub(supply, amount1), token1.Balance(getAccount(0).Address()))
	assert.Equal(t, amount1, token1.Balance(getAccount(1).Address()))
	assert.Equal(t, new(big.Int).Sub(supply, amount2), token2.Balance(getAccount(1).Address()))
	assert.Equal(t, amount2, token2.Balance(getAccount(0).Address()))
	assertEqualChain(t, chain, loadChain(t, chain.Address()))
}
