package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func rpcHandler(w http.ResponseWriter, r *http.Request) {
	var (
		buf bytes.Buffer
		v   struct{ Action string }
	)
	io.Copy(&buf, r.Body)
	r.Body.Close()
	if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
		return
	}
	var result map[string]interface{}
	switch v.Action {
	case "tokens":
		result = getTokens()
	case "token":
		result = getToken(&buf)
	case "token_balances":
		result = getTokenBalances(&buf)
	case "token_balance":
		result = getTokenBalance(&buf)
	}
	buf.Reset()
	json.NewEncoder(&buf).Encode(result)
	io.Copy(w, &buf)
}

func getTokens() (result map[string]interface{}) {
	result = make(map[string]interface{})
	chainMan.withLock(func(cm *chainManager) {
		for _, c := range cm.chains {
			for _, t := range c.Tokens() {
				hash := strings.ToUpper(hex.EncodeToString(t.Hash()))
				result[hash] = struct{ Name, Supply, Decimals string }{
					Name:     t.Name(),
					Supply:   t.Supply().String(),
					Decimals: strconv.Itoa(int(t.Decimals())),
				}
			}
		}
	})
	return
}

func getToken(buf *bytes.Buffer) (result map[string]interface{}) {
	result = make(map[string]interface{})
	var v struct{ Hash string }
	if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
		result["error"] = "Unable to decode request"
		return
	}
	hash, err := hex.DecodeString(v.Hash)
	if err != nil {
		result["error"] = "Unable to decode hash"
		return
	}
	chainMan.withLock(func(cm *chainManager) {
		for _, c := range cm.chains {
			if t, err := c.Token(hash); err == nil {
				result["Name"] = t.Name()
				result["Supply"] = t.Supply().String()
				result["Decimals"] = strconv.Itoa(int(t.Decimals()))
				return
			}
		}
		result["error"] = "Token not found"
	})
	return
}

func getTokenBalances(buf *bytes.Buffer) (result map[string]interface{}) {
	result = make(map[string]interface{})
	var v struct{ Hash string }
	if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
		result["error"] = "Unable to decode request"
		return
	}
	hash, err := hex.DecodeString(v.Hash)
	if err != nil {
		result["error"] = "Unable to decode hash"
		return
	}
	chainMan.withLock(func(cm *chainManager) {
		for _, c := range cm.chains {
			if t, err := c.Token(hash); err == nil {
				for account, balance := range t.Balances() {
					result[account] = balance.String()
				}
				return
			}
		}
		result["error"] = "Token not found"
	})
	return
}

func getTokenBalance(buf *bytes.Buffer) (result map[string]interface{}) {
	result = make(map[string]interface{})
	var v struct{ Hash, Account string }
	if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
		result["error"] = "Unable to decode request"
		return
	}
	hash, err := hex.DecodeString(v.Hash)
	if err != nil {
		result["error"] = "Unable to decode hash"
		return
	}
	chainMan.withLock(func(cm *chainManager) {
		for _, c := range cm.chains {
			if t, err := c.Token(hash); err == nil {
				result["Balance"] = t.Balance(v.Account).String()
				return
			}
		}
		result["error"] = "Token not found"
	})
	return
}
