package mmdbwriter

import (
	"encoding/binary"
	"io"
	"math/big"
	"math/bits"
	"sort"

	"github.com/pkg/errors"
)

type typeNum byte

const (
	typeNumExtended typeNum = iota
	typeNumPointer          // nolint: deadcode, varcheck
	typeNumString
	typeNumFloat64
	typeNumBytes
	typeNumUint16
	typeNumUint32
	typeNumMap
	typeNumInt32
	typeNumUint64
	typeNumUint128
	typeNumSlice
	// We don't use the next two. They are placeholders. See the spec
	// for more details.
	typeNumContainer // nolint: deadcode, varcheck
	typeNumMarker    // nolint: deadcode, varcheck
	typeNumBool
	typeNumFloat32
)

type writer interface {
	io.Writer
	WriteByte(byte) error
	WriteString(string) (int, error)
}

type dataType interface {
	size() int
	typeNum() typeNum
	writeTo(writer) error
}

type typeBool bool

func (t typeBool) size() int {
	if t {
		return 1
	}
	return 0
}

func (t typeBool) typeNum() typeNum {
	return typeNumBool
}

func (t typeBool) writeTo(w writer) error {
	return writeCtrlByte(w, t)
}

type typeBytes []byte

func (t typeBytes) size() int {
	return len(t)
}

func (t typeBytes) typeNum() typeNum {
	return typeNumBytes
}

func (t typeBytes) writeTo(w writer) error {
	err := writeCtrlByte(w, t)
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(t))
	if err != nil {
		return errors.Wrapf(err, `error writing "%s" as a string`, t)
	}
	return nil
}

type typeFloat32 float32

func (t typeFloat32) size() int {
	return 4
}

func (t typeFloat32) typeNum() typeNum {
	return typeNumFloat32
}

func (t typeFloat32) writeTo(w writer) error {
	err := writeCtrlByte(w, t)
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.BigEndian, t)
	if err != nil {
		return errors.Wrapf(err, "error writing %f as float32", t)
	}
	return nil
}

type typeFloat64 float64

func (t typeFloat64) size() int {
	return 8
}

func (t typeFloat64) typeNum() typeNum {
	return typeNumFloat64
}

func (t typeFloat64) writeTo(w writer) error {
	err := writeCtrlByte(w, t)
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.BigEndian, t)
	if err != nil {
		return errors.Wrapf(err, "error writing %f as float64", t)
	}
	return nil
}

type typeInt32 int32

func (t typeInt32) size() int {
	return 4 - bits.LeadingZeros32(uint32(t))/8
}

func (t typeInt32) typeNum() typeNum {
	return typeNumInt32
}

func (t typeInt32) writeTo(w writer) error {
	err := writeCtrlByte(w, t)
	if err != nil {
		return err
	}

	// We ignore leading zeros
	for i := t.size(); i > 0; i-- {
		err = w.WriteByte(byte((int32(t) >> (8 * (i - 1))) & 0xFF))
		if err != nil {
			return errors.Wrap(err, "error writing int32")
		}
	}
	return nil
}

type typeMap map[typeString]dataType

func (t typeMap) size() int {
	return len(t)
}

func (t typeMap) typeNum() typeNum {
	return typeNumMap
}

func (t typeMap) writeTo(w writer) error {
	err := writeCtrlByte(w, t)
	if err != nil {
		return err
	}

	// We want database builds to be reproducible. As such, we insert
	// the map items in order by key value. In the future, we will
	// likely use a more relevant characteristic here (e.g., putting
	// fields more likely to be accessed first).
	keys := make([]string, 0, len(t))
	for k := range t {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)

	for _, ks := range keys {
		k := typeString(ks)
		err := k.writeTo(w)
		if err != nil {
			return err
		}
		err = t[k].writeTo(w)
		if err != nil {
			return err
		}
	}
	return nil
}

type typeSlice []dataType

func (t typeSlice) size() int {
	return len(t)
}

func (t typeSlice) typeNum() typeNum {
	return typeNumSlice
}

func (t typeSlice) writeTo(w writer) error {
	err := writeCtrlByte(w, t)
	if err != nil {
		return err
	}

	for _, e := range t {
		err := e.writeTo(w)
		if err != nil {
			return err
		}
	}
	return nil
}

type typeString string

func (t typeString) size() int {
	return len(t)
}

func (t typeString) typeNum() typeNum {
	return typeNumString
}

func (t typeString) writeTo(w writer) error {
	err := writeCtrlByte(w, t)
	if err != nil {
		return err
	}

	_, err = w.WriteString(string(t))
	if err != nil {
		return errors.Wrapf(err, `error writing "%s" as a string`, t)
	}
	return nil
}

type typeUint16 uint16

func (t typeUint16) size() int {
	return 2 - bits.LeadingZeros16(uint16(t))/8
}

func (t typeUint16) typeNum() typeNum {
	return typeNumUint16
}

func (t typeUint16) writeTo(w writer) error {
	err := writeCtrlByte(w, t)
	if err != nil {
		return err
	}

	// We ignore leading zeros
	for i := t.size(); i > 0; i-- {
		err = w.WriteByte(byte(t >> (8 * (i - 1)) & 0xFF))
		if err != nil {
			return errors.Wrap(err, "error writing uint16")
		}
	}
	return nil
}

type typeUint32 uint32

func (t typeUint32) size() int {
	return 4 - bits.LeadingZeros32(uint32(t))/8
}

func (t typeUint32) typeNum() typeNum {
	return typeNumUint32
}

func (t typeUint32) writeTo(w writer) error {
	err := writeCtrlByte(w, t)
	if err != nil {
		return err
	}

	// We ignore leading zeros
	for i := t.size(); i > 0; i-- {
		err = w.WriteByte(byte(t >> (8 * (i - 1)) & 0xFF))
		if err != nil {
			return errors.Wrap(err, "error writing uint32")
		}
	}
	return nil
}

type typeUint64 uint64

func (t typeUint64) size() int {
	return 8 - bits.LeadingZeros64(uint64(t))/8
}

func (t typeUint64) typeNum() typeNum {
	return typeNumUint64
}

func (t typeUint64) writeTo(w writer) error {
	err := writeCtrlByte(w, t)
	if err != nil {
		return err
	}

	// We ignore leading zeros
	for i := t.size(); i > 0; i-- {
		err = w.WriteByte(byte(t >> (8 * (i - 1)) & 0xFF))
		if err != nil {
			return errors.Wrap(err, "error writing uint64")
		}
	}
	return nil
}

type typeUint128 big.Int

func (t *typeUint128) size() int {
	return ((*big.Int)(t).BitLen() + 7) / 8
}

func (t *typeUint128) typeNum() typeNum {
	return typeNumUint128
}

func (t *typeUint128) writeTo(w writer) error {
	err := writeCtrlByte(w, t)
	if err != nil {
		return err
	}

	_, err = w.Write((*big.Int)(t).Bytes())
	if err != nil {
		return errors.Wrap(err, "error writing uint128")
	}
	return nil
}

const (
	firstSize  = 29
	secondSize = firstSize + 256
	thirdSize  = secondSize + (1 << 16)
	maxSize    = thirdSize + (1 << 24)
)

func writeCtrlByte(w writer, t dataType) error {
	size := t.size()

	typeNum := t.typeNum()

	var firstByte byte
	var secondByte byte

	if typeNum < 8 {
		firstByte = byte(typeNum << 5)
	} else {
		firstByte = byte(typeNumExtended << 5)
		secondByte = byte(typeNum - 7)
	}

	leftOver := 0
	leftOverSize := 0
	switch {
	case size < firstSize:
		firstByte |= byte(size)
	case size <= secondSize:
		firstByte |= 29
		leftOver = size - firstSize
		leftOverSize = 1
	case size <= thirdSize:
		firstByte |= 30
		leftOver = size - secondSize
		leftOverSize = 2
	case size <= maxSize:
		firstByte |= 31
		leftOver = size - thirdSize
		leftOverSize = 3
	default:
		return errors.Errorf(
			"cannot store %d bytes; max size is %d",
			size,
			maxSize,
		)
	}

	err := w.WriteByte(firstByte)
	if err != nil {
		return errors.Wrapf(
			err,
			"error writing first ctrl byte (type: %d, size: %d)",
			typeNum,
			size,
		)
	}

	if secondByte != 0 {
		err = w.WriteByte(secondByte)
		if err != nil {
			return errors.Wrapf(
				err,
				"error writing second ctrl byte (type: %d, size: %d)",
				typeNum,
				size,
			)
		}
	}

	for i := leftOverSize - 1; i >= 0; i-- {
		v := byte((leftOver >> (8 * i)) & 0xFF)
		err = w.WriteByte(v)
		if err != nil {
			return errors.Wrapf(
				err,
				"error writing remaining ctrl bytes (type: %d, size: %d, value: %d)",
				typeNum,
				size,
				v,
			)
		}
	}
	return nil
}
