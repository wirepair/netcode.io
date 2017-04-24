package netcode

import (
	"bytes"
	"errors"
	//"log"
	"net"
	"time"
)

// ip types used in serialization of server addresses
const (
	ADDRESS_NONE = iota
	ADDRESS_IPV4
	ADDRESS_IPV6
)

// number of bytes for connect tokens
const CONNECT_TOKEN_BYTES = 2048

// Token used for connecting
type ConnectToken struct {
	sharedTokenData                      // a shared container holding the server addresses, client and server keys
	VersionInfo     []byte               // the version information for client <-> server communications
	ProtocolId      uint64               // protocol id for communications
	CreateTimestamp uint64               // when this token was created
	ExpireTimestamp uint64               // when this token expires
	Sequence        uint64               // the sequence id
	PrivateData     *ConnectTokenPrivate // reference to the private parts of this connect token
	TimeoutSeconds  uint32               // timeout of connect token in seconds
}

// Create a new empty token and empty private token
func NewConnectToken() *ConnectToken {
	token := &ConnectToken{}
	token.PrivateData = NewEmptyConnectTokenPrivate()
	token.VersionInfo = make([]byte, VERSION_INFO_BYTES)
	return token
}

// Generates the token and private token data with the supplied config values and sequence id.
// This will also write and encrypt the private token
func (token *ConnectToken) Generate(clientId uint64, serverAddrs []net.UDPAddr, versionInfo []byte, protocolId uint64, tokenExpiry uint64, timeoutSeconds uint32, sequence uint64, userData, privateKey []byte) error {
	token.CreateTimestamp = uint64(time.Now().Unix())
	token.ExpireTimestamp = token.CreateTimestamp + tokenExpiry
	token.VersionInfo = VERSION_INFO
	token.ProtocolId = protocolId
	token.TimeoutSeconds = timeoutSeconds
	token.Sequence = sequence

	token.PrivateData = NewConnectTokenPrivate(clientId, serverAddrs, userData)
	if err := token.PrivateData.Generate(); err != nil {
		return err
	}

	token.ClientKey = token.PrivateData.ClientKey
	token.ServerKey = token.PrivateData.ServerKey
	token.ServerAddrs = serverAddrs

	if _, err := token.PrivateData.Write(); err != nil {
		return err
	}

	if err := token.PrivateData.Encrypt(token.ProtocolId, token.ExpireTimestamp, sequence, privateKey); err != nil {
		return err
	}

	return nil
}

// Writes the ConnectToken and previously encrypted ConnectTokenPrivate data to a byte slice
func (token *ConnectToken) Write() ([]byte, error) {
	buffer := make([]byte, CONNECT_TOKEN_BYTES)
	start := buffer
	buffer, _ = WriteBytes(buffer, token.VersionInfo)
	buffer, _ = WriteUint64(buffer, token.ProtocolId)
	buffer, _ = WriteUint64(buffer, token.CreateTimestamp)
	buffer, _ = WriteUint64(buffer, token.ExpireTimestamp)
	buffer, _ = WriteUint64(buffer, token.Sequence)

	// assumes private token has already been encrypted
	buffer, _ = WriteBytes(buffer, token.PrivateData.TokenData)

	// writes server/client key and addresses to public part of the buffer
	if err := token.WriteShared(&buffer); err != nil {
		return nil, err
	}

	buffer, _ = WriteUint32(buffer, token.TimeoutSeconds)
	return start, nil
}

// Takes in a slice of decrypted connect token bytes and generates a new ConnectToken.
// Note that the ConnectTokenPrivate is still encrypted at this point.
func ReadConnectToken(tokenBuffer []byte) (*ConnectToken, error) {
	var err error

	token := NewConnectToken()

	if tokenBuffer, err = ReadBytes(tokenBuffer, &token.VersionInfo, VERSION_INFO_BYTES); err != nil {
		return nil, errors.New("read connect token data has bad version info " + err.Error())
	}

	if !bytes.Equal(VERSION_INFO, token.VersionInfo) {
		return nil, errors.New("read connect token data has bad version info: " + string(token.VersionInfo))
	}

	if tokenBuffer, err = ReadUint64(tokenBuffer, &token.ProtocolId); err != nil {
		return nil, errors.New("read connect token data has bad protocol id " + err.Error())
	}

	if tokenBuffer, err = ReadUint64(tokenBuffer, &token.CreateTimestamp); err != nil {
		return nil, errors.New("read connect token data has bad create timestamp " + err.Error())
	}

	if tokenBuffer, err = ReadUint64(tokenBuffer, &token.ExpireTimestamp); err != nil {
		return nil, errors.New("read connect token data has bad expire timestamp " + err.Error())
	}

	if token.CreateTimestamp > token.ExpireTimestamp {
		return nil, errors.New("expire timestamp is > create timestamp")
	}

	if tokenBuffer, err = ReadUint64(tokenBuffer, &token.Sequence); err != nil {
		return nil, errors.New("read connect data has bad sequence " + err.Error())
	}

	// it is still encrypted at this point.
	if tokenBuffer, err = ReadBytes(tokenBuffer, &token.PrivateData.TokenData, len(token.PrivateData.TokenData)); err != nil {
		return nil, errors.New("read connect data has bad private data " + err.Error())
	}

	// reads servers, client and server key
	if err = token.ReadShared(tokenBuffer); err != nil {
		return nil, err
	}

	if tokenBuffer, err = ReadUint32(tokenBuffer, &token.TimeoutSeconds); err != nil {
		return nil, err
	}

	return token, nil
}
