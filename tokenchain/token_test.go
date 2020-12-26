package tokenchain_test

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/hectorchu/gonano/wallet"
	"github.com/hectorchu/nano-token-server/tokenchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const rpcURL = "https://mynano.ninja/api/node"

func getAccount(i int) (a *wallet.Account) {
	seeds := []string{
		"52fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649",
		"dfaf7d4eba814bcb3a9926011d83e3fda34b8e11e635b3834a3e3cb5279a941e",
	}
	seed, _ := hex.DecodeString(seeds[i])
	w, _ := wallet.NewWallet(seed)
	w.RPC.URL = rpcURL
	a, _ = w.NewAccount(nil)
	return
}

func newChain(t *testing.T) (chain *tokenchain.Chain) {
	chain, err := tokenchain.NewChain(rpcURL)
	require.Nil(t, err)
	_, err = getAccount(0).Send(chain.Address(), big.NewInt(1))
	require.Nil(t, err)
	err = chain.WaitForOpen()
	require.Nil(t, err)
	return
}

func loadChain(t *testing.T, address string) (chain *tokenchain.Chain) {
	chain, err := tokenchain.LoadChain(address, rpcURL)
	require.Nil(t, err)
	err = chain.Parse()
	require.Nil(t, err)
	return
}

var supply = big.NewInt(1e9)

func genesis(t *testing.T, chain *tokenchain.Chain, a *wallet.Account) (token *tokenchain.Token) {
	token, err := tokenchain.TokenGenesis(chain, a, "TOKEN", supply, 5)
	require.Nil(t, err)
	return
}

func assertEqualChain(t *testing.T, c1, c2 *tokenchain.Chain) {
	assert.Equal(t, c1.Address(), c2.Address())
	tokens1, tokens2 := c1.Tokens(), c2.Tokens()
	assert.Len(t, tokens1, len(tokens2))
	for hash, t1 := range tokens1 {
		t2, ok := tokens2[hash]
		require.True(t, ok)
		assertEqualToken(t, t1, t2)
	}
}

func assertEqualToken(t *testing.T, t1, t2 *tokenchain.Token) {
	assert.Equal(t, t1.Name(), t2.Name())
	assert.Equal(t, t1.Supply(), t2.Supply())
	assert.Equal(t, t1.Decimals(), t2.Decimals())
	assert.Equal(t, t1.Hash(), t2.Hash())
	assert.Equal(t, t1.Balances(), t2.Balances())
}

func TestGenesis(t *testing.T) {
	chain := newChain(t)
	token := genesis(t, chain, getAccount(0))
	assert.Equal(t, supply, token.Balance(getAccount(0).Address()))
	assertEqualChain(t, chain, loadChain(t, chain.Address()))
}

func TestTransfer(t *testing.T) {
	chain := newChain(t)
	token := genesis(t, chain, getAccount(0))
	amount := big.NewInt(1000)
	_, err := token.Transfer(getAccount(0), getAccount(1).Address(), amount)
	require.Nil(t, err)
	assert.Equal(t, new(big.Int).Sub(supply, amount), token.Balance(getAccount(0).Address()))
	assert.Equal(t, amount, token.Balance(getAccount(1).Address()))
	assertEqualChain(t, chain, loadChain(t, chain.Address()))
}
