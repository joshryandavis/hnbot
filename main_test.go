package main

import (
	"testing"
)

func TestNormalizeRedditURL(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Full Reddit URL with share parameters",
			input:    "https://www.reddit.com/r/degoogle/comments/1mau7yl/eu_age_verification_app_to_ban_any_android_system/?share_id=iR05aexja3cz3w-ITsqz1&utm_content=2&utm_medium=android_app&utm_name=androidcss&utm_source=share&utm_term=1",
			expected: "1mau7yl",
		},
		{
			name:     "Reddit URL without parameters",
			input:    "https://www.reddit.com/r/degoogle/comments/1mau7yl/eu_age_verification_app_to_ban_any_android_system/",
			expected: "1mau7yl",
		},
		{
			name:     "Reddit URL without www",
			input:    "https://reddit.com/r/hackernews/comments/1mbdi2k/some_title",
			expected: "1mbdi2k",
		},
		{
			name:     "Old Reddit URL",
			input:    "https://old.reddit.com/r/hackernews/comments/1mbdi2k/",
			expected: "1mbdi2k",
		},
		{
			name:     "Short Reddit URL",
			input:    "https://redd.it/1mbdi2k",
			expected: "1mbdi2k",
		},
		{
			name:     "Shared link format",
			input:    "https://www.reddit.com/r/degoogle/s/YxmPgFes8a",
			expected: "https://www.reddit.com/r/degoogle/s/YxmPgFes8a",
		},
		{
			name:     "Non-Reddit URL",
			input:    "https://example.com/some-article",
			expected: "https://example.com/some-article",
		},
		{
			name:     "Empty URL",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeRedditURL(tc.input)
			if result != tc.expected {
				t.Errorf("normalizeRedditURL(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestNormalizeURL(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "HTTPS with www",
			input:    "https://www.thingino.com",
			expected: "https://thingino.com/",
		},
		{
			name:     "HTTP without www",
			input:    "http://thingino.com",
			expected: "https://thingino.com/",
		},
		{
			name:     "With trailing slash",
			input:    "https://thingino.com/",
			expected: "https://thingino.com/",
		},
		{
			name:     "With path no trailing slash",
			input:    "https://thingino.com/path",
			expected: "https://thingino.com/path",
		},
		{
			name:     "With path and trailing slash",
			input:    "https://thingino.com/path/",
			expected: "https://thingino.com/path",
		},
		{
			name:     "With query parameters",
			input:    "https://www.thingino.com/page?utm_source=test",
			expected: "https://thingino.com/page?utm_source=test",
		},
		{
			name:     "With fragment",
			input:    "https://thingino.com/#section",
			expected: "https://thingino.com/",
		},
		{
			name:     "Mixed case domain",
			input:    "https://www.ThinGino.COM/",
			expected: "https://thingino.com/",
		},
		{
			name:     "Reddit URL uses Reddit normalization",
			input:    "https://www.reddit.com/r/test/comments/123abc/title/",
			expected: "123abc",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeURL(tc.input)
			if result != tc.expected {
				t.Errorf("normalizeURL(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestDuplicateDetection(t *testing.T) {
	// Simulate existing posts map
	existingLinks := make(map[string]bool)

	// Add a Reddit post with the normalized ID
	redditURL := "https://www.reddit.com/r/degoogle/comments/1mau7yl/eu_age_verification_app_to_ban_any_android_system/"
	normalizedID := normalizeRedditURL(redditURL)
	existingLinks[normalizedID] = true

	// Test various forms of the same URL - all should be detected as duplicates
	duplicateURLs := []string{
		"https://www.reddit.com/r/degoogle/comments/1mau7yl/eu_age_verification_app_to_ban_any_android_system/?share_id=iR05aexja3cz3w-ITsqz1&utm_content=2",
		"https://reddit.com/r/degoogle/comments/1mau7yl/different_title_here",
		"https://old.reddit.com/r/degoogle/comments/1mau7yl/",
		"https://www.reddit.com/r/degoogle/comments/1mau7yl/eu_age_verification_app_to_ban_any_android_system/",
	}

	for _, url := range duplicateURLs {
		t.Run("Duplicate detection for "+url, func(t *testing.T) {
			linkKey := normalizeRedditURL(url)
			if !existingLinks[linkKey] {
				t.Errorf("URL %q should be detected as duplicate but wasn't. Normalized to: %q", url, linkKey)
			}
		})
	}

	// Test a different Reddit post - should NOT be detected as duplicate
	differentURL := "https://www.reddit.com/r/hackernews/comments/1mbdi2k/different_post/"
	t.Run("Different post detection", func(t *testing.T) {
		linkKey := normalizeRedditURL(differentURL)
		if existingLinks[linkKey] {
			t.Errorf("URL %q should NOT be detected as duplicate but was. Normalized to: %q", differentURL, linkKey)
		}
	})
}
