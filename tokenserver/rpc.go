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

func rpcHandler(cm *chainManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
			result = getTokens(cm)
		case "token":
			result = getToken(cm, &buf)
		case "token_balances":
			result = getTokenBalances(cm, &buf)
		case "token_balance":
			result = getTokenBalance(cm, &buf)
		}
		json.NewEncoder(w).Encode(result)
	}
}

func getTokens(cm *chainManager) (result map[string]interface{}) {
	result = make(map[string]interface{})
	cm.withLock(func() {
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

func getToken(cm *chainManager, buf *bytes.Buffer) (result map[string]interface{}) {
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
	cm.withLock(func() {
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

func getTokenBalances(cm *chainManager, buf *bytes.Buffer) (result map[string]interface{}) {
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
	cm.withLock(func() {
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

func getTokenBalance(cm *chainManager, buf *bytes.Buffer) (result map[string]interface{}) {
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
	cm.withLock(func() {
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
