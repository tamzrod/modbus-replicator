// internal/writer/writer.go
package writer

import (
	"errors"
	"fmt"
	"strings"

	"modbus-replicator/internal/poller"
)

// endpointClient is the exact contract the writer uses.
// IMPORTANT: There must be NO other version of this interface anywhere.
type endpointClient interface {
	WriteBits(area byte, unitID uint8, addr uint16, bits []bool) error
	WriteRegisters(area byte, unitID uint8, addr uint16, regs []uint16) error
}

type writerImpl struct {
	plan    Plan
	clients map[string]endpointClient
}

func New(plan Plan, clients map[string]endpointClient) Writer {
	return &writerImpl{
		plan:    plan,
		clients: clients,
	}
}

func (w *writerImpl) Write(res poller.PollResult) error {
	var errs []string

	// ------------------------------------------------------------
	// DATA WRITES ONLY
	// ------------------------------------------------------------

	if res.Err == nil {
		for _, tgt := range w.plan.Targets {
			cli := w.clients[tgt.Endpoint]
			if cli == nil {
				errs = append(errs, fmt.Sprintf(
					"writer: missing client for endpoint %s",
					tgt.Endpoint,
				))
				continue
			}

			if tgt.TargetID > 255 {
				errs = append(errs, fmt.Sprintf(
					"writer: target unit id %d out of range",
					tgt.TargetID,
				))
				continue
			}
			unitID := uint8(tgt.TargetID)

			for _, mem := range tgt.Memories {
				for _, b := range res.Blocks {

					area := byte(b.FC)
					dstAddr := offsetForFC(mem.Offsets, b.FC) + b.Address

					switch b.FC {
					case 1, 2:
						if err := cli.WriteBits(area, unitID, dstAddr, b.Bits); err != nil {
							errs = append(errs, fmt.Sprintf(
								"writer: ep=%s unit=%d fc=%d addr=%d err=%v",
								tgt.Endpoint, unitID, b.FC, dstAddr, err,
							))
						}
					case 3, 4:
						if err := cli.WriteRegisters(area, unitID, dstAddr, b.Registers); err != nil {
							errs = append(errs, fmt.Sprintf(
								"writer: ep=%s unit=%d fc=%d addr=%d err=%v",
								tgt.Endpoint, unitID, b.FC, dstAddr, err,
							))
						}
					default:
						errs = append(errs, fmt.Sprintf(
							"writer: unsupported fc %d",
							b.FC,
						))
					}
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
