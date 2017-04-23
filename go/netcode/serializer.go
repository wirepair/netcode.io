package netcode

func ReadUint8(buf []byte, out *uint8) []byte {
	*out = uint8(buf[0])
	return buf[SizeUint8:]
}

func ReadUint16(buf []byte, out *uint16) []byte {
	*out |= uint16(buf[0])
	*out |= uint16(buf[1]) << 8
	return buf[SizeUint16:]
}

func ReadUint32(buf []byte, out *uint32) []byte {
	*out |= uint32(buf[0])
	*out |= uint32(buf[1]) << 8
	*out |= uint32(buf[2]) << 16
	*out |= uint32(buf[3]) << 24
	return buf[SizeUint32:]
}

func ReadUint64(buf []byte, out *uint64) []byte {
	*out |= uint64(buf[0])
	*out |= uint64(buf[1]) << 8
	*out |= uint64(buf[2]) << 16
	*out |= uint64(buf[3]) << 24
	*out |= uint64(buf[4]) << 32
	*out |= uint64(buf[5]) << 40
	*out |= uint64(buf[6]) << 48
	*out |= uint64(buf[7]) << 56
	return buf[SizeUint64:]
}

func ReadInt8(buf []byte, out *int8) []byte {
	*out = int8(buf[0])
	return buf[SizeInt8:]
}

func ReadInt16(buf []byte, out *int16) []byte {
	*out |= int16(buf[0])
	*out |= int16(buf[1]) << 8
	return buf[SizeInt16:]
}

func ReadInt32(buf []byte, out *int32) []byte {
	*out |= int32(buf[0])
	*out |= int32(buf[1]) << 8
	*out |= int32(buf[2]) << 16
	*out |= int32(buf[3]) << 24
	return buf[SizeInt32:]
}

func ReadInt64(buf []byte, out *int64) []byte {
	*out |= int64(buf[0])
	*out |= int64(buf[1]) << 8
	*out |= int64(buf[2]) << 16
	*out |= int64(buf[3]) << 24
	*out |= int64(buf[4]) << 32
	*out |= int64(buf[5]) << 40
	*out |= int64(buf[6]) << 48
	*out |= int64(buf[7]) << 56
	return buf[SizeInt64:]
}

func ReadByte(buf []byte, out *byte) []byte {
	return ReadUint8(buf, out)
}

func ReadBytes(buf []byte, out *[]byte, length int) []byte {
	*out = buf[0:length]
	return buf[length:]
}

func WriteByte(buf []byte, in byte) []byte {
	buf[0] = uint8(in)
	return buf[SizeByte:]
}

func WriteBytes(buf []byte, in []byte) []byte {
	for i := 0; i < len(in); i++ {
		buf[i] = uint8(in[i])
	}
	return buf[len(in):]
}

func WriteBytesN(buf []byte, in []byte, length int) []byte {
	for i := 0; i < length; i++ {
		buf[i] = uint8(in[i])
	}
	return buf[length:]
}

func WriteUint8(buf []byte, in uint8) []byte {
	buf[0] = uint8(in)
	return buf[SizeUint8:]
}

func WriteUint16(buf []byte, in uint16) []byte {
	buf[0] = byte(in)
	buf[1] = byte(in >> 8)
	return buf[SizeUint16:]
}

func WriteUint32(buf []byte, in uint32) []byte {
	buf[0] = byte(in)
	buf[1] = byte(in >> 8)
	buf[2] = byte(in >> 16)
	buf[3] = byte(in >> 24)
	return buf[SizeUint32:]
}

func WriteUint64(buf []byte, in uint64) []byte {
	for i := uint(0); i < uint(SizeUint64); i++ {
		buf[i] = byte(in >> (i * 8))
	}
	return buf[SizeUint64:]
}

func WriteInt8(buf []byte, in int8) []byte {
	buf[0] = byte(in)
	return buf[SizeInt8:]
}

func WriteInt16(buf []byte, in int16) []byte {
	buf[0] = byte(in)
	buf[1] = byte(in >> 8)
	return buf[SizeInt16:]
}

func WriteInt32(buf []byte, in int32) []byte {
	buf[0] = byte(in)
	buf[1] = byte(in >> 8)
	buf[2] = byte(in >> 16)
	buf[3] = byte(in >> 24)
	return buf[SizeInt32:]
}

func WriteInt64(buf []byte, in int64) []byte {
	for i := uint(0); i < uint(SizeInt64); i++ {
		buf[i] = byte(in >> (i * 8))
	}
	return buf[SizeUint64:]
}
