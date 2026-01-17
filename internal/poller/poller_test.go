package poller

import (
	"errors"
	"testing"
	"time"
)

// ---- fake client ----

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

// ---- tests ----

func TestPollOnce_AllFCsSuccess(t *testing.T) {
	p, err := New(Config{
		UnitID:   "unit-1",
		Interval: time.Second,
		Reads: []ReadBlock{
			{FC: 1, Address: 0, Quantity: 8},
			{FC: 2, Address: 0, Quantity: 8},
			{FC: 3, Address: 0, Quantity: 10},
			{FC: 4, Address: 0, Quantity: 10},
		},
	}, &fakeClient{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	res := p.PollOnce()
	if res.Err != nil {
		t.Fatalf("unexpected poll error: %v", res.Err)
	}

	if len(res.Blocks) != 4 {
		t.Fatalf("expected 4 blocks, got %d", len(res.Blocks))
	}

	for _, b := range res.Blocks {
		switch b.FC {
		case 1, 2:
			if len(b.Bits) != int(b.Quantity) {
				t.Fatalf("fc %d: expected %d bits, got %d", b.FC, b.Quantity, len(b.Bits))
			}
		case 3, 4:
			if len(b.Registers) != int(b.Quantity) {
				t.Fatalf("fc %d: expected %d regs, got %d", b.FC, b.Quantity, len(b.Registers))
			}
		}
	}
}

func TestPollOnce_UnsupportedFC(t *testing.T) {
	p, _ := New(Config{
		UnitID:   "unit-1",
		Interval: time.Second,
		Reads: []ReadBlock{
			{FC: 9, Address: 0, Quantity: 1},
		},
	}, &fakeClient{})

	res := p.PollOnce()
	if res.Err == nil {
		t.Fatalf("expected error for unsupported FC")
	}
	if len(res.Blocks) != 0 {
		t.Fatalf("expected no blocks on failure")
	}
}

func TestPollOnce_AllOrNothing(t *testing.T) {
	p, _ := New(Config{
		UnitID:   "unit-1",
		Interval: time.Second,
		Reads: []ReadBlock{
			{FC: 3, Address: 0, Quantity: 10},
			{FC: 4, Address: 0, Quantity: 10},
		},
	}, &fakeClient{failFC: 4})

	res := p.PollOnce()
	if res.Err == nil {
		t.Fatalf("expected poll error")
	}

	if len(res.Blocks) != 0 {
		t.Fatalf("expected zero blocks on partial failure, got %d", len(res.Blocks))
	}
}
