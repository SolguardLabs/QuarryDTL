package main

import (
	"encoding/json"
	"fmt"
	"io"
)

func WriteJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func DecodeJSON(data []byte, value any) error {
	if len(data) == 0 {
		return fmt.Errorf("empty json")
	}
	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}
	return nil
}

type ErrorPayload struct {
	Error string `json:"error"`
}
