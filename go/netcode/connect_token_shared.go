package netcode

import (
	"errors"
	"net"
	"strconv"
)

// This struct contains data that is shared in both public and private parts of the
// connect token.
type sharedTokenData struct {
	ServerAddrs []net.UDPAddr // list of server addresses this client may connect to
	ClientKey   []byte        // client to server key
	ServerKey   []byte        // server to client key
}

func (shared *sharedTokenData) GenerateShared() error {
	var err error

	if shared.ClientKey, err = GenerateKey(); err != nil {
		return errors.New("error generating client key: " + err.Error())
	}

	if shared.ServerKey, err = GenerateKey(); err != nil {
		return errors.New("error generating server key: " + err.Error())
	}
	return nil
}

// Reads and validates the servers, client <-> server keys.
func (shared *sharedTokenData) ReadShared(buffer []byte) error {
	var err error
	var servers uint32
	var ipBytes []byte
	var port uint16

	if buffer, err = ReadUint32(buffer, &servers); err != nil {
		return err
	}

	if servers <= 0 {
		return errors.New("empty servers")
	}

	if servers > MAX_SERVERS_PER_CONNECT {
		return errors.New("too many servers")
	}

	shared.ServerAddrs = make([]net.UDPAddr, servers)

	for i := 0; i < int(servers); i += 1 {
		var serverType uint8
		if buffer, err = ReadUint8(buffer, &serverType); err != nil {
			return err
		}

		if serverType == ADDRESS_IPV4 {
			ipBytes = make([]byte, 4)
			if buffer, err = ReadBytes(buffer, &ipBytes, len(ipBytes)); err != nil {
				return err
			}
		} else if serverType == ADDRESS_IPV6 {
			ipBytes = make([]byte, 16)
			for i := 0; i < 16; i += 2 {
				var n uint16
				if buffer, err = ReadUint16(buffer, &n); err != nil {
					return err
				}
				// decode little endian -> big endian for net.IP
				ipBytes[i] = byte(n) << 8
				ipBytes[i+1] = byte(n)
			}
		} else {
			return errors.New("unknown ip address")
		}

		ip := net.IP(ipBytes)

		if buffer, err = ReadUint16(buffer, &port); err != nil {
			return err
		}
		shared.ServerAddrs[i] = net.UDPAddr{IP: ip, Port: int(port)}
	}

	key := make([]byte, KEY_BYTES)
	if buffer, err = ReadBytes(buffer, &key, KEY_BYTES); err != nil {
		return err
	}
	copy(shared.ClientKey, key)

	if buffer, err = ReadBytes(buffer, &key, KEY_BYTES); err != nil {
		return err
	}
	copy(shared.ServerKey, key)
	return nil
}

// Writes the servers and client <-> server keys to the supplied buffer
func (shared *sharedTokenData) WriteShared(buffer []byte) error {

	serverLen := uint32(len(shared.ServerAddrs))
	buffer, _ = WriteUint32(buffer, serverLen)

	for _, addr := range shared.ServerAddrs {
		host, port, err := net.SplitHostPort(addr.String())
		if err != nil {
			return errors.New("invalid port for host: " + addr.String())
		}

		parsed := net.ParseIP(host)
		if parsed == nil {
			return errors.New("invalid ip address")
		}

		parsedIpv4 := parsed.To4()
		if parsedIpv4 != nil {
			buffer, _ = WriteUint8(buffer, uint8(ADDRESS_IPV4))

			for i := 0; i < len(parsedIpv4); i += 1 {
				buffer, _ = WriteUint8(buffer, parsedIpv4[i])
			}
		} else {
			buffer, _ = WriteUint8(buffer, uint8(ADDRESS_IPV6))
			for i := 0; i < len(parsed); i += 2 {
				var n uint16
				// net.IP is already big endian encoded, encode it to create little endian encoding.
				n = uint16(parsed[i]) << 8
				n = uint16(parsed[i+1])
				buffer, _ = WriteUint16(buffer, n)
			}
		}

		p, err := strconv.ParseUint(port, 10, 16)
		if err != nil {
			return err
		}

		buffer, _ = WriteUint16(buffer, uint16(p))
	}

	buffer, _ = WriteBytesN(buffer, shared.ClientKey, KEY_BYTES)
	buffer, _ = WriteBytesN(buffer, shared.ServerKey, KEY_BYTES)
	return nil
}
