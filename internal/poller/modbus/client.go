// internal/poller/modbus/client.go
package modbus

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/tamzrod/modbus/protocol"
	"github.com/tamzrod/modbus/transport/tcp"
)

// Client implements poller.Client using Modbus TCP.
// This adapter is geometry-only: it builds requests and unpacks raw responses.
type Client struct {
	tr     *tcp.Client
	unitID uint8
	tid    uint16
}

// Config is minimal transport config.
type Config struct {
	Endpoint string
	UnitID   uint8
	Timeout  time.Duration
}

// New creates a connected Modbus TCP client.
func New(cfg Config) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("modbus client: endpoint required")
	}

	conn, err := net.DialTimeout("tcp", cfg.Endpoint, cfg.Timeout)
	if err != nil {
		return nil, err
	}

	c := &Client{
		tr: &tcp.Client{
			Conn:    conn,
			Timeout: cfg.Timeout,
		},
		unitID: cfg.UnitID,
	}

	// Randomize starting TID (best effort).
	var b [2]byte
	if _, err := rand.Read(b[:]); err == nil {
		c.tid = binary.BigEndian.Uint16(b[:])
	}

	return c, nil
}

// Close closes the TCP connection.
func (c *Client) Close() error {
	if c == nil || c.tr == nil || c.tr.Conn == nil {
		return nil
	}
	return c.tr.Conn.Close()
}

// ---- poller.Client interface ----

func (c *Client) ReadCoils(addr, qty uint16) ([]bool, error) {
	pdu, err := c.doReadBits(1, addr, qty)
	if err != nil {
		return nil, err
	}
	return unpackBits(pdu, int(qty)), nil
}

func (c *Client) ReadDiscreteInputs(addr, qty uint16) ([]bool, error) {
	pdu, err := c.doReadBits(2, addr, qty)
	if err != nil {
		return nil, err
	}
	return unpackBits(pdu, int(qty)), nil
}

func (c *Client) ReadHoldingRegisters(addr, qty uint16) ([]uint16, error) {
	pdu, err := c.doReadRegisters(3, addr, qty)
	if err != nil {
		return nil, err
	}
	return unpackRegisters(pdu), nil
}

func (c *Client) ReadInputRegisters(addr, qty uint16) ([]uint16, error) {
	pdu, err := c.doReadRegisters(4, addr, qty)
	if err != nil {
		return nil, err
	}
	return unpackRegisters(pdu), nil
}

// ---- internal request/response helpers ----

func (c *Client) nextTID() uint16 {
	c.tid++
	return c.tid
}

// buildReadRequest builds a Modbus TCP ADU for read functions 1/2/3/4.
//
// MBAP:
//   TID(2) PID(2=0) LEN(2) UID(1)
// PDU:
//   FC(1) Address(2) Quantity(2)
func (c *Client) buildReadRequest(fc uint8, addr, qty uint16) ([]byte, uint16) {
	tid := c.nextTID()

	// Length = UnitID(1) + PDU(1+2+2) = 6
	const protoID uint16 = 0
	const length uint16 = 6

	adu := make([]byte, 7+5) // MBAP(7) + PDU(5)
	binary.BigEndian.PutUint16(adu[0:2], tid)
	binary.BigEndian.PutUint16(adu[2:4], protoID)
	binary.BigEndian.PutUint16(adu[4:6], length)
	adu[6] = c.unitID

	adu[7] = fc
	binary.BigEndian.PutUint16(adu[8:10], addr)
	binary.BigEndian.PutUint16(adu[10:12], qty)

	return adu, tid
}

func (c *Client) roundTripRead(fc uint8, addr, qty uint16) ([]byte, error) {
	if c == nil || c.tr == nil || c.tr.Conn == nil {
		return nil, errors.New("modbus client: not connected")
	}

	req, tid := c.buildReadRequest(fc, addr, qty)

	raw, err := c.tr.Send(req)
	if err != nil {
		return nil, err
	}

	resp, err := protocol.DecodeTCP(raw)
	if err != nil {
		return nil, err
	}

	// Best-effort sanity checks (still geometry-only).
	if resp.TransactionID != tid {
		return nil, fmt.Errorf("modbus tcp: transaction id mismatch: got=%d want=%d", resp.TransactionID, tid)
	}
	if resp.ProtocolID != 0 {
		return nil, fmt.Errorf("modbus tcp: protocol id mismatch: got=%d want=0", resp.ProtocolID)
	}
	if resp.UnitID != c.unitID {
		return nil, fmt.Errorf("modbus tcp: unit id mismatch: got=%d want=%d", resp.UnitID, c.unitID)
	}
	if resp.Exception != nil {
		return nil, fmt.Errorf("modbus exception: fc=%d code=%d", resp.Function, uint8(*resp.Exception))
	}
	if resp.Function != fc {
		return nil, fmt.Errorf("modbus: function mismatch: got=%d want=%d", resp.Function, fc)
	}

	return resp.Payload, nil
}

func (c *Client) doReadBits(fc uint8, addr, qty uint16) ([]byte, error) {
	if qty == 0 {
		return nil, nil
	}
	p, err := c.roundTripRead(fc, addr, qty)
	if err != nil {
		return nil, err
	}
	if len(p) < 1 {
		return nil, errors.New("modbus: short read-bits payload")
	}
	// payload[0] = byte count, remaining = packed bits
	byteCount := int(p[0])
	if len(p)-1 < byteCount {
		return nil, errors.New("modbus: read-bits payload shorter than byte count")
	}
	return p[1 : 1+byteCount], nil
}

func (c *Client) doReadRegisters(fc uint8, addr, qty uint16) ([]byte, error) {
	if qty == 0 {
		return nil, nil
	}
	p, err := c.roundTripRead(fc, addr, qty)
	if err != nil {
		return nil, err
	}
	if len(p) < 1 {
		return nil, errors.New("modbus: short read-registers payload")
	}
	// payload[0] = byte count, remaining = registers big-endian
	byteCount := int(p[0])
	if byteCount%2 != 0 {
		return nil, errors.New("modbus: read-registers byte count not even")
	}
	if len(p)-1 < byteCount {
		return nil, errors.New("modbus: read-registers payload shorter than byte count")
	}
	return p[1 : 1+byteCount], nil
}

// ---- helpers (pure geometry) ----

func unpackBits(data []byte, count int) []bool {
	out := make([]bool, count)
	for i := 0; i < count; i++ {
		byteIdx := i / 8
		bitIdx := i % 8
		if byteIdx >= len(data) {
			out[i] = false
			continue
		}
		out[i] = (data[byteIdx]&(1<<bitIdx) != 0)
	}
	return out
}

func unpackRegisters(data []byte) []uint16 {
	n := len(data) / 2
	out := make([]uint16, n)
	for i := 0; i < n; i++ {
		out[i] = uint16(data[2*i])<<8 | uint16(data[2*i+1])
	}
	return out
}
