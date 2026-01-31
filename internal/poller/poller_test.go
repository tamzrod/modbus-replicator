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

func TestPollOnce_Success(t *testing.T) {
	cfg := Config{
		UnitID:   "u1",
		Interval: 1 * time.Second,
		Reads: []ReadBlock{
			{FC: 1, Address: 0, Quantity: 8},
			{FC: 3, Address: 0, Quantity: 10},
		},
	}

	p, err := New(cfg, &fakeClient{}, nil)
	if err != nil {
		t.Fatalf("New() err=%v", err)
	}

	res := p.PollOnce()
	if res.Err != nil {
		t.Fatalf("PollOnce err=%v", res.Err)
	}
	if len(res.Blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(res.Blocks))
	}
}

func TestPollOnce_Failure(t *testing.T) {
	cfg := Config{
		UnitID:   "u1",
		Interval: 1 * time.Second,
		Reads: []ReadBlock{
			{FC: 1, Address: 0, Quantity: 8},
			{FC: 3, Address: 0, Quantity: 10},
		},
	}

	p, err := New(cfg, &fakeClient{failFC: 3}, nil)
	if err != nil {
		t.Fatalf("New() err=%v", err)
	}

	res := p.PollOnce()
	if res.Err == nil {
		t.Fatalf("expected error, got nil")
	}
}
