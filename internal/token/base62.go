package token

import (
	"strconv"
	"strings"

	"github.com/marksalpeter/token/v2"
)

type Generator interface {
	GetToken(id uint) string
}

// Base62TokenGenerator generates tokens using base62 encoding.
type Base62TokenGenerator struct{}

// GetToken divides the input integer ID into segments, each with a maximum length of 10, as per the max base62 token length.
// It returns the result as a concatenated string of generated tokens for each segment.
func (tg *Base62TokenGenerator) GetToken(id uint) string {
	strID := strconv.Itoa(int(id))
	var result strings.Builder

	for len(strID) > 0 {
		subIDStr := strID
		if len(strID) > token.MaxTokenLength {
			subIDStr = strID[:token.MaxTokenLength-1]
			strID = strID[token.MaxTokenLength:]
		} else {
			strID = ""
		}

		subID, _ := strconv.Atoi(subIDStr)
		result.WriteString(token.Token(subID).Encode())
	}

	return result.String()
}
