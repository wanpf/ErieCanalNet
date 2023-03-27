package helpers

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
)

var (
	bridgeIP     uint32
	bridgeIPAddr net.IP
)

// GetBridgeIP retrieves cni bridge veth's ipv4 addr
func GetBridgeIP() (net.IP, uint32, error) {
	if bridgeIP == 0 {
		found := false
		if ifaces, err := net.Interfaces(); err == nil {
			for _, iface := range ifaces {
				if iface.Flags&net.FlagUp != 0 && strings.HasPrefix(iface.Name, "cni0") {
					if addrs, addrErr := iface.Addrs(); addrErr == nil {
						for _, addr := range addrs {
							if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
								if bridgeIPAddr = ipnet.IP.To4(); bridgeIPAddr != nil {
									bridgeIP = binary.BigEndian.Uint32(bridgeIPAddr)
									found = true
									break
								}
							}
						}
					} else {
						return bridgeIPAddr, bridgeIP, fmt.Errorf("unexpected exit err: %v", err)
					}
					break
				}
			}
		} else {
			return bridgeIPAddr, bridgeIP, fmt.Errorf("unexpected exit err: %v", err)
		}
		if !found {
			return bridgeIPAddr, bridgeIP, fmt.Errorf("unexpected retrieves cni bridge veth[%s]'s ipv4 addr", "cni0")
		}
	}
	return bridgeIPAddr, bridgeIP, nil
}
