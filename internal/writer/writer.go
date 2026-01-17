// internal/writer/writer.go
package writer

import (
	"errors"
	"fmt"
	"strings"

	"modbus-replicator/internal/poller"
)

// endpointClient is the minimal write surface the writer needs.
// Transport-specific implementations live elsewhere.
type endpointClient interface {
	WriteCoils(unitID uint8, addr uint16, bits []bool) error
	WriteRegisters(unitID uint8, addr uint16, regs []uint16) error
}

type modbusWriter struct {
	plan    Plan
	clients map[string]endpointClient // endpoint -> client
}

func NewModbusWriter(plan Plan, clients map[string]endpointClient) Writer {
	return &modbusWriter{plan: plan, clients: clients}
}

func (w *modbusWriter) Write(res poller.PollResult) error {
	if res.Err != nil {
		return res.Err
	}

	var errs []string

	for _, tgt := range w.plan.Targets {
		cli := w.clients[tgt.Endpoint]
		if cli == nil {
			errs = append(errs, fmt.Sprintf("writer: missing client for endpoint %s", tgt.Endpoint))
			continue
		}

		for _, mem := range tgt.Memories {
			unitID, ok := asUnitID(mem.MemoryID)
			if !ok {
				errs = append(errs, fmt.Sprintf("writer: memory_id %d out of range", mem.MemoryID))
				continue
			}

			for _, b := range res.Blocks {
				off := offsetForFC(mem.Offsets, b.FC)
				dstAddr := off + b.Address

				switch b.FC {
				case 1, 2:
					if err := cli.WriteCoils(unitID, dstAddr, b.Bits); err != nil {
						errs = append(errs, fmt.Sprintf(
							"writer: ep=%s mem=%d fc=%d addr=%d err=%v",
							tgt.Endpoint, mem.MemoryID, b.FC, dstAddr, err,
						))
					}

				case 3, 4:
					if err := cli.WriteRegisters(unitID, dstAddr, b.Registers); err != nil {
						errs = append(errs, fmt.Sprintf(
							"writer: ep=%s mem=%d fc=%d addr=%d err=%v",
							tgt.Endpoint, mem.MemoryID, b.FC, dstAddr, err,
						))
					}

				default:
					errs = append(errs, fmt.Sprintf("writer: unsupported fc %d", b.FC))
				}
			}
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, " | "))
	}

	return nil
}

func offsetForFC(offsets map[int]uint16, fc uint8) uint16 {
	if offsets == nil {
		return 0
	}
	if v, ok := offsets[int(fc)]; ok {
		return v
	}
	return 0
}

func asUnitID(memoryID uint16) (uint8, bool) {
	if memoryID > 255 {
		return 0, false
	}
	return uint8(memoryID), true
}
