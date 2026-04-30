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

// GetToken chunks the integer ID because the base62 library has a fixed MaxTokenLength.
// Large IDs that exceed that length are split, encoded separately, and concatenated.
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
