package ztnet

import (
	"encoding/hex"
	"fmt"
	"net"
)

// RFC4193 computes the RFC 4193 IPv6 address for a ZeroTier node.
func RFC4193(networkID, nodeID string) (net.IP, error) {
	nw, id, err := validateIDs(networkID, nodeID)
	if err != nil {
		return nil, err
	}

	ip := make(net.IP, net.IPv6len)
	ip[0] = 0xfd
	copy(ip[1:8], nw[:7])
	ip[8] = nw[7]
	ip[9] = 0x99
	ip[10] = 0x93
	copy(ip[11:13], id[0:2])
	copy(ip[13:15], id[2:4])
	ip[15] = id[4]
	return ip, nil
}

// SixPlane computes the 6PLANE IPv6 address for a ZeroTier node.
func SixPlane(networkID, nodeID string) (net.IP, error) {
	nw, id, err := validateIDs(networkID, nodeID)
	if err != nil {
		return nil, err
	}
	top := uint32(nw[0])<<24 | uint32(nw[1])<<16 | uint32(nw[2])<<8 | uint32(nw[3])
	bot := uint32(nw[4])<<24 | uint32(nw[5])<<16 | uint32(nw[6])<<8 | uint32(nw[7])
	hashed := top ^ bot

	ip := make(net.IP, net.IPv6len)
	ip[0] = 0xfc
	ip[1] = byte((hashed >> 24) & 0xff)
	ip[2] = byte((hashed >> 16) & 0xff)
	ip[3] = byte((hashed >> 8) & 0xff)
	ip[4] = byte(hashed & 0xff)
	ip[5] = id[0]
	copy(ip[6:8], id[1:3])
	copy(ip[8:10], id[3:5])
	ip[15] = 0x01
	return ip, nil
}

func validateIDs(networkID, nodeID string) ([]byte, []byte, error) {
	if len(networkID) != 16 {
		return nil, nil, fmt.Errorf("ztnet: ipv6: invalid networkID length %d", len(networkID))
	}
	if len(nodeID) != 10 {
		return nil, nil, fmt.Errorf("ztnet: ipv6: invalid nodeID length %d", len(nodeID))
	}

	nw := make([]byte, 8)
	if _, err := hex.Decode(nw, []byte(networkID)); err != nil {
		return nil, nil, fmt.Errorf("ztnet: ipv6: invalid networkID hex: %w", err)
	}
	id := make([]byte, 5)
	if _, err := hex.Decode(id, []byte(nodeID)); err != nil {
		return nil, nil, fmt.Errorf("ztnet: ipv6: invalid nodeID hex: %w", err)
	}
	return nw, id, nil
}
