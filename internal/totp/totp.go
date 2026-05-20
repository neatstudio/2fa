package totp

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

type Options struct {
	Period int64
	Digits int
}

type Code struct {
	Value     string
	Remaining int64
}

func GenerateDefault(secret string, at time.Time) (Code, error) {
	return Generate(secret, at, Options{Period: 30, Digits: 6})
}

func Generate(secret string, at time.Time, options Options) (Code, error) {
	if options.Period <= 0 {
		options.Period = 30
	}
	if options.Digits <= 0 {
		options.Digits = 6
	}
	if options.Digits > 10 {
		return Code{}, errors.New("digits must be 10 or less")
	}

	key, err := decodeBase32Secret(secret)
	if err != nil {
		return Code{}, err
	}

	unix := at.Unix()
	counter := uint64(unix / options.Period)
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)

	mac := hmac.New(sha1.New, key)
	if _, err := mac.Write(msg[:]); err != nil {
		return Code{}, err
	}
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	binaryCode := (uint32(sum[offset])&0x7f)<<24 |
		(uint32(sum[offset+1])&0xff)<<16 |
		(uint32(sum[offset+2])&0xff)<<8 |
		(uint32(sum[offset+3]) & 0xff)

	modulo := uint32(math.Pow10(options.Digits))
	value := binaryCode % modulo
	remaining := options.Period - (unix % options.Period)
	if remaining == 0 {
		remaining = options.Period
	}

	return Code{
		Value:     fmt.Sprintf("%0*d", options.Digits, value),
		Remaining: remaining,
	}, nil
}

func decodeBase32Secret(secret string) ([]byte, error) {
	normalized := normalizeSecret(secret)
	if normalized == "" {
		return nil, errors.New("secret is required")
	}

	padding := len(normalized) % 8
	if padding != 0 {
		normalized += strings.Repeat("=", 8-padding)
	}

	decoded, err := base32.StdEncoding.DecodeString(normalized)
	if err != nil {
		return nil, fmt.Errorf("invalid base32 secret: %w", err)
	}
	return decoded, nil
}

func normalizeSecret(secret string) string {
	var b strings.Builder
	b.Grow(len(secret))
	for _, r := range secret {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		b.WriteRune(r)
	}
	return strings.ToUpper(strings.TrimRight(b.String(), "="))
}
