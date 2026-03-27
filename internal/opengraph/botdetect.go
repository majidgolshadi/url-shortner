package opengraph

import "strings"

// knownBotPatterns contains User-Agent substrings used by social media and
// messaging platform crawlers that look for Open Graph meta tags.
var knownBotPatterns = []string{
	"facebookexternalhit",
	"Facebot",
	"Twitterbot",
	"Slackbot",
	"LinkedInBot",
	"WhatsApp",
	"TelegramBot",
	"Discordbot",
	"Applebot",
	"Pinterestbot",
	"Googlebot",
	"bingbot",
	"Embedly",
	"Quora Link Preview",
	"Showyoubot",
	"outbrain",
	"vkShare",
	"Skype",
	"redditbot",
}

// IsBotRequest checks if the given User-Agent string belongs to a known
// social media or messaging platform crawler/bot.
func IsBotRequest(userAgent string) bool {
	ua := strings.ToLower(userAgent)
	for _, pattern := range knownBotPatterns {
		if strings.Contains(ua, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}