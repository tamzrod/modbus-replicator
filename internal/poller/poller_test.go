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

// TestAddInvert_FC1 verifies that AddInvert=true appends an inverted copy block
// immediately after the original block for FC1.
func TestAddInvert_FC1(t *testing.T) {
	client := &fakeClient{bitsFC1: []bool{true, false, true, false}}

	cfg := Config{
		UnitID:   "u1",
		Interval: 1 * time.Second,
		Reads: []ReadBlock{
			{FC: 1, Address: 0, Quantity: 4, AddInvert: true},
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

	if len(res.Blocks) != 2 {
		t.Fatalf("expected 2 blocks (original + inverted), got %d", len(res.Blocks))
	}

	orig := res.Blocks[0]
	inv := res.Blocks[1]

	// Original block at address 0
	if orig.Address != 0 {
		t.Errorf("original block address: got %d, want 0", orig.Address)
	}
	wantOrig := []bool{true, false, true, false}
	for i, v := range wantOrig {
		if orig.Bits[i] != v {
			t.Errorf("orig.Bits[%d]: got %v, want %v", i, orig.Bits[i], v)
		}
	}

	// Inverted block at address 4 (= 0 + 4)
	if inv.Address != 4 {
		t.Errorf("inverted block address: got %d, want 4", inv.Address)
	}
	wantInv := []bool{false, true, false, true}
	for i, v := range wantInv {
		if inv.Bits[i] != v {
			t.Errorf("inv.Bits[%d]: got %v, want %v", i, inv.Bits[i], v)
		}
	}
}

// TestAddInvert_FC2 verifies that AddInvert=true appends an inverted copy block
// immediately after the original block for FC2.
func TestAddInvert_FC2(t *testing.T) {
	client := &fakeClient{bitsFC2: []bool{false, true, false}}

	cfg := Config{
		UnitID:   "u1",
		Interval: 1 * time.Second,
		Reads: []ReadBlock{
			{FC: 2, Address: 10, Quantity: 3, AddInvert: true},
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

	if len(res.Blocks) != 2 {
		t.Fatalf("expected 2 blocks (original + inverted), got %d", len(res.Blocks))
	}

	orig := res.Blocks[0]
	inv := res.Blocks[1]

	if orig.Address != 10 {
		t.Errorf("original block address: got %d, want 10", orig.Address)
	}
	if inv.Address != 13 { // 10 + 3
		t.Errorf("inverted block address: got %d, want 13", inv.Address)
	}

	wantInv := []bool{true, false, true}
	for i, v := range wantInv {
		if inv.Bits[i] != v {
			t.Errorf("inv.Bits[%d]: got %v, want %v", i, inv.Bits[i], v)
		}
	}
}

// TestAddInvert_WithInvert verifies that AddInvert appends the inversion of the
// already-transformed bits when Invert is also true.
func TestAddInvert_WithInvert(t *testing.T) {
	// Device returns [true, false, true]
	client := &fakeClient{bitsFC1: []bool{true, false, true}}

	cfg := Config{
		UnitID:   "u1",
		Interval: 1 * time.Second,
		Reads: []ReadBlock{
			{FC: 1, Address: 0, Quantity: 3, Invert: true, AddInvert: true},
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

	if len(res.Blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(res.Blocks))
	}

	orig := res.Blocks[0]
	inv := res.Blocks[1]

	// Invert=true flips raw bits: [true,false,true] -> [false,true,false]
	wantOrig := []bool{false, true, false}
	for i, v := range wantOrig {
		if orig.Bits[i] != v {
			t.Errorf("orig.Bits[%d]: got %v, want %v", i, orig.Bits[i], v)
		}
	}

	// AddInvert copies are the inversion of the already-inverted bits: [true,false,true]
	wantInv := []bool{true, false, true}
	for i, v := range wantInv {
		if inv.Bits[i] != v {
			t.Errorf("inv.Bits[%d]: got %v, want %v", i, inv.Bits[i], v)
		}
	}
}

// TestAddInvert_FC3_Ignored verifies that AddInvert has no effect on FC3.
func TestAddInvert_FC3_Ignored(t *testing.T) {
	cfg := Config{
		UnitID:   "u1",
		Interval: 1 * time.Second,
		Reads: []ReadBlock{
			// AddInvert must never be set for FC3 by the builder.
			// Even if set directly, FC3 branch must not emit an extra block.
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
	if len(res.Blocks) != 1 {
		t.Fatalf("expected exactly 1 block for FC3, got %d", len(res.Blocks))
	}
}
