package netcode

import (
	"errors"
	//"log"
	"net"
)

// The private parts of a connect token
type ConnectTokenPrivate struct {
	sharedTokenData        // holds the server addresses, client <-> server keys
	ClientId        uint64 // id for this token
	UserData        []byte // used to store user data
	mac             []byte // used to store the message authentication code after encryption/before decryption
	TokenData       []byte // used to store the serialized/encrypted buffer
}

// Create a new connect token private with an empty TokenData buffer
func NewConnectTokenPrivate(clientId uint64, serverAddrs []net.UDPAddr, userData []byte) *ConnectTokenPrivate {
	p := &ConnectTokenPrivate{}
	p.TokenData = make([]byte, CONNECT_TOKEN_PRIVATE_BYTES-MAC_BYTES) // mac will be appended by EncryptAead
	p.ClientId = clientId
	p.UserData = userData
	p.ServerAddrs = serverAddrs
	p.mac = make([]byte, MAC_BYTES)
	return p
}

func NewEmptyConnectTokenPrivate() *ConnectTokenPrivate {
	p := &ConnectTokenPrivate{}
	p.TokenData = make([]byte, CONNECT_TOKEN_PRIVATE_BYTES)
	p.mac = make([]byte, MAC_BYTES)
	return p
}

func (p *ConnectTokenPrivate) Generate() error {
	return p.GenerateShared()
}

// Create a new connect token private with an pre-set, encrypted buffer
// Caller is expected to call Decrypt() and Read() to set the instances properties
func NewConnectTokenPrivateEncrypted(buffer []byte) *ConnectTokenPrivate {
	p := &ConnectTokenPrivate{}
	p.TokenData = make([]byte, CONNECT_TOKEN_PRIVATE_BYTES) // need to allocate for mac
	p.mac = make([]byte, MAC_BYTES)
	copy(p.TokenData, buffer[:CONNECT_TOKEN_PRIVATE_BYTES])
	copy(p.mac, buffer[CONNECT_TOKEN_PRIVATE_BYTES-MAC_BYTES:])
	return p
}

// Returns the message authentication code for the encrypted buffer
// by splicing the token data, returns an empty byte slice if the tokendata
// buffer is empty/less than MAC_BYTES
func (p *ConnectTokenPrivate) Mac() []byte {
	return p.mac
}

// Reads the token properties from the internal TokenData buffer.
func (p *ConnectTokenPrivate) Read() error {
	var err error
	start := p.TokenData

	if p.TokenData, err = ReadUint64(p.TokenData, &p.ClientId); err != nil {
		return err
	}

	if err = p.ReadShared(p.TokenData); err != nil {
		return err
	}

	if p.TokenData, err = ReadBytes(p.TokenData, &p.UserData, USER_DATA_BYTES); err != nil {
		return errors.New("error reading user data")
	}
	p.TokenData = start
	return nil
}

// Writes the token data to our TokenData buffer and alternatively returns the buffer to caller.
func (p *ConnectTokenPrivate) Write() ([]byte, error) {
	var err error
	start := p.TokenData

	if p.TokenData, err = WriteUint64(p.TokenData, p.ClientId); err != nil {
		return nil, err
	}

	if err := p.WriteShared(&p.TokenData); err != nil {
		return nil, err
	}

	if p.TokenData, err = WriteBytesN(p.TokenData, p.UserData, USER_DATA_BYTES); err != nil {
		return nil, err
	}

	p.TokenData = start
	return p.TokenData, nil
}

// Encrypts, in place, the TokenData buffer, assumes Write() has already been called.
func (token *ConnectTokenPrivate) Encrypt(protocolId, expireTimestamp, sequence uint64, privateKey []byte) error {
	additionalData, nonce := buildTokenCryptData(protocolId, expireTimestamp, sequence)
	if err := EncryptAead(&token.TokenData, additionalData, nonce, privateKey); err != nil {
		return err
	}
	if len(token.TokenData) != CONNECT_TOKEN_PRIVATE_BYTES {
		return errors.New("invalid token private byte size")
	}

	copy(token.mac, token.TokenData[CONNECT_TOKEN_PRIVATE_BYTES-MAC_BYTES:])
	return nil
}

// Decrypts the internal TokenData buffer, assumes that TokenData has been populated with the encrypted data
// (most likely via NewConnectTokenPrivateEncrypted(...)). Optionally returns the decrypted buffer to caller.
func (p *ConnectTokenPrivate) Decrypt(protocolId, expireTimestamp, sequence uint64, privateKey []byte) ([]byte, error) {
	var err error

	if len(p.TokenData) != CONNECT_TOKEN_PRIVATE_BYTES {
		return nil, errors.New("invalid token private byte size")
	}

	copy(p.mac, p.TokenData[CONNECT_TOKEN_PRIVATE_BYTES-MAC_BYTES:])
	additionalData, nonce := buildTokenCryptData(protocolId, expireTimestamp, sequence)
	if p.TokenData, err = DecryptAead(p.TokenData, additionalData, nonce, privateKey); err != nil {
		return nil, err
	}

	return p.TokenData, nil
}

// Builds the additional data and nonce necessary for encryption and decryption.
func buildTokenCryptData(protocolId, expireTimestamp, sequence uint64) ([]byte, []byte) {
	additionalData := make([]byte, VERSION_INFO_BYTES+8+8)
	start := additionalData

	additionalData, _ = WriteBytes(additionalData, VERSION_INFO)
	additionalData, _ = WriteUint64(additionalData, protocolId)
	additionalData, _ = WriteUint64(additionalData, expireTimestamp)

	nonce := make([]byte, SizeUint64)
	WriteUint64(nonce, sequence)

	return start, nonce
}
