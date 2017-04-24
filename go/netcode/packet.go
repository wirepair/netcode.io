package netcode

import (
	"bytes"
	"errors"
	"log"
	"strconv"
)

const MAX_CLIENTS = 60
const CONNECT_TOKEN_PRIVATE_BYTES = 1024
const CHALLENGE_TOKEN_BYTES = 300
const VERSION_INFO_BYTES = 13
const USER_DATA_BYTES = 256
const MAX_PACKET_BYTES = 1220
const MAX_PAYLOAD_BYTES = 1200
const MAX_ADDRESS_STRING_LENGTH = 256
const REPLAY_PROTECTION_BUFFER_SIZE = 256

const KEY_BYTES = 32
const MAC_BYTES = 16
const NONCE_BYTES = 8
const MAX_SERVERS_PER_CONNECT = 32

var VERSION_INFO = []byte("NETCODE 1.00\x00")

// Used for determining the type of packet, part of the serialization protocol
type PacketType uint8

const (
	ConnectionRequest PacketType = iota
	ConnectionDenied
	ConnectionChallenge
	ConnectionResponse
	ConnectionKeepAlive
	ConnectionPayload
	ConnectionDisconnect
)

func (p PacketType) Peek(packetBuffer []byte) PacketType {
	prefix := uint8(packetBuffer[0])
	return PacketType(prefix & 0xF)
}

// reference map of packet -> string values
var packetTypeMap = map[PacketType]string{
	ConnectionRequest:    "CONNECTION_REQUEST",
	ConnectionDenied:     "CONNECTION_DENIED",
	ConnectionChallenge:  "CONNECTION_CHALLENGE",
	ConnectionResponse:   "CONNECTION_RESPONSE",
	ConnectionKeepAlive:  "CONNECTION_KEEPALIVE",
	ConnectionPayload:    "CONNECTION_PAYLOAD",
	ConnectionDisconnect: "CONNECTION_DISCONNECT",
}

// not a packet type, but value is last packetType+1
const ConnectionNumPackets = ConnectionDisconnect + 1

// Packet interface supporting reading and writing.
type Packet interface {
	GetType() PacketType                                                                                                                                                      // The type of packet
	Sequence() uint64                                                                                                                                                         // sequence number of this packet, if it supports it                                                                                                                                           // returns the packet type
	Write(buffer []byte, protocolId, sequence uint64, writePacketKey []byte) (int, error)                                                                                     // writes and encrypts the packet data to the supplied buffer.
	Read(packetBuffer []byte, packetLen int, protocolId, currentTimestamp uint64, readPacketKey, privateKey, allowedPackets []byte, replayProtection *ReplayProtection) error // reads in and decrypts from the supplied buffer to set the packet properties
}

// Returns the type of packet given packetbuffer by peaking the packet type
func NewPacket(packetBuffer []byte) Packet {
	var packetType PacketType
	t := packetType.Peek(packetBuffer)
	switch t {
	case ConnectionRequest:
		return &RequestPacket{}
	case ConnectionDenied:
		return &DeniedPacket{}
	case ConnectionChallenge:
		return &ChallengePacket{}
	case ConnectionResponse:
		return &ResponsePacket{}
	case ConnectionKeepAlive:
		return &KeepAlivePacket{}
	case ConnectionPayload:
		return &PayloadPacket{}
	case ConnectionDisconnect:
		return &DisconnectPacket{}
	}
	return nil
}

// The connection request packet
type RequestPacket struct {
	VersionInfo                 []byte               // version information of communications
	ProtocolId                  uint64               // protocol id used in communications
	ConnectTokenExpireTimestamp uint64               // when the connect token expires
	ConnectTokenSequence        uint64               // the sequence id of this token
	Token                       *ConnectTokenPrivate // reference to the private parts of this packet
	ConnectTokenData            []byte               // the encrypted Token after Write -> Encrypt
}

// request packets do not have a sequence value
func (p *RequestPacket) Sequence() uint64 {
	return 0
}

// Writes the RequestPacket data to a supplied buffer and returns the length of bytes written to it.
func (p *RequestPacket) Write(buffer []byte, protocolId, sequence uint64, writePacketKey []byte) (int, error) {
	start := buffer
	buffer, _ = WriteUint8(buffer, uint8(ConnectionRequest))
	buffer, _ = WriteBytes(buffer, p.VersionInfo)
	buffer, _ = WriteUint64(buffer, p.ProtocolId)
	buffer, _ = WriteUint64(buffer, p.ConnectTokenExpireTimestamp)
	buffer, _ = WriteUint64(buffer, p.ConnectTokenSequence)
	buffer, _ = WriteBytes(buffer, p.ConnectTokenData) // write the encrypted connection token private data
	if len(start)-len(buffer) != 1+13+8+8+8+CONNECT_TOKEN_PRIVATE_BYTES {
		return -1, errors.New("invalid buffer size written")
	}
	return len(start) - len(buffer), nil
}

// Reads a request packet and decrypts the connect token private data. Request packets do not return a sequenceId
func (p *RequestPacket) Read(packetBuffer []byte, packetLen int, protocolId, currentTimestamp uint64, readPacketKey, privateKey, allowedPackets []byte, replayProtection *ReplayProtection) error {
	var err error
	var packetType uint8

	start := packetBuffer

	if packetBuffer, err = ReadUint8(packetBuffer, &packetType); err != nil || PacketType(packetType) != ConnectionRequest {
		return errors.New("invalid packet type")
	}

	if allowedPackets[0] == 0 {
		return errors.New("ignored connection request packet. packet type is not allowed")
	}

	if packetLen != 1+VERSION_INFO_BYTES+8+8+8+CONNECT_TOKEN_PRIVATE_BYTES {
		return errors.New("ignored connection request packet. bad packet length")
	}

	if privateKey == nil {
		return errors.New("ignored connection request packet. no private key\n")
	}

	if packetBuffer, err = ReadBytes(packetBuffer, &p.VersionInfo, VERSION_INFO_BYTES); err != nil {
		return errors.New("ignored connection request packet. bad version info invalid bytes returned\n")
	}

	if !bytes.Equal(p.VersionInfo, VERSION_INFO) {
		return errors.New("ignored connection request packet. bad version info did not match\n")
	}

	packetBuffer, err = ReadUint64(packetBuffer, &p.ProtocolId)
	if err != nil || p.ProtocolId != protocolId {
		return errors.New("ignored connection request packet. wrong protocol id\n")
	}

	packetBuffer, err = ReadUint64(packetBuffer, &p.ConnectTokenExpireTimestamp)
	if err != nil || p.ConnectTokenExpireTimestamp <= currentTimestamp {
		return errors.New("ignored connection request packet. connect token expired\n")
	}

	if packetBuffer, err = ReadUint64(packetBuffer, &p.ConnectTokenSequence); err != nil {
		return err
	}

	if len(start)-len(packetBuffer) != 1+VERSION_INFO_BYTES+8+8+8 {
		return errors.New("invalid length of packet buffer read")
	}

	tokenBuffer := make([]byte, CONNECT_TOKEN_PRIVATE_BYTES)
	if packetBuffer, err = ReadBytes(packetBuffer, &tokenBuffer, CONNECT_TOKEN_PRIVATE_BYTES); err != nil {
		return err
	}

	p.Token = NewConnectTokenPrivateEncrypted(tokenBuffer)
	if _, err := p.Token.Decrypt(p.ProtocolId, p.ConnectTokenExpireTimestamp, p.ConnectTokenSequence, privateKey); err != nil {
		return errors.New("error decrypting connect token private data: " + err.Error())
	}

	if err := p.Token.Read(); err != nil {
		return errors.New("error reading decrypted connect token private data: " + err.Error())
	}

	return nil
}

func (p *RequestPacket) GetType() PacketType {
	return ConnectionRequest
}

// Denied packet type, contains no information
type DeniedPacket struct {
	sequence uint64
}

func (p *DeniedPacket) Sequence() uint64 {
	return p.sequence
}

func (p *DeniedPacket) Write(buffer []byte, protocolId, sequence uint64, writePacketKey []byte) (int, error) {
	start := buffer
	prefixByte, err := writePacketPrefix(p, &buffer, sequence)
	if err != nil {
		return -1, err
	}
	// denied packets are empty
	pos := len(start) - len(buffer)
	encryptLength, err := encryptPacket(buffer, pos, pos, prefixByte, protocolId, sequence, writePacketKey)
	log.Printf("%#v\n", start[:encryptLength])
	return encryptLength + pos, err
}

func (p *DeniedPacket) Read(packetBuffer []byte, packetLen int, protocolId, currentTimestamp uint64, readPacketKey, privateKey, allowedPackets []byte, replayProtection *ReplayProtection) error {
	//start := packetBuffer
	sequence, decryptedBuf, err := decryptPacket(&packetBuffer, packetLen, protocolId, currentTimestamp, readPacketKey, allowedPackets, replayProtection)
	if err != nil {
		return err
	}
	p.sequence = sequence

	if len(decryptedBuf) != 0 {
		return errors.New("ignored connection denied packet. decrypted packet data is wrong size")
	}
	return nil
}

func (p *DeniedPacket) GetType() PacketType {
	return ConnectionDenied
}

// Challenge packet containing token data and the sequence id used
type ChallengePacket struct {
	sequence               uint64
	ChallengeTokenSequence uint64
	ChallengeTokenData     []byte
}

func (p *ChallengePacket) Sequence() uint64 {
	return p.sequence
}

func (p *ChallengePacket) Write(buffer []byte, protocolId, sequence uint64, writePacketKey []byte) (int, error) {
	start := buffer

	prefixByte, err := writePacketPrefix(p, &buffer, sequence)
	if err != nil {
		return -1, err
	}

	encryptedStart := len(start) - len(buffer)
	buffer, _ = WriteUint64(buffer, p.ChallengeTokenSequence)
	buffer, _ = WriteBytesN(buffer, p.ChallengeTokenData, CHALLENGE_TOKEN_BYTES)
	encryptedFinish := len(start) - len(buffer)

	encryptedBuf := buffer[encryptedStart:encryptedFinish]
	additionalData, nonce := packetCryptData(prefixByte, protocolId, sequence)
	if err := EncryptAead(&buffer[encryptedStart:encryptedFinish], additionalData, nonce, writePacketKey); err != nil {
		return -1, err
	}
	buffer, _ = WriteBytes(start[encryptedStart:], encryptedBuf)
	log.Printf("%p %d\n", &buffer, len(buffer)-len(encryptedBuf))

	//encryptedLen, err := encryptPacket(buffer, encryptedStart, encryptedFinish, prefixByte, protocolId, sequence, writePacketKey)

	return len(buffer) + encryptedStart, err
}

func (p *ChallengePacket) Read(packetBuffer []byte, packetLen int, protocolId, currentTimestamp uint64, readPacketKey, privateKey, allowedPackets []byte, replayProtection *ReplayProtection) error {
	//start := packetBuffer
	sequence, decryptedBuf, err := decryptPacket(&packetBuffer, packetLen, protocolId, currentTimestamp, readPacketKey, allowedPackets, replayProtection)
	if err != nil {
		return err
	}
	p.sequence = sequence

	if len(decryptedBuf) != 8+CHALLENGE_TOKEN_BYTES {
		log.Printf("len: %d expected: %d\n", len(decryptedBuf), 8+CHALLENGE_TOKEN_BYTES)
		return errors.New("ignored connection challenge packet. decrypted packet data is wrong size")
	}

	if decryptedBuf, err = ReadUint64(decryptedBuf, &p.ChallengeTokenSequence); err != nil {
		return errors.New("error reading challenge token sequence")
	}

	if decryptedBuf, err = ReadBytes(decryptedBuf, &p.ChallengeTokenData, CHALLENGE_TOKEN_BYTES); err != nil {
		return errors.New("error reading challenge token data")
	}

	return nil
}

func (p *ChallengePacket) GetType() PacketType {
	return ConnectionChallenge
}

// Response packet, containing the token data and sequence id
type ResponsePacket struct {
	sequence               uint64
	ChallengeTokenSequence uint64
	ChallengeTokenData     []byte
}

func (p *ResponsePacket) Sequence() uint64 {
	return p.sequence
}

func (p *ResponsePacket) Write(buffer []byte, protocolId, sequence uint64, writePacketKey []byte) (int, error) {
	start := buffer
	prefixByte, err := writePacketPrefix(p, &buffer, sequence)
	if err != nil {
		return -1, err
	}

	encryptedStart := len(start) - len(buffer)
	buffer, _ = WriteUint64(buffer, p.ChallengeTokenSequence)
	buffer, _ = WriteBytesN(buffer, p.ChallengeTokenData, CHALLENGE_TOKEN_BYTES)
	encryptedFinish := len(buffer) - encryptedStart
	log.Printf("START %d, FINISH %d\n", encryptedStart, encryptedFinish)
	return encryptPacket(buffer, encryptedStart, encryptedFinish, prefixByte, protocolId, sequence, writePacketKey)
}

func (p *ResponsePacket) Read(packetBuffer []byte, packetLen int, protocolId, currentTimestamp uint64, readPacketKey, privateKey, allowedPackets []byte, replayProtection *ReplayProtection) error {
	//start := packetBuffer
	sequence, decryptedBuf, err := decryptPacket(&packetBuffer, packetLen, protocolId, currentTimestamp, readPacketKey, allowedPackets, replayProtection)
	if err != nil {
		return err
	}
	p.sequence = sequence

	if len(decryptedBuf) != 8+CHALLENGE_TOKEN_BYTES {
		return errors.New("ignored connection challenge response packet. decrypted packet data is wrong size")
	}

	if decryptedBuf, err = ReadUint64(decryptedBuf, &p.ChallengeTokenSequence); err != nil {
		return errors.New("error reading challenge token sequence")
	}

	if decryptedBuf, err = ReadBytes(decryptedBuf, &p.ChallengeTokenData, CHALLENGE_TOKEN_BYTES); err != nil {
		return errors.New("error reading challenge token data")
	}

	return nil
}

func (p *ResponsePacket) GetType() PacketType {
	return ConnectionResponse
}

// used for heart beats
type KeepAlivePacket struct {
	sequence    uint64
	ClientIndex uint32
	MaxClients  uint32
}

func (p *KeepAlivePacket) Sequence() uint64 {
	return p.sequence
}

func (p *KeepAlivePacket) Write(buffer []byte, protocolId, sequence uint64, writePacketKey []byte) (int, error) {
	start := buffer
	prefixByte, err := writePacketPrefix(p, &buffer, sequence)
	if err != nil {
		return -1, err
	}

	encryptedStart := len(start) - len(buffer)
	buffer, _ = WriteUint32(buffer, uint32(p.ClientIndex))
	buffer, _ = WriteUint32(buffer, uint32(p.MaxClients))
	encryptedFinish := len(buffer) - encryptedStart
	return encryptPacket(buffer, encryptedStart, encryptedFinish, prefixByte, protocolId, sequence, writePacketKey)
}

func (p *KeepAlivePacket) Read(packetBuffer []byte, packetLen int, protocolId, currentTimestamp uint64, readPacketKey, privateKey, allowedPackets []byte, replayProtection *ReplayProtection) error {
	//start := packetBuffer
	sequence, decryptedBuf, err := decryptPacket(&packetBuffer, packetLen, protocolId, currentTimestamp, readPacketKey, allowedPackets, replayProtection)
	if err != nil {
		return err
	}
	p.sequence = sequence

	if len(decryptedBuf) != 8 {
		return errors.New("ignored connection keep alive packet. decrypted packet data is wrong size")
	}

	if decryptedBuf, err = ReadUint32(decryptedBuf, &p.ClientIndex); err != nil {
		return errors.New("error reading keepalive client index")
	}

	if decryptedBuf, err = ReadUint32(decryptedBuf, &p.MaxClients); err != nil {
		return errors.New("error reading keepalive max clients")
	}

	return nil
}

func (p *KeepAlivePacket) GetType() PacketType {
	return ConnectionKeepAlive
}

// Contains user supplied payload data between server <-> client
type PayloadPacket struct {
	sequence     uint64
	PayloadBytes uint32
	PayloadData  []byte
}

func (p *PayloadPacket) GetType() PacketType {
	return ConnectionPayload
}

// Helper function to create a new payload packet with the supplied buffer
func NewPayloadPacket(payloadData []byte) *PayloadPacket {
	packet := &PayloadPacket{}
	packet.PayloadBytes = uint32(len(payloadData))
	packet.PayloadData = payloadData
	return packet
}

func (p *PayloadPacket) Sequence() uint64 {
	return p.sequence
}

func (p *PayloadPacket) Write(buffer []byte, protocolId, sequence uint64, writePacketKey []byte) (int, error) {
	start := buffer
	prefixByte, err := writePacketPrefix(p, &buffer, sequence)
	if err != nil {
		return -1, err
	}
	encryptedStart := len(start) - len(buffer)
	buffer, _ = WriteBytesN(buffer, p.PayloadData, int(p.PayloadBytes))
	encryptedFinish := len(buffer) - encryptedStart
	return encryptPacket(buffer, encryptedStart, encryptedFinish, prefixByte, protocolId, sequence, writePacketKey)
}

func (p *PayloadPacket) Read(packetBuffer []byte, packetLen int, protocolId, currentTimestamp uint64, readPacketKey, privateKey, allowedPackets []byte, replayProtection *ReplayProtection) error {
	//start := packetBuffer
	sequence, decryptedBuf, err := decryptPacket(&packetBuffer, packetLen, protocolId, currentTimestamp, readPacketKey, allowedPackets, replayProtection)
	if err != nil {
		return err
	}
	p.sequence = sequence

	decryptedSize := uint32(len(decryptedBuf))
	if decryptedSize < 1 {
		return errors.New("ignored connection payload packet. payload is too small")
	}

	if decryptedSize > MAX_PAYLOAD_BYTES {
		return errors.New("ignored connection payload packet. payload is too large")
	}

	p.PayloadBytes = decryptedSize
	p.PayloadData = decryptedBuf
	return nil
}

// Signals to server/client to disconnect, contains no data.
type DisconnectPacket struct {
	sequence uint64
}

func (p *DisconnectPacket) Sequence() uint64 {
	return p.sequence
}

func (p *DisconnectPacket) Write(buffer []byte, protocolId, sequence uint64, writePacketKey []byte) (int, error) {
	start := buffer
	prefixByte, err := writePacketPrefix(p, &buffer, sequence)
	if err != nil {
		return -1, err
	}
	pos := len(start) - len(buffer)
	// disconnect packets are empty
	return encryptPacket(buffer, pos, pos, prefixByte, protocolId, sequence, writePacketKey)
}

func (p *DisconnectPacket) Read(packetBuffer []byte, packetLen int, protocolId, currentTimestamp uint64, readPacketKey, privateKey, allowedPackets []byte, replayProtection *ReplayProtection) error {
	//start := packetBuffer
	sequence, decryptedBuf, err := decryptPacket(&packetBuffer, packetLen, protocolId, currentTimestamp, readPacketKey, allowedPackets, replayProtection)
	if err != nil {
		return err
	}
	p.sequence = sequence

	if len(decryptedBuf) != 0 {
		return errors.New("ignored connection denied packet. decrypted packet data is wrong size")
	}
	return nil
}

func (p *DisconnectPacket) GetType() PacketType {
	return ConnectionDisconnect
}

// Decrypts the packet after reading in the prefix byte and sequence id. Used for all PacketTypes except RequestPacket. Returns a buffer containing the decrypted data
func decryptPacket(packetBuffer *[]byte, packetLen int, protocolId, currentTimestamp uint64, readPacketKey, allowedPackets []byte, replayProtection *ReplayProtection) (uint64, []byte, error) {
	var packetSequence uint64
	var prefixByte uint8
	var err error

	start := *packetBuffer
	if *packetBuffer, err = ReadUint8(*packetBuffer, &prefixByte); err != nil {
		return 0, nil, errors.New("invalid buffer length")
	}

	if packetSequence, err = readSequence(packetBuffer, packetLen, prefixByte); err != nil {
		return 0, nil, err
	}

	if err := validateSequence(packetLen, prefixByte, packetSequence, readPacketKey, allowedPackets, replayProtection); err != nil {
		return 0, nil, err
	}

	// decrypt the per-packet type data
	additionalData, nonce := packetCryptData(prefixByte, protocolId, packetSequence)

	encryptedSize := packetLen - (len(start) - len(*packetBuffer))
	if encryptedSize < MAC_BYTES {
		return 0, nil, errors.New("ignored encrypted packet. encrypted payload is too small")
	}

	encryptedBuf := make([]byte, encryptedSize)
	if *packetBuffer, err = ReadBytes(*packetBuffer, &encryptedBuf, encryptedSize); err != nil {
		return 0, nil, errors.New("ignored encrypted packet. encrypted payload is too small")
	}
	//log.Printf("encryptedBuf: %#v\n", encryptedBuf)
	decryptedBuf, err := DecryptAead(encryptedBuf, additionalData, nonce, readPacketKey)
	if err != nil {
		return 0, nil, errors.New("ignored encrypted packet. failed to decrypt: " + err.Error())
	}

	return packetSequence, decryptedBuf, nil
}

// Reads and verifies the sequence id
func readSequence(packetBuffer *[]byte, packetLen int, prefixByte uint8) (uint64, error) {
	var err error
	var sequence uint64

	sequenceBytes := prefixByte >> 4
	if sequenceBytes < 1 || sequenceBytes > 8 {
		return 0, errors.New("ignored encrypted packet. sequence bytes is out of range [1,8]")
	}

	if packetLen < 1+int(sequenceBytes)+MAC_BYTES {
		return 0, errors.New("ignored encrypted packet. buffer is too small for sequence bytes + encryption mac")
	}

	var i uint8
	// read variable length sequence number [1,8]
	for i = 0; i < sequenceBytes; i += 1 {
		var val uint8
		*packetBuffer, err = ReadUint8(*packetBuffer, &val)
		if err != nil {
			return 0, err
		}
		sequence |= (uint64(val) << (8 * i))
	}
	return sequence, nil
}

// Validates the data prior to the encrypted segment before we bother attempting to decrypt.
func validateSequence(packetLen int, prefixByte uint8, sequence uint64, readPacketKey, allowedPackets []byte, replayProtection *ReplayProtection) error {

	if readPacketKey == nil {
		return errors.New("empty packet key")
	}

	if packetLen < 1+1+MAC_BYTES {
		return errors.New("ignored encrypted packet. packet is too small to be valid")
	}

	packetType := prefixByte & 0xF
	if PacketType(packetType) >= ConnectionNumPackets {
		return errors.New("ignored encrypted packet. packet type " + packetTypeMap[PacketType(packetType)] + " is invalid")
	}

	if allowedPackets[packetType] == 0 {
		return errors.New("ignored encrypted packet. packet type " + packetTypeMap[PacketType(packetType)] + " is invalid")
	}

	// replay protection (optional)
	if replayProtection != nil && PacketType(packetType) >= ConnectionKeepAlive {
		if replayProtection.AlreadyReceived(sequence) == 1 {
			v := strconv.FormatUint(sequence, 10)
			return errors.New("ignored connection payload packet. sequence " + v + " already received (replay protection)")
		}
	}
	return nil
}

// write the prefix byte (this is a combination of the packet type and number of sequence bytes)
func writePacketPrefix(p Packet, buffer *[]byte, sequence uint64) (uint8, error) {
	sequenceBytes := sequenceNumberBytesRequired(sequence)
	if sequenceBytes < 1 || sequenceBytes > 8 {
		return 0, errors.New("invalid sequence bytes, must be between [1-8]")
	}

	prefixByte := uint8(p.GetType()) | uint8(sequenceBytes<<4)
	*buffer, _ = WriteUint8(*buffer, prefixByte)

	sequenceTemp := sequence

	var i uint8
	for ; i < sequenceBytes; i += 1 {
		*buffer, _ = WriteUint8(*buffer, uint8(sequenceTemp&0xFF))
		sequenceTemp >>= 8
	}
	return prefixByte, nil
}

// Encrypts the packet data of the supplied buffer between encryptedStart and encrypedFinish.
func encryptPacket(buffer []byte, encryptedStart, encryptedFinish int, prefixByte uint8, protocolId, sequence uint64, writePacketKey []byte) (int, error) {
	// slice up the buffer for the bits we will encrypt
	encryptedBuffer := buffer[encryptedStart:encryptedFinish]
	additionalData, nonce := packetCryptData(prefixByte, protocolId, sequence)
	if err := EncryptAead(&encryptedBuffer, additionalData, nonce, writePacketKey); err != nil {
		return -1, err
	}

	buffer, _ = WriteBytes(buffer, encryptedBuffer)
	return len(encryptedBuffer), nil
}

// used for encrypting the per-packet packet written with the prefix byte, protocol id and version as the associated data. this must match to decrypt.
func packetCryptData(prefixByte uint8, protocolId, sequence uint64) ([]byte, []byte) {
	additionalData := make([]byte, VERSION_INFO_BYTES+8+1)
	start := additionalData

	additionalData, _ = WriteBytesN(additionalData, VERSION_INFO, VERSION_INFO_BYTES)
	additionalData, _ = WriteUint64(additionalData, protocolId)
	additionalData, _ = WriteUint8(additionalData, prefixByte)

	nonce := make([]byte, SizeUint64)
	WriteUint64(nonce, sequence)
	return start, nonce
}

// Depending on size of sequence number, we need to reserve N bytes
func sequenceNumberBytesRequired(sequence uint64) uint8 {
	var mask uint64
	mask = 0xFF00000000000000
	var i uint8
	for ; i < 7; i += 1 {
		if sequence&mask != 0 {
			break
		}
		mask >>= 8
	}
	return 8 - i
}
