// internal/poller/poller_test.go
package poller

import (
	"errors"
	"testing"
	"time"
)

type fakeClient struct {
	failFC uint8
}

func (f *fakeClient) ReadCoils(addr, qty uint16) ([]bool, error) {
	if f.failFC == 1 {
		return nil, errors.New("fail fc1")
	}
	return make([]bool, qty), nil
}

func (f *fakeClient) ReadDiscreteInputs(addr, qty uint16) ([]bool, error) {
	if f.failFC == 2 {
		return nil, errors.New("fail fc2")
	}
	return make([]bool, qty), nil
}

func (f *fakeClient) ReadHoldingRegisters(addr, qty uint16) ([]uint16, error) {
	if f.failFC == 3 {
		return nil, errors.New("fail fc3")
	}
	return make([]uint16, qty), nil
}

func (f *fakeClient) ReadInputRegisters(addr, qty uint16) ([]uint16, error) {
	if f.failFC == 4 {
		return nil, errors.New("fail fc4")
	}
	return make([]uint16, qty), nil
}

func TestExecuteSingleRead_Success(t *testing.T) {
	cfg := Config{
		UnitID: "u1",
		Reads: []ReadBlock{
			{FC: 1, Address: 0, Quantity: 8, Interval: time.Second},
			{FC: 3, Address: 0, Quantity: 10, Interval: time.Second},
		},
	}

	p, err := New(cfg, &fakeClient{}, nil)
	if err != nil {
		t.Fatalf("New() err=%v", err)
	}

	for _, rb := range cfg.Reads {
		res := p.executeSingleRead(rb)
		if res.Err != nil {
			t.Fatalf("executeSingleRead(%d) err=%v", rb.FC, res.Err)
		}
		if len(res.Blocks) != 1 {
			t.Fatalf("expected 1 block, got %d", len(res.Blocks))
		}
		if res.Blocks[0].FC != rb.FC {
			t.Fatalf("expected FC %d, got %d", rb.FC, res.Blocks[0].FC)
		}
	}
}

func TestExecuteSingleRead_Failure(t *testing.T) {
	cfg := Config{
		UnitID: "u1",
		Reads: []ReadBlock{
			{FC: 3, Address: 0, Quantity: 10, Interval: time.Second},
		},
	}

	p, err := New(cfg, &fakeClient{failFC: 3}, nil)
	if err != nil {
		t.Fatalf("New() err=%v", err)
	}

	res := p.executeSingleRead(cfg.Reads[0])
	if res.Err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestNew_RejectsZeroInterval(t *testing.T) {
	cfg := Config{
		UnitID: "u1",
		Reads: []ReadBlock{
			{FC: 3, Address: 0, Quantity: 10, Interval: 0},
		},
	}

	_, err := New(cfg, &fakeClient{}, nil)
	if err == nil {
		t.Fatalf("expected error for zero interval, got nil")
	}
}

func TestNew_RejectsEmptyUnitID(t *testing.T) {
	cfg := Config{
		Reads: []ReadBlock{
			{FC: 3, Address: 0, Quantity: 10, Interval: time.Second},
		},
	}

	_, err := New(cfg, &fakeClient{}, nil)
	if err == nil {
		t.Fatalf("expected error for empty unit id, got nil")
	}
}

func TestNew_RejectsNoReads(t *testing.T) {
	cfg := Config{
		UnitID: "u1",
	}

	_, err := New(cfg, &fakeClient{}, nil)
	if err == nil {
		t.Fatalf("expected error for empty reads, got nil")
	}
}

func TestCounters_IncrementPerRead(t *testing.T) {
	cfg := Config{
		UnitID: "u1",
		Reads: []ReadBlock{
			{FC: 1, Address: 0, Quantity: 8, Interval: time.Second},
			{FC: 3, Address: 0, Quantity: 10, Interval: time.Second},
		},
	}

	p, err := New(cfg, &fakeClient{}, nil)
	if err != nil {
		t.Fatalf("New() err=%v", err)
	}

	p.executeSingleRead(cfg.Reads[0])
	p.executeSingleRead(cfg.Reads[1])

	c := p.Counters()
	if c.RequestsTotal != 2 {
		t.Fatalf("expected 2 requests, got %d", c.RequestsTotal)
	}
	if c.ResponsesValidTotal != 2 {
		t.Fatalf("expected 2 valid responses, got %d", c.ResponsesValidTotal)
	}
}

