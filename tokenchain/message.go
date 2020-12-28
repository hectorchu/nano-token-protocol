package tokenchain

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math/big"
	"strings"

	"github.com/hectorchu/gonano/rpc"
)

type message interface {
	serialize() []byte
	deserialize([]byte)
	process(*Chain, rpc.BlockHash, uint32, rpc.BlockInfo) (bool, error)
}

const (
	genesisOp     = 1
	transferOp    = 2
	swapProposeOp = 3
	swapAcceptOp  = 4
	swapConfirmOp = 5
	swapCancelOp  = 6
)

func newMessageBuffer(op byte) (buf *bytes.Buffer) {
	buf = new(bytes.Buffer)
	buf.WriteString("TKN")
	buf.WriteByte(op)
	return
}

func parseMessage(data []byte) (m message, err error) {
	if string(data[:3]) != "TKN" {
		return nil, errors.New("Missing preamble")
	}
	switch data[3] {
	case genesisOp:
		m = new(genesisMessage)
	case transferOp:
		m = new(transferMessage)
	case swapProposeOp:
		m = new(swapProposeMessage)
	case swapAcceptOp:
		m = new(swapAcceptMessage)
	case swapConfirmOp:
		m = new(swapConfirmMessage)
	case swapCancelOp:
		m = new(swapCancelMessage)
	default:
		return nil, errors.New("Unrecognized op")
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
	name := make([]byte, 16-buf.Len())
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
	buf.Write(make([]byte, 16-buf.Len()))
	writeBigInt(buf, m.amount)
	return buf.Bytes()
}

func (m *transferMessage) deserialize(data []byte) {
	r := bytes.NewReader(data)
	binary.Read(r, binary.BigEndian, &m.token)
	m.amount = new(big.Int).SetBytes(data[12:])
}

type swapProposeMessage struct {
	token  uint32
	amount *big.Int
}

func (m *swapProposeMessage) serialize() []byte {
	buf := newMessageBuffer(swapProposeOp)
	binary.Write(buf, binary.BigEndian, m.token)
	buf.Write(make([]byte, 16-buf.Len()))
	writeBigInt(buf, m.amount)
	return buf.Bytes()
}

func (m *swapProposeMessage) deserialize(data []byte) {
	r := bytes.NewReader(data)
	binary.Read(r, binary.BigEndian, &m.token)
	m.amount = new(big.Int).SetBytes(data[12:])
}

type swapAcceptMessage struct {
	swap   uint32
	token  uint32
	amount *big.Int
}

func (m *swapAcceptMessage) serialize() []byte {
	buf := newMessageBuffer(swapAcceptOp)
	binary.Write(buf, binary.BigEndian, m.swap)
	binary.Write(buf, binary.BigEndian, m.token)
	buf.Write(make([]byte, 16-buf.Len()))
	writeBigInt(buf, m.amount)
	return buf.Bytes()
}

func (m *swapAcceptMessage) deserialize(data []byte) {
	r := bytes.NewReader(data)
	binary.Read(r, binary.BigEndian, &m.swap)
	binary.Read(r, binary.BigEndian, &m.token)
	m.amount = new(big.Int).SetBytes(data[12:])
}

type swapConfirmMessage struct {
	swap uint32
}

func (m *swapConfirmMessage) serialize() []byte {
	buf := newMessageBuffer(swapConfirmOp)
	binary.Write(buf, binary.BigEndian, m.swap)
	buf.Write(make([]byte, 32-buf.Len()))
	return buf.Bytes()
}

func (m *swapConfirmMessage) deserialize(data []byte) {
	r := bytes.NewReader(data)
	binary.Read(r, binary.BigEndian, &m.swap)
}

type swapCancelMessage struct {
	swap uint32
}

func (m *swapCancelMessage) serialize() []byte {
	buf := newMessageBuffer(swapCancelOp)
	binary.Write(buf, binary.BigEndian, m.swap)
	buf.Write(make([]byte, 32-buf.Len()))
	return buf.Bytes()
}

func (m *swapCancelMessage) deserialize(data []byte) {
	r := bytes.NewReader(data)
	binary.Read(r, binary.BigEndian, &m.swap)
}
