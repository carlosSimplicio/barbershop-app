package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

const (
	MinPasswordCodePoints = 8
	MaxPasswordCodePoints = 32
)

var (
	ErrInvalidPassword     = fmt.Errorf("password must contain %d to %d characters without whitespace, controls or invisible formatting characters", MinPasswordCodePoints, MaxPasswordCodePoints)
	ErrInvalidPasswordHash = errors.New("invalid password hash")
)

type PasswordParameters struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

func DefaultPasswordParameters() PasswordParameters {
	return PasswordParameters{
		MemoryKiB:   19 * 1024,
		Iterations:  2,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}
}

type PasswordHasher struct {
	parameters PasswordParameters
}

func NewPasswordHasher(parameters PasswordParameters) (*PasswordHasher, error) {
	if parameters.MemoryKiB < 19*1024 || parameters.MemoryKiB > 1024*1024 ||
		parameters.Iterations < 2 || parameters.Iterations > 10 ||
		parameters.Parallelism < 1 || parameters.Parallelism > 16 ||
		parameters.SaltLength < 16 || parameters.SaltLength > 64 ||
		parameters.KeyLength < 16 || parameters.KeyLength > 64 {
		return nil, errors.New("argon2id parameters are outside the supported range")
	}
	return &PasswordHasher{parameters: parameters}, nil
}

func ValidatePassword(password string) error {
	if !utf8.ValidString(password) {
		return ErrInvalidPassword
	}
	length := utf8.RuneCountInString(password)
	if length < MinPasswordCodePoints || length > MaxPasswordCodePoints {
		return ErrInvalidPassword
	}
	for _, character := range password {
		// Go exposes the components of Default_Ignorable_Code_Point separately.
		// Rejecting Cf in full also covers the exceptions to that derived property.
		if unicode.IsSpace(character) || unicode.IsControl(character) ||
			unicode.In(character, unicode.Cf, unicode.Other_Default_Ignorable_Code_Point, unicode.Variation_Selector) {
			return ErrInvalidPassword
		}
	}
	return nil
}

func (hasher *PasswordHasher) Hash(password string) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}

	salt := make([]byte, hasher.parameters.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, hasher.parameters.Iterations, hasher.parameters.MemoryKiB, hasher.parameters.Parallelism, hasher.parameters.KeyLength)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		hasher.parameters.MemoryKiB,
		hasher.parameters.Iterations,
		hasher.parameters.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func (hasher *PasswordHasher) Verify(password, encoded string) (bool, error) {
	parameters, salt, expected, err := parsePasswordHash(encoded)
	if err != nil {
		return false, err
	}
	actual := argon2.IDKey([]byte(password), salt, parameters.Iterations, parameters.MemoryKiB, parameters.Parallelism, parameters.KeyLength)
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

func (hasher *PasswordHasher) NeedsRehash(encoded string) (bool, error) {
	parameters, _, _, err := parsePasswordHash(encoded)
	if err != nil {
		return false, err
	}
	return parameters != hasher.parameters, nil
}

func parsePasswordHash(encoded string) (PasswordParameters, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" {
		return PasswordParameters{}, nil, nil, ErrInvalidPasswordHash
	}

	var memory, iterations uint64
	var parallelism uint64
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return PasswordParameters{}, nil, nil, ErrInvalidPasswordHash
	}
	if parts[3] != fmt.Sprintf("m=%d,t=%d,p=%d", memory, iterations, parallelism) {
		return PasswordParameters{}, nil, nil, ErrInvalidPasswordHash
	}
	if memory > uint64(^uint32(0)) || iterations > uint64(^uint32(0)) || parallelism > uint64(^uint8(0)) {
		return PasswordParameters{}, nil, nil, ErrInvalidPasswordHash
	}
	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return PasswordParameters{}, nil, nil, ErrInvalidPasswordHash
	}
	hash, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil {
		return PasswordParameters{}, nil, nil, ErrInvalidPasswordHash
	}

	parameters := PasswordParameters{
		MemoryKiB:   uint32(memory),
		Iterations:  uint32(iterations),
		Parallelism: uint8(parallelism),
		SaltLength:  uint32(len(salt)),
		KeyLength:   uint32(len(hash)),
	}
	if _, err := NewPasswordHasher(parameters); err != nil {
		return PasswordParameters{}, nil, nil, ErrInvalidPasswordHash
	}
	return parameters, salt, hash, nil
}
