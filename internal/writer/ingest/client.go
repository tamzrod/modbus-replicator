// internal/writer/ingest/client.go
package ingest

import (
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	magicHi byte = 0x52 // 'R'
	magicLo byte = 0x49 // 'I'

	versionV1 byte = 0x01

	respOK       byte = 0x00
	respRejected byte = 0x01
)

// Raw Ingest v1 client (stateless, 1 packet = 1 connection)
type EndpointClient struct {
	endpoint string
	timeout  time.Duration
}

type Config struct {
	Endpoint string
	Timeout  time.Duration
}

func NewEndpointClient(cfg Config) (*EndpointClient, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("writer ingest: endpoint required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 2 * time.Second
	}
	return &EndpointClient{
		endpoint: cfg.Endpoint,
		timeout:  cfg.Timeout,
	}, nil
}

func (c *EndpointClient) Close() error { return nil }

//
// Implements writer.endpointClient
//

// FC1 / FC2
func (c *EndpointClient) WriteBits(
	area byte,
	unitID uint8,
	addr uint16,
	bits []bool,
) error {
	payload := packBits(bits)
	return c.send(area, unitID, addr, uint16(len(bits)), payload)
}

// FC3 / FC4
func (c *EndpointClient) WriteRegisters(
	area byte,
	unitID uint8,
	addr uint16,
	regs []uint16,
) error {
	payload := packRegisters(regs)
	return c.send(area, unitID, addr, uint16(len(regs)), payload)
}

//
// ---- Raw Ingest v1 sender ----
// Header is EXACTLY 10 bytes (matches Node-RED)
//

func (c *EndpointClient) send(
	area byte,
	unitID uint8,
	addr uint16,
	count uint16,
	payload []byte,
) error {

	pkt := buildPacketV1(area, unitID, addr, count, payload)

	conn, err := net.DialTimeout("tcp", c.endpoint, c.timeout)
	if err != nil {
		return fmt.Errorf("writer ingest: dial: %w", err)
	}
	defer conn.Close()

	_ = conn.SetWriteDeadline(time.Now().Add(c.timeout))
	if err := writeAll(conn, pkt); err != nil {
		return fmt.Errorf("writer ingest: write: %w", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(c.timeout))
	var resp [1]byte
	if _, err := io.ReadFull(conn, resp[:]); err != nil {
		return fmt.Errorf("writer ingest: read status: %w", err)
	}

	switch resp[0] {
	case respOK:
		return nil
	case respRejected:
		return errors.New("writer ingest: rejected")
	default:
		return fmt.Errorf("writer ingest: unknown status 0x%02x", resp[0])
	}
}

//
// ---- Raw Ingest v1 packet builder (LOCKED) ----
//
// Layout (10 bytes header):
// 0–1  Magic "RI"
// 2    Version (0x01)
// 3    Area
// 4–5  UnitID
// 6–7  Address
// 8–9  Count
// 10+  Payload
//

func buildPacketV1(
	area byte,
	unitID uint8,
	addr uint16,
	count uint16,
	payload []byte,
) []byte {

	header := make([]byte, 10)

	header[0] = magicHi
	header[1] = magicLo
	header[2] = versionV1
	header[3] = area

	putU16(header[4:6], uint16(unitID))
	putU16(header[6:8], addr)
	putU16(header[8:10], count)

	return append(header, payload...)
}

//
// ---- helpers ----
//

func writeAll(w io.Writer, b []byte) error {
	for len(b) > 0 {
		n, err := w.Write(b)
		if err != nil {
			return err
		}
		b = b[n:]
	}
	return nil
}

func putU16(dst []byte, v uint16) {
	dst[0] = byte(v >> 8)
	dst[1] = byte(v)
}

func packBits(bits []bool) []byte {
	n := (len(bits) + 7) / 8
	out := make([]byte, n)
	for i, v := range bits {
		if v {
			out[i/8] |= 1 << uint(i%8)
		}
	}
	return out
}

// Modbus register memory order (BIG-ENDIAN)
func packRegisters(regs []uint16) []byte {
	out := make([]byte, len(regs)*2)
	for i, r := range regs {
		out[2*i] = byte(r >> 8)
		out[2*i+1] = byte(r)
	}
	return out
}
