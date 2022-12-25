package modbus

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cast"
)

func castStartingAddress(i interface{}) (uint16, error) {
	res, err := cast.ToUint16E(i)
	if err != nil {
		return 0, errors.New("startingAddress should be castable to an integer value")
	}

	return res, nil
}

func normalizeRawType(rawType string) (normalized string, err error) {
	switch strings.ToUpper(rawType) {
	case UINT16:
		normalized = "Uint16"
	case INT16:
		normalized = "Int16"
	default:
		return "", fmt.Errorf("the raw type %s is not supported", rawType)
	}
	return normalized, err
}

func castSwapAttribute(i interface{}) (bool, error) {
	res, err := cast.ToBoolE(i)
	if err != nil {
		return res, errors.New("swap attribute should be castable to a boolean value")
	}

	return res, nil
}
