package telemetry

import (
	"fmt"
	"io"
	"io/ioutil"
)

// Represents one second of telemetry data
type TELEM struct {
	deviceName  string
	Gps         []GPS5
	GpsFix      GPSF
	GpsAccuracy GPSP
	Time        GPSU
}

func Read(t *TELEM, f io.Reader) (*TELEM, error) {

	fourCC := make([]byte, 4) // 4 byte ascii label of data

	// https://github.com/gopro/gpmf-parser#length-type-size-repeat-structure
	desc := make([]byte, 4) // 4 byte description of length of data

	// keep a copy of the scale to apply to subsequent sentences
	s := SCAL{}

	for {
		// pick out the label
		read, err := f.Read(fourCC)
		if err == io.EOF || read == 0 {
			break
		}

		label_string := string(fourCC)

		// pick out the label description
		read, err = f.Read(desc)
		if err == io.EOF || read == 0 {
			break
		}

		// extract the size and length (https://github.com/gopro/gpmf-parser/blob/main/docs/readmegfx/KLVDesign.png)
		// desc[0] contains a 1-byte type. We don't use this information.
		structSize := uint8(desc[1])
		numStructs := (uint16(desc[2]) << 8) | uint16(desc[3])

		// uncomment to see label, type, size and length
		//fmt.Printf("%s:  %d samples of len %d, type %c\n", fourCC, numStructs, structSize, desc[0)

		if label_string == "SCAL" {
			value := make([]byte, numStructs*uint16(structSize))
			read, err = f.Read(value)
			if err == io.EOF || read == 0 {
				return nil, err
			}

			// clear the scales
			s.Values = s.Values[:0]

			err := s.Parse(value, structSize)
			if err != nil {
				return nil, err
			}
		} else if label_string == "DEVC" {
			fmt.Println("Found DEVC container")
			if _, err := Read(t, f); err != nil {
				return nil, err
			}
		} else if label_string == "STRM" {
			/* New stream container, read the nested data */
			if _, err := Read(t, f); err != nil {
				return nil, err
			}
		} else {
			value := make([]byte, structSize)
			allValues := make([][]byte, numStructs)

			for i := uint16(0); i < numStructs; i++ {
				read, err := f.Read(value)
				if err == io.EOF || read == 0 {
					return nil, err
				}
				allValues[i] = make([]byte, structSize)
				copy(allValues[i], value)
			}
			switch label_string {
			case "STNM":
				var streamName []byte
				for i := 0; i < len(allValues)-1; i++ {
					streamName = append(streamName, allValues[i][0])
				}
				//fmt.Printf("Stream name: %q\n", string(st))
			case "DVNM":
				var st []byte
				for i := 0; i < len(allValues); i++ {
					st = append(st, allValues[i][0])
				}
				desc := string(st)
				t.deviceName = desc

			case "GPS5":
				for i := 0; i < len(allValues); i++ {
					g := GPS5{}
					g.Parse(allValues[i], &s)
					t.Gps = append(t.Gps, g)
				}
			case "GPSU":
				g := GPSU{}
				err := g.Parse(value)
				if err != nil {
					return nil, err
				}
				t.Time = g
			case "GPSP":
				g := GPSP{}
				err := g.Parse(value)
				if err != nil {
					return nil, err
				}
				t.GpsAccuracy = g
			case "GPSF":
				g := GPSF{}
				err := g.Parse(value)
				if err != nil {
					return nil, err
				}
				t.GpsFix = g
			case "TSMP":
				tsmp := TSMP{}
				tsmp.Parse(value, &s)
			default:
				//fmt.Printf("Unknown verb %q\n", label_string)
			}
		}

		// pack into 4 bytes
		mod := (numStructs * uint16(structSize)) % 4
		if mod != 0 {
			seek := 4 - mod
			io.CopyN(ioutil.Discard, f, int64(seek))
		}
	}

	return t, nil
}
