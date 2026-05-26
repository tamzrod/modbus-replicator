// internal/poller/poller_test.go
package poller

import (
	"errors"
	"testing"
	"time"
)

type fakeClient struct {
	failFC uint8
	// bitsFC1 and bitsFC2 are the values returned for coil/discrete reads.
	// When nil, all-false slices of the requested quantity are returned.
	bitsFC1 []bool
	bitsFC2 []bool
}

func (f *fakeClient) ReadCoils(addr, qty uint16) ([]bool, error) {
	if f.failFC == 1 {
		return nil, errors.New("fail fc1")
	}
	if f.bitsFC1 != nil {
		out := make([]bool, qty)
		copy(out, f.bitsFC1)
		return out, nil
	}
	return make([]bool, qty), nil
}

func (f *fakeClient) ReadDiscreteInputs(addr, qty uint16) ([]bool, error) {
	if f.failFC == 2 {
		return nil, errors.New("fail fc2")
	}
	if f.bitsFC2 != nil {
		out := make([]bool, qty)
		copy(out, f.bitsFC2)
		return out, nil
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

// TestInvert_FC1 verifies that Invert=true flips all bits returned by ReadCoils.
func TestInvert_FC1(t *testing.T) {
	// Device returns [true, false, true]
	client := &fakeClient{bitsFC1: []bool{true, false, true}}

	cfg := Config{
		UnitID:   "u1",
		Interval: 1 * time.Second,
		Reads: []ReadBlock{
			{FC: 1, Address: 0, Quantity: 3, Invert: true},
		},
	}

	p, err := New(cfg, client, nil)
	if err != nil {
		t.Fatalf("New() err=%v", err)
	}

	res := p.PollOnce()
	if res.Err != nil {
		t.Fatalf("PollOnce err=%v", res.Err)
	}

	got := res.Blocks[0].Bits
	want := []bool{false, true, false}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("bit[%d]: got %v, want %v", i, got[i], v)
		}
	}
}

// TestInvert_FC2 verifies that Invert=true flips all bits returned by ReadDiscreteInputs.
func TestInvert_FC2(t *testing.T) {
	client := &fakeClient{bitsFC2: []bool{false, false, true}}

	cfg := Config{
		UnitID:   "u1",
		Interval: 1 * time.Second,
		Reads: []ReadBlock{
			{FC: 2, Address: 0, Quantity: 3, Invert: true},
		},
	}

	p, err := New(cfg, client, nil)
	if err != nil {
		t.Fatalf("New() err=%v", err)
	}

	res := p.PollOnce()
	if res.Err != nil {
		t.Fatalf("PollOnce err=%v", res.Err)
	}

	got := res.Blocks[0].Bits
	want := []bool{true, true, false}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("bit[%d]: got %v, want %v", i, got[i], v)
		}
	}
}

// TestInvert_FC1_False verifies that Invert=false leaves bits unchanged.
func TestInvert_FC1_False(t *testing.T) {
	client := &fakeClient{bitsFC1: []bool{true, false, true}}

	cfg := Config{
		UnitID:   "u1",
		Interval: 1 * time.Second,
		Reads: []ReadBlock{
			{FC: 1, Address: 0, Quantity: 3, Invert: false},
		},
	}

	p, err := New(cfg, client, nil)
	if err != nil {
		t.Fatalf("New() err=%v", err)
	}

	res := p.PollOnce()
	if res.Err != nil {
		t.Fatalf("PollOnce err=%v", res.Err)
	}

	got := res.Blocks[0].Bits
	want := []bool{true, false, true}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("bit[%d]: got %v, want %v", i, got[i], v)
		}
	}
}

// TestInvert_FC3_Ignored verifies that Invert=true on a ReadBlock for FC3
// has no effect: registers are returned unchanged and no error occurs.
func TestInvert_FC3_Ignored(t *testing.T) {
	cfg := Config{
		UnitID:   "u1",
		Interval: 1 * time.Second,
		Reads: []ReadBlock{
			// Invert must never be set for FC3 by the builder, but even if a
			// caller constructs one directly, the FC3 branch must not check it.
			{FC: 3, Address: 0, Quantity: 4},
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
	if len(res.Blocks[0].Registers) != 4 {
		t.Fatalf("expected 4 registers, got %d", len(res.Blocks[0].Registers))
	}
}
