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
