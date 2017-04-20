package netcode

import (
	"bytes"
	"net"
)

type connectTokenEntry struct {
	mac     []byte
	address *net.UDPAddr
	time    float64
}

type encryptionEntry struct {
	expireTime float64
	lastAccess float64
	address    *net.UDPAddr
	sendKey    []byte
	recvKey    []byte
}

type tokenRequest struct {
	serverTime float64
	entry      *connectTokenEntry
	responseCh chan bool
}

func newTokenRequest(addr *net.UDPAddr, mac []byte, time float64) *tokenRequest {
	r := &tokenRequest{address: addr, mac: mac, time: time}
	r.responseCh = make(chan bool)
	return r
}

type EncryptionManager struct {
	maxClients int
	maxEntries int

	encryptionEntries map[*net.UDPAddr]*encryptionEntry
	tokenEntries      map[[]byte]*connectTokenEntry
	timeout           float64

	closeCh      chan struct{}
	tokenCh      chan *tokenRequest
	encryptionCh chan *encryptionRequest

	emptyMac      []byte // used to ensure empty mac (all empty bytes) doesn't match
	emptyWriteKey []byte // used to test for empty write key
}

func NewEncryptionManager(timeout float64, maxClients int) *EncryptionManager {
	m := &EncryptionManager{}
	m.maxClients = maxClients
	m.maxEntries = maxClients * 8
	m.timeout = timeout

	m.encryptionEntries = make(map[*net.UDPAddr]*encryptionEntry)
	m.tokenEntries = make(map[*net.UDPAddr]*connectTokenEntry)

	m.emptyMac = make([]byte, MAC_BYTES)
	m.emptyWriteKey = make([]byte, KEY_BYTES)
	return m
}

func (m *EncryptionManager) listenEvents() {
	for {
		select {
		case <-m.closeCh:
			return
		case token := <-m.tokenCh:
			ret := m.findOrAddToken(token.entry)
			token.responseCh <- ret
		case crypt := <-m.encryptionCh:
		}
	}
}

func (m *EncryptionManager) findOrAddToken(token *connectTokenEntry, serverTime float64) bool {
	var entry *connectTokenEntry

	if bytes.Equal(token.mac, m.emptyMac) {
		return false
	}

	if entry, ok := m.tokenEntries[token.mac]
}
