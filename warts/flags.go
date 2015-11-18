package warts

import (
	"fmt"
	"io"
)

func getFlags(f io.Reader, first uint8) ([]uint8, error) {
	var ret []uint8
	check := first
	byteNum := uint8(1)
	for {
		for i := uint8(1); i < 8; i++ {
			if isset(check, i) {
				ret = append(ret, i+(7*(byteNum-1)))
			}
		}
		if !isset(check, 8) {
			return ret, nil
		}
		next, err := readOne(f)
		if err != nil {
			return nil, err
		}
		check = next
		byteNum++

	}
}

func readListFlags(f io.Reader) (ListFlags, error) {
	first := make([]byte, 1)
	var lf ListFlags
	var dscIsSet bool
	var monIsSet bool
	n, err := f.Read(first)
	if err != nil {
		return lf, fmt.Errorf("Failed to read list flag: %v", err)
	}
	if n != 1 {
		return lf, fmt.Errorf("Bad Read readListFlags")
	}
	flag := getUint8(first)
	if isset(flag, 1) {
		dscIsSet = true
	}
	if isset(flag, 2) {
		monIsSet = true
	}
	if !dscIsSet && !monIsSet {
		return lf, nil
	}
	l, err := readUint16(f)
	if err != nil {
		return lf, err
	}
	lf.Length = l
	if dscIsSet {
		desc, err := getString(f)
		if err != nil {
			return lf, nil
		}
		lf.Description = desc
	}
	if monIsSet {
		mon, err := getString(f)
		if err != nil {
			return lf, nil
		}
		lf.MonitorName = mon
	}
	return lf, nil
}
