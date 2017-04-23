package netcode

import (
	"errors"
)

// Challenge tokens are used in certain packet types
type ChallengeToken struct {
	ClientId  uint64 // the clientId associated with this token
	UserData  []byte // the userdata payload
	TokenData []byte // the serialized payload container
}

// Creates a new empty challenge token with only the clientId set
func NewChallengeToken(clientId uint64) *ChallengeToken {
	token := &ChallengeToken{}
	token.ClientId = clientId
	token.TokenData = make([]byte, CHALLENGE_TOKEN_BYTES-MAC_BYTES) // mac bytes will be appended by EncryptAead
	token.UserData = make([]byte, USER_DATA_BYTES)
	return token
}

// Serializes the client id and userData, also sets the UserData buffer.
func (t *ChallengeToken) Write(userData []byte) []byte {
	copy(t.UserData, userData)
	ref := t.TokenData
	t.TokenData, _ = WriteUint64(t.TokenData, t.ClientId)
	t.TokenData, _ = WriteBytes(t.TokenData, userData)
	return ref
}

// Encrypts the TokenData buffer with the sequence nonce and provided key
func EncryptChallengeToken(tokenBuffer *[]byte, sequence uint64, key []byte) error {
	nonce := make([]byte, SizeUint64)
	WriteUint64(nonce, sequence)
	return EncryptAead(tokenBuffer, nil, nonce, key)
}

// Decrypts the TokenData buffer with the sequence nonce and provided key, updating the
// internal TokenData buffer
func DecryptChallengeToken(tokenBuffer []byte, sequence uint64, key []byte) ([]byte, error) {
	nonce := make([]byte, SizeUint64)
	WriteUint64(nonce, sequence)
	return DecryptAead(tokenBuffer, nil, nonce, key)
}

// Generates a new ChallengeToken from the provided buffer byte slice. Only sets the ClientId
// and UserData buffer.
func ReadChallengeToken(buffer []byte) (*ChallengeToken, error) {
	var err error
	var clientId uint64

	tokenBuffer := buffer
	if tokenBuffer, err = ReadUint64(tokenBuffer, &clientId); err != nil {
		return nil, errors.New("error reading clientId: " + err.Error())
	}

	token := NewChallengeToken(clientId)
	if _, err = ReadBytes(tokenBuffer, &token.UserData, USER_DATA_BYTES); err != nil {
		return nil, errors.New("error reading user data: " + err.Error())
	}
	return token, nil
}
