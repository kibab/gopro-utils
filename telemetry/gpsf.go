package telemetry

import (
	"encoding/binary"
	"errors"
)

// GPS Fix. 0 - no lock, 2 or 3 - 2D or 3D Lock
type GPSF struct {
	F uint32
}

func (gpsf *GPSF) Parse(bytes []byte) error {
	if len(bytes) != 4 {
		return errors.New("invalid length GPSF packet")
	}

	gpsf.F = binary.BigEndian.Uint32(bytes[0:4])
	return nil
}
