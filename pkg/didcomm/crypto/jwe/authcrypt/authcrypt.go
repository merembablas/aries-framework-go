/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package authcrypt

import (
	"crypto/rand"
	"errors"

	chacha "golang.org/x/crypto/chacha20poly1305"
)

// This package deals with Authcrypt encryption for Packing/Unpacking DID Comm exchange
// Using Chacha20Poly1035 encryption/authentication

// ContentEncryption represents a content encryption algorithm.
type ContentEncryption string

// C20P Chacha20Poly1035 algorithm
const C20P = ContentEncryption("C20P") // Chacha20 encryption + Poly1035 authenticator cipher (96 bits nonce)

// XC20P XChacha20Poly1035 algorithm
const XC20P = ContentEncryption("XC20P") // XChacha20 encryption + Poly1035 authenticator cipher (192 bits nonce)

// randReader is a cryptographically secure random number generator.
// TODO: document usage for tests or find another mechanism.
//nolint:gochecknoglobals
var randReader = rand.Reader

// errEmptyRecipients is used when recipients list is empty
var errEmptyRecipients = errors.New("empty recipients")

// errInvalidKeypair is used when a keypair is invalid
var errInvalidKeypair = errors.New("invalid keypair")

// errRecipientNotFound is used when a recipient is not found
var errRecipientNotFound = errors.New("recipient not found")

// errUnsupportedAlg is used when a bad encryption algorithm is used
var errUnsupportedAlg = errors.New("algorithm not supported")

// Crypter represents an Authcrypt Encrypter (Decrypter) that outputs/reads JWE envelopes
type Crypter struct {
	alg       ContentEncryption
	nonceSize int
}

// Envelope represents a JWE envelope as per the Aries Encryption envelope specs
type Envelope struct {
	Protected  string      `json:"protected,omitempty"`
	Recipients []Recipient `json:"recipients,omitempty"`
	AAD        string      `json:"aad,omitempty"`
	IV         string      `json:"iv,omitempty"`
	Tag        string      `json:"tag,omitempty"`
	CipherText string      `json:"ciphertext,omitempty"`
}

// jweHeaders are the Protected JWE headers in a map format
type jweHeaders map[string]string

// Recipient is a recipient of an envelope including the shared encryption key
type Recipient struct {
	EncryptedKey string           `json:"encrypted_key,omitempty"`
	Header       RecipientHeaders `json:"header,omitempty"`
}

// RecipientHeaders are the recipient headers
type RecipientHeaders struct {
	APU string `json:"apu,omitempty"`
	IV  string `json:"iv,omitempty"`
	Tag string `json:"tag,omitempty"`
	KID string `json:"kid,omitempty"`
	OID string `json:"oid,omitempty"`
}

// New will create an encrypter instance to 'AuthCrypt' payloads for the given sender and recipients arguments
// and the encryption alg argument. Possible algorithms supported are:
// C20P (chacha20-poly1035 ietf)
// XC20P (xchacha20-poly1035 ietf)
// The returned crypter contains all the information required to encrypt payloads.
func New(alg ContentEncryption) (*Crypter, error) {
	var nonceSize int
	switch alg {
	case C20P:
		nonceSize = chacha.NonceSize
	case XC20P:
		nonceSize = chacha.NonceSizeX
	default:
		return nil, errUnsupportedAlg
	}

	c := &Crypter{
		alg,
		nonceSize,
	}

	return c, nil
}