package util

import (
	"bytes"
	"encoding/json"
)

func JsonEncoderHTML(v interface{}) ([]byte, error) {
	bf := bytes.NewBuffer([]byte{})
	jsonEncoder := json.NewEncoder(bf)
	jsonEncoder.SetEscapeHTML(false)
	err := jsonEncoder.Encode(v)
	if err != nil {
		return []byte{}, nil
	}
	return bf.Bytes(), nil
}
