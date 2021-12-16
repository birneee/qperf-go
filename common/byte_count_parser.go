package common

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// ParseByteCountWithUnit supports the following unit suffixes:
// 10^3 based: kb, mb, gb, tb, pb .
// 2^10 based: kib, mib, gib, tib, pib .
// the unit suffix is case-insensitive.
// no unit suffix will result in normal integer parsing.
func ParseByteCountWithUnit(s string) (uint64, error) {
	expr := regexp.MustCompile("(\\d+)\\s*(\\w*)")
	match := expr.FindStringSubmatch(s)
	if len(match) != 3 {
		return 0, errors.New("failed to parse")
	}
	number, err := strconv.ParseUint(match[1], 10, 64)
	if err != nil {
		return 0, err
	}
	suffix := strings.TrimSpace(strings.ToLower(match[2]))
	switch suffix {
	case "":
		return number, nil
	case "b":
		return number, nil
	case "kb":
		return number * 1e3, nil
	case "mb":
		return number * 1e6, nil
	case "gb":
		return number * 1e9, nil
	case "tb":
		return number * 1e12, nil
	case "pb":
		return number * 1e15, nil
	case "kib":
		return number * (1 << 10), nil
	case "mib":
		return number * (1 << 20), nil
	case "gib":
		return number * (1 << 30), nil
	case "tib":
		return number * (1 << 40), nil
	case "pib":
		return number * (1 << 50), nil
	default:
		return 0, errors.New("invalid suffix")
	}
}
