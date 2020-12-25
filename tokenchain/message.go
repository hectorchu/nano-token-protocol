package tokenchain

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math/big"
	"strings"
)

type message interface {
	serialize() []byte
	deserialize([]byte)
}

const (
	genesisOp  = 1
	transferOp = 2
)

func newMessageBuffer(op byte) (buf *bytes.Buffer) {
	buf = new(bytes.Buffer)
	buf.WriteString("TKN")
	buf.WriteByte(op)
	return
}

func parseMessage(data []byte) (m message, err error) {
	if string(data[:3]) != "TKN" {
		return nil, errors.New("missing preamble")
	}
	switch data[3] {
	case genesisOp:
		m = new(genesisMessage)
	case transferOp:
		m = new(transferMessage)
	default:
		return nil, errors.New("unrecognized op")
	}
	m.deserialize(data[4:])
	return
}

func writeBigInt(buf *bytes.Buffer, x *big.Int) {
	if buf.Len() != 16 {
		panic("buf not at expected len")
	}
	buf.Write(x.FillBytes(make([]byte, 16)))
}

type genesisMessage struct {
	decimals byte
	name     string
	supply   *big.Int
}

func (m *genesisMessage) serialize() []byte {
	buf := newMessageBuffer(genesisOp)
	buf.WriteByte(m.decimals)
	name := make([]byte, 11)
	copy(name, m.name)
	buf.Write(name)
	writeBigInt(buf, m.supply)
	return buf.Bytes()
}

func (m *genesisMessage) deserialize(data []byte) {
	m.decimals = data[0]
	m.name = strings.TrimRight(string(data[1:12]), "\x00")
	m.supply = new(big.Int).SetBytes(data[12:])
}

type transferMessage struct {
	token  uint32
	amount *big.Int
}

func (m *transferMessage) serialize() []byte {
	buf := newMessageBuffer(transferOp)
	binary.Write(buf, binary.BigEndian, m.token)
	buf.Write(make([]byte, 8))
	writeBigInt(buf, m.amount)
	return buf.Bytes()
}

func (m *transferMessage) deserialize(data []byte) {
	r := bytes.NewReader(data)
	binary.Read(r, binary.BigEndian, &m.token)
	m.amount = new(big.Int).SetBytes(data[12:])
}
