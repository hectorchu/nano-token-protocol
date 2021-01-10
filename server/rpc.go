package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
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
	switch v.Action {
	}
	buf.Reset()
	json.NewEncoder(&buf).Encode(map[string]string{})
	io.Copy(w, &buf)
}
