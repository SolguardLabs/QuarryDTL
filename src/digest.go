package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func DigestJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return DigestString(fmt.Sprintf("%v", value))
	}
	return DigestBytes(encoded)
}

func DigestString(value string) string {
	return DigestBytes([]byte(value))
}

func DigestBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:16])
}

func DigestLedgerSeed(parts ...string) string {
	return DigestString(StableJoin(parts...))
}
