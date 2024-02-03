package token

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGenerateToken(t *testing.T) {
	tokenGen := &base64TokenGenerator{}

	tests := map[string]struct {
		id            int
		expectedToken string
	}{
		"less than 10 digit length ID": {
			id:            1,
			expectedToken: "1",
		},
		"10 digit length ID": {
			id:            1234567890,
			expectedToken: "1LY7VK",
		},
		"more than 10 digit length ID": {
			id:            12345678901,
			expectedToken: "8M0kX1",
		},
		"more than 10 digit length ID all zero": {
			id:            100000000000,
			expectedToken: "6laZE",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expectedToken, tokenGen.GetToken(test.id))
		})
	}
}

func TestKnownIssues(t *testing.T) {
	tokenGen := &base64TokenGenerator{}

	tests := map[string]struct {
		firstID       int
		secondID      int
		expectedToken string
	}{
		"same token for two different ID with zero longer than 10 digit": {
			firstID:       100000000000,
			secondID:      10000000000000,
			expectedToken: "6laZE",
		},
		"same token for two different ID with one at the end longer than 10 digit": {
			firstID:       10000000000001,
			secondID:      10000000000000001,
			expectedToken: "6laZE1",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expectedToken, tokenGen.GetToken(test.firstID))
			assert.Equal(t, test.expectedToken, tokenGen.GetToken(test.secondID))
		})
	}
}
