package netcode

import (
	"io"
)

func ReadUint8(buf []byte, out *uint8) ([]byte, error) {
	if len(buf) < SizeUint8 {
		return nil, io.EOF
	}

	*out = uint8(buf[0])
	return buf[SizeUint8:], nil
}

func ReadUint16(buf []byte, out *uint16) ([]byte, error) {
	if len(buf) < SizeUint16 {
		return nil, io.EOF
	}

	*out |= uint16(buf[0])
	*out |= uint16(buf[1]) << 8
	return buf[SizeUint16:], nil
}

func ReadUint32(buf []byte, out *uint32) ([]byte, error) {
	if len(buf) < SizeUint32 {
		return nil, io.EOF
	}

	*out |= uint32(buf[0])
	*out |= uint32(buf[1]) << 8
	*out |= uint32(buf[2]) << 16
	*out |= uint32(buf[3]) << 24
	return buf[SizeUint32:], nil
}

func ReadUint64(buf []byte, out *uint64) ([]byte, error) {
	if len(buf) < SizeUint8 {
		return nil, io.EOF
	}

	*out |= uint64(buf[0])
	*out |= uint64(buf[1]) << 8
	*out |= uint64(buf[2]) << 16
	*out |= uint64(buf[3]) << 24
	*out |= uint64(buf[4]) << 32
	*out |= uint64(buf[5]) << 40
	*out |= uint64(buf[6]) << 48
	*out |= uint64(buf[7]) << 56
	return buf[SizeUint64:], nil
}

func ReadInt8(buf []byte, out *int8) ([]byte, error) {
	if len(buf) < SizeInt8 {
		return nil, io.EOF
	}

	*out = int8(buf[0])
	return buf[SizeInt8:], nil
}

func ReadInt16(buf []byte, out *int16) ([]byte, error) {
	if len(buf) < SizeInt16 {
		return nil, io.EOF
	}

	*out |= int16(buf[0])
	*out |= int16(buf[1]) << 8
	return buf[SizeInt16:], nil
}

func ReadInt32(buf []byte, out *int32) ([]byte, error) {
	if len(buf) < SizeInt32 {
		return nil, io.EOF
	}

	*out |= int32(buf[0])
	*out |= int32(buf[1]) << 8
	*out |= int32(buf[2]) << 16
	*out |= int32(buf[3]) << 24
	return buf[SizeInt32:], nil
}

func ReadInt64(buf []byte, out *int64) ([]byte, error) {
	if len(buf) < SizeInt64 {
		return nil, io.EOF
	}
	*out |= int64(buf[0])
	*out |= int64(buf[1]) << 8
	*out |= int64(buf[2]) << 16
	*out |= int64(buf[3]) << 24
	*out |= int64(buf[4]) << 32
	*out |= int64(buf[5]) << 40
	*out |= int64(buf[6]) << 48
	*out |= int64(buf[7]) << 56
	return buf[SizeInt64:], nil
}

func ReadByte(buf []byte, out *byte) ([]byte, error) {
	if len(buf) < SizeByte {
		return nil, io.EOF
	}

	return ReadUint8(buf, out)
}

func ReadBytes(buf []byte, out *[]byte, length int) ([]byte, error) {
	if len(buf) < length {
		return nil, io.EOF
	}
	*out = buf[0:length]
	return buf[length:], nil
}

func WriteByte(buf []byte, in byte) ([]byte, error) {
	if len(buf) < SizeByte {
		return nil, io.EOF
	}

	buf[0] = uint8(in)
	return buf[SizeByte:], nil
}

func WriteBytes(buf []byte, in []byte) ([]byte, error) {
	if len(buf) < len(in) {
		return nil, io.EOF
	}

	for i := 0; i < len(in); i++ {
		buf[i] = uint8(in[i])
	}
	return buf[len(in):], nil
}

func WriteBytesN(buf []byte, in []byte, length int) ([]byte, error) {
	if len(buf) < length {
		return nil, io.EOF
	}

	for i := 0; i < length; i++ {
		buf[i] = uint8(in[i])
	}
	return buf[length:], nil
}

func WriteUint8(buf []byte, in uint8) ([]byte, error) {
	if len(buf) < SizeUint8 {
		return nil, io.EOF
	}

	buf[0] = uint8(in)
	return buf[SizeUint8:], nil
}

func WriteUint16(buf []byte, in uint16) ([]byte, error) {
	if len(buf) < SizeUint16 {
		return nil, io.EOF
	}

	buf[0] = byte(in)
	buf[1] = byte(in >> 8)
	return buf[SizeUint16:], nil
}

func WriteUint32(buf []byte, in uint32) ([]byte, error) {
	if len(buf) < SizeUint32 {
		return nil, io.EOF
	}

	buf[0] = byte(in)
	buf[1] = byte(in >> 8)
	buf[2] = byte(in >> 16)
	buf[3] = byte(in >> 24)
	return buf[SizeUint32:], nil
}

func WriteUint64(buf []byte, in uint64) ([]byte, error) {
	if len(buf) < SizeUint64 {
		return nil, io.EOF
	}

	for i := uint(0); i < uint(SizeUint64); i++ {
		buf[i] = byte(in >> (i * 8))
	}
	return buf[SizeUint64:], nil
}

func WriteInt8(buf []byte, in int8) ([]byte, error) {
	if len(buf) < SizeInt8 {
		return nil, io.EOF
	}

	buf[0] = byte(in)
	return buf[SizeInt8:], nil
}

func WriteInt16(buf []byte, in int16) ([]byte, error) {
	if len(buf) < SizeInt16 {
		return nil, io.EOF
	}

	buf[0] = byte(in)
	buf[1] = byte(in >> 8)
	return buf[SizeInt16:], nil
}

func WriteInt32(buf []byte, in int32) ([]byte, error) {
	if len(buf) < SizeInt32 {
		return nil, io.EOF
	}

	buf[0] = byte(in)
	buf[1] = byte(in >> 8)
	buf[2] = byte(in >> 16)
	buf[3] = byte(in >> 24)
	return buf[SizeInt32:], nil
}

func WriteInt64(buf []byte, in int64) ([]byte, error) {
	if len(buf) < SizeInt64 {
		return nil, io.EOF
	}

	for i := uint(0); i < uint(SizeInt64); i++ {
		buf[i] = byte(in >> (i * 8))
	}
	return buf[SizeUint64:], nil
}
