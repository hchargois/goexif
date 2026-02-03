// Package tiff implements TIFF decoding as defined in TIFF 6.0 specification at
// http://partners.adobe.com/public/developer/en/tiff/TIFF6.pdf
package tiff

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// ReadAtReader is used when decoding Tiff tags and directories
type ReadAtReader interface {
	io.Reader
	io.ReaderAt
}

// Tiff provides access to a decoded tiff data structure.
type Tiff struct {
	// Dirs is an ordered slice of the tiff's Image File Directories (IFDs).
	// The IFD at index 0 is IFD0.
	Dirs []*Dir
	// The tiff's byte-encoding (i.e. big/little endian).
	Order binary.ByteOrder
}

// Decode parses tiff-encoded data from r and returns a Tiff struct that
// reflects the structure and content of the tiff data. The first read from r
// should be the first byte of the tiff-encoded data and not necessarily the
// first byte of an os.File object.
func Decode(r io.Reader) (*Tiff, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, errors.New("tiff: could not read data")
	}
	return DecodeBytes(data)
}

// DecodeBytes does the same as Decode but should be used when you have the
// entire tiff data in a byte slice.
func DecodeBytes(data []byte) (*Tiff, error) {
	buf := bytes.NewReader(data)

	t := new(Tiff)

	// read byte order
	bo := make([]byte, 2)
	if _, err := io.ReadFull(buf, bo); err != nil {
		return nil, errors.New("tiff: could not read tiff byte order")
	}
	if string(bo) == "II" {
		t.Order = binary.LittleEndian
	} else if string(bo) == "MM" {
		t.Order = binary.BigEndian
	} else {
		return nil, errors.New("tiff: could not read tiff byte order")
	}

	// check for special tiff marker
	var sp int16
	err := binary.Read(buf, t.Order, &sp)
	if err != nil || 42 != sp {
		return nil, errors.New("tiff: could not find special tiff marker")
	}

	// load offset to first IFD
	var offset int32
	err = binary.Read(buf, t.Order, &offset)
	if err != nil {
		return nil, errors.New("tiff: could not read offset to first IFD")
	}

	// load IFD's
	var d *Dir
	prev := offset
	for offset != 0 {
		if int(offset) >= len(data) {
			return nil, errors.New("tiff: seek offset after EOF")
		}

		// load the dir
		d, offset, err = DecodeDirBytes(data, int(offset), t.Order)
		if err != nil {
			return nil, err
		}

		if offset == prev {
			return nil, errors.New("tiff: recursive IFD")
		}
		prev = offset

		t.Dirs = append(t.Dirs, d)
	}

	return t, nil
}

func (tf *Tiff) String() string {
	var buf bytes.Buffer
	fmt.Fprint(&buf, "Tiff{")
	for _, d := range tf.Dirs {
		fmt.Fprintf(&buf, "%s, ", d.String())
	}
	fmt.Fprintf(&buf, "}")
	return buf.String()
}

// Dir provides access to the parsed content of a tiff Image File Directory (IFD).
type Dir struct {
	Tags []*Tag
}

// DecodeDir parses a tiff-encoded IFD from r and returns a Dir object.  offset
// is the offset to the next IFD.  The first read from r should be at the first
// byte of the IFD. ReadAt offsets should generally be relative to the
// beginning of the tiff structure (not relative to the beginning of the IFD).
func DecodeDir(r ReadAtReader, order binary.ByteOrder) (d *Dir, offset int32, err error) {
	d = new(Dir)

	// get num of tags in ifd
	var nTags int16
	err = binary.Read(r, order, &nTags)
	if err != nil {
		return nil, 0, errors.New("tiff: failed to read IFD tag count: " + err.Error())
	}

	// load tags
	for n := 0; n < int(nTags); n++ {
		t, err := DecodeTag(r, order)
		if err != nil {
			return nil, 0, err
		}
		d.Tags = append(d.Tags, t)
	}

	// get offset to next ifd
	err = binary.Read(r, order, &offset)
	if err != nil {
		return nil, 0, errors.New("tiff: falied to read offset to next IFD: " + err.Error())
	}

	return d, offset, nil
}

func (d *Dir) String() string {
	s := "Dir{"
	for _, t := range d.Tags {
		s += t.String() + ", "
	}
	return s + "}"
}

// DecodeDirBytes parses a tiff-encoded IFD from data starting at offset and returns a Dir object.
// nextOffset is the offset to the next IFD. Value offsets within tags are relative to the
// beginning of data (not relative to offset).
func DecodeDirBytes(data []byte, offset int, order binary.ByteOrder) (d *Dir, nextOffset int32, err error) {
	if offset+2 > len(data) {
		return nil, 0, errors.New("tiff: failed to read IFD tag count: insufficient data")
	}

	nTags := int16(order.Uint16(data[offset : offset+2]))
	offset += 2
	d = &Dir{
		Tags: make([]*Tag, 0, nTags),
	}

	for n := 0; n < int(nTags); n++ {
		t, err := DecodeTagBytes(data, offset, order)
		if err != nil {
			return nil, 0, err
		}
		d.Tags = append(d.Tags, t)
		offset += 12
	}

	if offset+4 > len(data) {
		return nil, 0, errors.New("tiff: failed to read offset to next IFD: insufficient data")
	}

	nextOffset = int32(order.Uint32(data[offset : offset+4]))

	return d, nextOffset, nil
}
