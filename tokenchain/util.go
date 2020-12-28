package tokenchain

import (
	"errors"
	"math/big"

	"github.com/hectorchu/gonano/util"
	"github.com/hectorchu/gonano/wallet"
)

func setData(a *wallet.Account, data []byte) (err error) {
	address, err := util.PubkeyToAddress(data)
	if err != nil {
		return
	}
	return a.SetRep(address)
}

func checkPositive(x *big.Int) (err error) {
	if x.Sign() < 0 {
		err = errors.New("Amount is negative")
	}
	return
}
