package common

import (
	"fmt"
	"regexp"
)

// AppendPortIfNotSpecified parses network addresses in the form "host:port" or only "host".
// if port is not specified use the defaultPort.
func AppendPortIfNotSpecified(address string, port int) string {
	hasPort, _ := regexp.MatchString("^.*:\\d+$", address)
	if hasPort {
		return address
	}
	return fmt.Sprintf("%s:%d", address, port)
}

// GetHost returns host of string and remove port if present;
// example: GetHost("localhost") = "localhost"
// example: GetHost("localhost:80") = "localhost"
func GetHost(address string) string {
	r := regexp.MustCompile("^(.*):\\d+?$")
	groups := r.FindStringSubmatch(address)
	if len(groups) == 2 {
		return groups[1]
	}
	return address
}
