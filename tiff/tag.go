package tiff

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Format specifies the Go type equivalent used to represent the basic
// tiff data types.
type Format int

const (
	IntVal Format = iota
	FloatVal
	RatVal
	StringVal
	UndefVal
	OtherVal
)

var ErrShortReadTagValue = errors.New("tiff: short read of tag value")

var formatNames = map[Format]string{
	IntVal:    "int",
	FloatVal:  "float",
	RatVal:    "rational",
	StringVal: "string",
	UndefVal:  "undefined",
	OtherVal:  "other",
}

// DataType represents the basic tiff tag data types.
type DataType uint16

const (
	DTByte      DataType = 1
	DTAscii     DataType = 2
	DTShort     DataType = 3
	DTLong      DataType = 4
	DTRational  DataType = 5
	DTSByte     DataType = 6
	DTUndefined DataType = 7
	DTSShort    DataType = 8
	DTSLong     DataType = 9
	DTSRational DataType = 10
	DTFloat     DataType = 11
	DTDouble    DataType = 12
)

var typeNames = map[DataType]string{
	DTByte:      "byte",
	DTAscii:     "ascii",
	DTShort:     "short",
	DTLong:      "long",
	DTRational:  "rational",
	DTSByte:     "signed byte",
	DTUndefined: "undefined",
	DTSShort:    "signed short",
	DTSLong:     "signed long",
	DTSRational: "signed rational",
	DTFloat:     "float",
	DTDouble:    "double",
}

// typeSize specifies the size in bytes of each type.
var typeSize = map[DataType]uint32{
	DTByte:      1,
	DTAscii:     1,
	DTShort:     2,
	DTLong:      4,
	DTRational:  8,
	DTSByte:     1,
	DTUndefined: 1,
	DTSShort:    2,
	DTSLong:     4,
	DTSRational: 8,
	DTFloat:     4,
	DTDouble:    8,
}

// Tag reflects the parsed content of a tiff IFD tag.
type Tag struct {
	// Id is the 2-byte tiff tag identifier.
	Id uint16
	// Type is an integer (1 through 12) indicating the tag value's data type.
	Type DataType
	// Count is the number of type Type stored in the tag's value (i.e. the
	// tag's value is an array of type Type and length Count).
	Count uint32
	// Val holds the bytes that represent the tag's value.
	Val []byte
	// ValOffset holds byte offset of the tag value w.r.t. the beginning of the
	// reader it was decoded from. Zero if the tag value fit inside the offset
	// field.
	ValOffset uint32

	bigEndian bool
}

func (t *Tag) order() binary.ByteOrder {
	if t.bigEndian {
		return binary.BigEndian
	}
	return binary.LittleEndian
}

// DecodeTag parses a tiff-encoded IFD tag from r and returns a Tag object. The
// first read from r should be the first byte of the tag. ReadAt offsets should
// generally be relative to the beginning of the tiff structure (not relative
// to the beginning of the tag).
func DecodeTag(r ReadAtReader, order binary.ByteOrder) (*Tag, error) {
	t := new(Tag)
	t.bigEndian = order == binary.BigEndian

	var raw [12]byte
	if _, err := io.ReadFull(r, raw[:]); err != nil {
		return nil, errors.New("tiff: could not read tag header: " + err.Error())
	}
	t.Id = order.Uint16(raw[0:2])
	t.Type = DataType(order.Uint16(raw[2:4]))
	t.Count = order.Uint32(raw[4:8])

	var err error

	// There seems to be a relatively common corrupt tag which has a Count of
	// MaxUint32. This is probably not a valid value, so return early.
	if t.Count == 1<<32-1 {
		return t, errors.New("invalid Count offset in tag")
	}

	valLen := typeSize[t.Type] * t.Count
	if valLen == 0 {
		return t, errors.New("zero length tag value")
	}

	if valLen > 4 {
		t.ValOffset = order.Uint32(raw[8:12])

		if valLen <= 32*1024 {
			// if the length is < 32kiB, we'll trust it and allocate the slice;
			t.Val = make([]byte, valLen)
			err = ReadFullAt(r, t.Val, int64(t.ValOffset))
		} else {
			// otherwise we use a bytes.Buffer so we don't allocate a huge slice if the tag
			// is corrupt.
			sr := io.NewSectionReader(r, int64(t.ValOffset), int64(valLen))
			t.Val, err = io.ReadAll(sr)
		}
		n := len(t.Val)
		if err != nil {
			return t, errors.New("tiff: tag value read failed: " + err.Error())
		} else if n != int(valLen) {
			return t, ErrShortReadTagValue
		}
	} else {
		t.Val = raw[8 : 8+valLen]
	}

	return t, nil
}

// DecodeTagBytes parses a tiff-encoded IFD tag from data starting at offset and returns a Tag object.
// The offset should point to the first byte of the tag. Value offsets are relative to the beginning
// of data (not relative to offset).
func DecodeTagBytes(data []byte, offset int, order binary.ByteOrder) (*Tag, error) {
	t := new(Tag)
	err := decodeTagBytesInto(t, data, offset, order)
	if err != nil {
		return t, err
	}
	return t, nil
}

func decodeTagBytesInto(t *Tag, data []byte, offset int, order binary.ByteOrder) error {
	t.bigEndian = order == binary.BigEndian

	if offset+12 > len(data) {
		return errors.New("tiff: could not read tag header: insufficient data")
	}

	raw := data[offset : offset+12]
	t.Id = order.Uint16(raw[0:2])
	t.Type = DataType(order.Uint16(raw[2:4]))
	t.Count = order.Uint32(raw[4:8])

	if t.Count == 1<<32-1 {
		return errors.New("invalid Count offset in tag")
	}

	valLen := typeSize[t.Type] * t.Count
	if valLen == 0 {
		return errors.New("zero length tag value")
	}

	if valLen > 4 {
		t.ValOffset = order.Uint32(raw[8:12])
		valEnd := int(t.ValOffset) + int(valLen)
		if valEnd > len(data) || int(t.ValOffset) > len(data) {
			return ErrShortReadTagValue
		}
		t.Val = make([]byte, valLen)
		copy(t.Val, data[t.ValOffset:valEnd])
	} else {
		t.Val = make([]byte, valLen)
		copy(t.Val, raw[8:8+valLen])
	}

	return nil
}

// ReadFullAt reads exactly len(p) bytes from r starting at offset off into p.
// It returns nil on success.
// If it cannot read len(p) bytes, it returns io.ErrUnexpectedEOF (or another error).
func ReadFullAt(r io.ReaderAt, p []byte, off int64) error {
	for n := 0; n < len(p); {
		m, err := r.ReadAt(p[n:], off+int64(n))
		n += m

		if err == nil {
			continue
		}
		// io.ReaderAt is allowed to return a non-nil err with m > 0.
		// If we filled the buffer, consider it success.
		if n == len(p) {
			return nil
		}
		if err == io.EOF {
			return io.ErrUnexpectedEOF
		}
		return err
	}
	return nil
}

// Format returns a value indicating which method can be called to retrieve the
// tag's value properly typed (e.g. integer, rational, etc.).
func (t *Tag) Format() Format {
	switch t.Type {
	case DTByte, DTShort, DTLong, DTSByte, DTSShort, DTSLong:
		return IntVal
	case DTRational, DTSRational:
		return RatVal
	case DTFloat, DTDouble:
		return FloatVal
	case DTAscii:
		return StringVal
	case DTUndefined:
		return UndefVal
	default:
		return OtherVal
	}
}

func (t *Tag) typeErr(to Format) error {
	return &wrongFmtErr{typeNames[t.Type], formatNames[to]}
}

// Rat returns the tag's i'th value as a rational number. It returns a nil and
// an error if this tag's Format is not RatVal.  It panics for zero deminators
// or if i is out of range.
func (t *Tag) Rat(i int) (*big.Rat, error) {
	n, d, err := t.Rat2(i)
	if err != nil {
		return nil, err
	}
	return big.NewRat(n, d), nil
}

// Rat2 returns the tag's i'th value as a rational number represented by a
// numerator-denominator pair. It returns an error if the tag's Format is not
// RatVal. It panics if i is out of range.
func (t *Tag) Rat2(i int) (num, den int64, err error) {
	off := i * 8
	switch t.Type {
	case DTRational:
		var n, d uint32
		n = t.order().Uint32(t.Val[off : off+4])
		d = t.order().Uint32(t.Val[off+4 : off+8])
		return int64(n), int64(d), nil
	case DTSRational:
		var n, d int32
		n = int32(t.order().Uint32(t.Val[off : off+4]))
		d = int32(t.order().Uint32(t.Val[off+4 : off+8]))
		return int64(n), int64(d), nil
	default:
		return 0, 0, t.typeErr(RatVal)
	}
}

// Int64 returns the tag's i'th value as an integer. It returns an error if the
// tag's Format is not IntVal. It panics if i is out of range.
func (t *Tag) Int64(i int) (int64, error) {
	switch t.Type {
	case DTByte:
		return int64(t.Val[i]), nil
	case DTShort:
		return int64(t.order().Uint16(t.Val[i*2 : i*2+2])), nil
	case DTLong:
		return int64(t.order().Uint32(t.Val[i*4 : i*4+4])), nil
	case DTSByte:
		return int64(int8(t.Val[i])), nil
	case DTSShort:
		return int64(int16(t.order().Uint16(t.Val[i*2 : i*2+2]))), nil
	case DTSLong:
		return int64(int32(t.order().Uint32(t.Val[i*4 : i*4+4]))), nil
	default:
		return 0, t.typeErr(IntVal)
	}
}

// Int returns the tag's i'th value as an integer. It returns an error if the
// tag's Format is not IntVal. It panics if i is out of range.
func (t *Tag) Int(i int) (int, error) {
	i64, err := t.Int64(i)
	return int(i64), err
}

// Float returns the tag's i'th value as a float. It returns an error if the
// tag's Format is not IntVal.  It panics if i is out of range.
func (t *Tag) Float(i int) (float64, error) {
	switch t.Type {
	case DTFloat: // float32
		off := i * 4
		return float64(math.Float32frombits(t.order().Uint32(t.Val[off : off+4]))), nil
	case DTDouble:
		off := i * 8
		return math.Float64frombits(t.order().Uint64(t.Val[off : off+8])), nil
	default:
		return 0, t.typeErr(FloatVal)
	}
}

// StringVal returns the tag's value as a string. It returns an error if the
// tag's Format is not StringVal. It panics if i is out of range.
func (t *Tag) StringVal() (string, error) {
	switch t.Type {
	case DTAscii:
		if len(t.Val) <= 0 {
			return "", nil
		}
		nullPos := bytes.IndexByte(t.Val, 0)
		if nullPos == -1 {
			return string(t.Val), nil
		} else {
			// ignore all trailing NULL bytes, in case of a broken t.Count
			return string(t.Val[:nullPos]), nil
		}
	default:
		return "", t.typeErr(StringVal)
	}
}

// String returns a nicely formatted version of the tag.
func (t *Tag) String() string {
	data, err := t.MarshalJSON()
	if err != nil {
		return "ERROR: " + err.Error()
	}

	if t.Count == 1 {
		return strings.Trim(fmt.Sprintf("%s", data), "[]")
	}
	return fmt.Sprintf("%s", data)
}

func (t *Tag) MarshalJSON() ([]byte, error) {
	format := t.Format()
	switch format {
	case StringVal, UndefVal:
		return nullString(t.Val), nil
	case OtherVal:
		return []byte(fmt.Sprintf("unknown tag type '%v'", t.Type)), nil
	}

	rv := []string{}
	for i := 0; i < int(t.Count); i++ {
		switch format {
		case RatVal:
			n, d, _ := t.Rat2(i)
			rv = append(rv, fmt.Sprintf(`"%v/%v"`, n, d))
		case FloatVal:
			v, _ := t.Float(i)
			rv = append(rv, fmt.Sprintf("%v", v))
		case IntVal:
			v, _ := t.Int(i)
			rv = append(rv, fmt.Sprintf("%v", v))
		}
	}
	return []byte(fmt.Sprintf(`[%s]`, strings.Join(rv, ","))), nil
}

func nullString(in []byte) []byte {
	rv := bytes.Buffer{}
	rv.WriteByte('"')
	for _, b := range in {
		if unicode.IsPrint(rune(b)) {
			rv.WriteByte(b)
		}
	}
	rv.WriteByte('"')
	rvb := rv.Bytes()
	if utf8.Valid(rvb) {
		return rvb
	}
	return []byte(`""`)
}

type wrongFmtErr struct {
	From, To string
}

func (e *wrongFmtErr) Error() string {
	return fmt.Sprintf("cannot convert tag type '%v' into '%v'", e.From, e.To)
}
