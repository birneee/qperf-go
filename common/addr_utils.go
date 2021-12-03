package common

import (
	"net"
	"regexp"
	"strconv"
)

// ParseResolveHost parses network address and resolve ip.
// parses network addresses in the form "host:port" or only "host".
// if port is not specified use the defaultPort.
func ParseResolveHost(address string, defaultPort int) (*net.UDPAddr, error) {
	hasPort, _ := regexp.MatchString(".*:\\d+", address)

	var host string
	var port int
	if hasPort {
		var portString string
		var err error
		host, portString, err = net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		port, err = strconv.Atoi(portString)
		if err != nil {
			return nil, err
		}
	} else {
		host = address
		port = defaultPort
	}

	ip, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		return nil, err
	}

	return &net.UDPAddr{
		IP:   ip.IP,
		Port: port,
	}, nil
}
