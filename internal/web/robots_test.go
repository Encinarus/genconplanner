package web

import (
	"os"
	"strings"
	"testing"
)

func TestRobotsTxtBlocksAIBots(t *testing.T) {
	// Look for static/robots.txt from the perspective of internal/web package (which is 2 levels deep)
	content, err := os.ReadFile("../../static/robots.txt")
	if err != nil {
		t.Fatalf("failed to read static/robots.txt: %v", err)
	}

	robotsTxt := string(content)

	expectedBots := []string{
		"GPTBot",
		"ChatGPT-User",
		"OAI-SearchBot",
		"ClaudeBot",
		"Claude-Web",
		"anthropic-ai",
		"Claude-SearchBot",
		"Claude-User",
		"Google-Extended",
		"CCBot",
		"Applebot-Extended",
		"PerplexityBot",
		"Perplexity-User",
		"Meta-ExternalAgent",
		"Meta-ExternalFetcher",
		"FacebookBot",
		"Amazonbot",
		"Bytespider",
		"cohere-ai",
		"Omgilibot",
		"Omgili",
		"Diffbot",
		"YouBot",
	}

	for _, bot := range expectedBots {
		if !strings.Contains(robotsTxt, "User-agent: "+bot) {
			t.Errorf("robots.txt is missing User-agent: %s", bot)
		}
	}

	if !strings.Contains(robotsTxt, "Disallow: /") {
		t.Errorf("robots.txt is missing Disallow: / directive")
	}
}
