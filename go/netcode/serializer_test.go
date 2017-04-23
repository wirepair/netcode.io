package netcode

import (
	"bytes"
	"testing"
)

func TestReadUint8(t *testing.T) {
	buf := make([]byte, SizeUint8)
	bufRef := buf

	val := uint8(0x1e)
	buf = WriteUint8(buf, val)

	var out uint8
	bufRef = ReadUint8(bufRef, &out)
	if out != val {
		t.Fatalf("error values did not match: %d != %d\n", out, val)
	}
}

func TestReadUint16(t *testing.T) {
	buf := make([]byte, SizeUint16)
	bufRef := buf

	val := uint16(0x1eff)
	buf = WriteUint16(buf, val)

	var out uint16
	bufRef = ReadUint16(bufRef, &out)
	if out != val {
		t.Fatalf("error values did not match: %d != %d\n", out, val)
	}
}

func TestReadUint32(t *testing.T) {
	buf := make([]byte, SizeUint32)
	bufRef := buf

	val := uint32(0x1e1e1e1e)
	buf = WriteUint32(buf, val)

	var out uint32
	bufRef = ReadUint32(bufRef, &out)
	if out != val {
		t.Fatalf("error values did not match: %d != %d\n", out, val)
	}
}

func TestReadUint64(t *testing.T) {
	buf := make([]byte, SizeUint64)
	bufRef := buf

	val := uint64(0x1e1e1e1efefefefe)
	buf = WriteUint64(buf, val)

	var out uint64
	bufRef = ReadUint64(bufRef, &out)
	if out != val {
		t.Fatalf("error values did not match: %d != %d\n", out, val)
	}
}

func TestReadByte(t *testing.T) {
	buf := make([]byte, 1)
	buf[0] = 0xfe
	bufRef := buf

	var out byte
	buf = ReadByte(buf, &out)
	if bufRef[0] != out {
		t.Fatalf("bytes did not match: %v %v\n", bufRef, out)
	}

	if len(buf) != 0 {
		t.Fatalf("buffer still has bytes left")
	}
}

func TestReadBytes(t *testing.T) {
	buf := make([]byte, 10)
	bufRef := buf

	copy(buf, string("0123456789"))

	out := make([]byte, 10)

	buf = ReadBytes(buf, &out, 10)
	if !bytes.Equal(bufRef, out) {
		t.Fatalf("bytes did not match: %v %v\n", bufRef, out)
	}

	if len(buf) != 0 {
		t.Fatalf("buffer still has bytes left")
	}
}

func TestWriteBytesN(t *testing.T) {
	buf := make([]byte, 10)
	bufRef := buf
	bufRef2 := buf

	buf = WriteBytesN(buf, []byte("abc"), 3)

	out := make([]byte, 10)

	buf = ReadBytes(bufRef, &out, 3)
	if !bytes.Equal(bufRef2[:3], out) {
		t.Fatalf("bytes did not match: %v %v\n", bufRef, out)
	}

	if len(buf) == 0 {
		t.Fatalf("buffer has no bytes left")
	}
}
