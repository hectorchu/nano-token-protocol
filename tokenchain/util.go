package tokenchain

import (
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
