package opengraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsBotRequest_KnownBots(t *testing.T) {
	tests := map[string]string{
		"Facebook":  "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)",
		"Facebot":   "Facebot",
		"Twitter":   "Twitterbot/1.0",
		"Slack":     "Slackbot-LinkExpanding 1.0 (+https://api.slack.com/robots)",
		"LinkedIn":  "LinkedInBot/1.0 (compatible; Mozilla/5.0; Apache-HttpClient +http://www.linkedin.com)",
		"WhatsApp":  "WhatsApp/2.19.81 A",
		"Telegram":  "TelegramBot (like TwitterBot)",
		"Discord":   "Mozilla/5.0 (compatible; Discordbot/2.0; +https://discordapp.com)",
		"Apple":     "Applebot/0.1 (+http://www.apple.com/go/applebot)",
		"Pinterest": "Pinterestbot/1.0",
		"Google":    "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
		"Bing":      "Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)",
		"Reddit":    "redditbot/1.0",
		"Skype":     "SkypeUriPreview Preview/0.5",
	}

	for name, ua := range tests {
		t.Run(name, func(t *testing.T) {
			assert.True(t, IsBotRequest(ua), "expected %s to be detected as bot", name)
		})
	}
}

func TestIsBotRequest_RegularBrowsers(t *testing.T) {
	tests := map[string]string{
		"Chrome":  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Firefox": "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
		"Safari":  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
		"Edge":    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Edg/91.0.864.59",
		"curl":    "curl/7.68.0",
		"Empty":   "",
	}

	for name, ua := range tests {
		t.Run(name, func(t *testing.T) {
			assert.False(t, IsBotRequest(ua), "expected %s to NOT be detected as bot", name)
		})
	}
}

func TestIsBotRequest_CaseInsensitive(t *testing.T) {
	assert.True(t, IsBotRequest("FACEBOOKEXTERNALHIT/1.1"))
	assert.True(t, IsBotRequest("twitterbot/1.0"))
	assert.True(t, IsBotRequest("SLACKBOT-LinkExpanding"))
}