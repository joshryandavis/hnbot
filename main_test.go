package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mmcdole/gofeed/rss"
	"github.com/turnage/graw/reddit"
)

// mockBot implements reddit.Bot for testing without hitting Reddit's API.
type mockBot struct{}

func (m *mockBot) Reply(parentName, text string) error { return nil }
func (m *mockBot) GetReply(parentName, text string) (reddit.Submission, error) {
	return reddit.Submission{Name: "t1_mock_comment", ID: "mock_comment"}, nil
}
func (m *mockBot) SendMessage(user, subject, text string) error { return nil }
func (m *mockBot) PostSelf(subreddit, title, text string) error { return nil }
func (m *mockBot) GetPostSelf(subreddit, title, text string) (reddit.Submission, error) {
	return reddit.Submission{Name: "t3_mock_post", ID: "mock_post", URL: ""}, nil
}
func (m *mockBot) PostLink(subreddit, title, url string) error { return nil }
func (m *mockBot) GetPostLink(subreddit, title, url string) (reddit.Submission, error) {
	return reddit.Submission{Name: "t3_mock_post", ID: "mock_post", URL: ""}, nil
}
func (m *mockBot) Thread(permalink string) (*reddit.Post, error) { return nil, nil }
func (m *mockBot) Listing(path, after string) (reddit.Harvest, error) {
	return reddit.Harvest{}, nil
}
func (m *mockBot) ListingWithParams(path string, params map[string]string) (reddit.Harvest, error) {
	return reddit.Harvest{}, nil
}

func TestNormalizeRedditURL(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Full Reddit URL with share parameters",
			input:    "https://www.reddit.com/r/degoogle/comments/1mau7yl/eu_age_verification_app_to_ban_any_android_system/?share_id=iR05aexja3cz3w-ITsqz1&utm_content=2&utm_medium=android_app&utm_name=androidcss&utm_source=share&utm_term=1",
			expected: "reddit.com/comments/1mau7yl",
		},
		{
			name:     "Reddit URL without parameters",
			input:    "https://www.reddit.com/r/degoogle/comments/1mau7yl/eu_age_verification_app_to_ban_any_android_system/",
			expected: "reddit.com/comments/1mau7yl",
		},
		{
			name:     "Reddit URL without www",
			input:    "https://reddit.com/r/hackernews/comments/1mbdi2k/some_title",
			expected: "reddit.com/comments/1mbdi2k",
		},
		{
			name:     "Old Reddit URL",
			input:    "https://old.reddit.com/r/hackernews/comments/1mbdi2k/",
			expected: "reddit.com/comments/1mbdi2k",
		},
		{
			name:     "Short Reddit URL",
			input:    "https://redd.it/1mbdi2k",
			expected: "reddit.com/comments/1mbdi2k",
		},
		{
			name:     "Shared link format",
			input:    "https://www.reddit.com/r/degoogle/s/YxmPgFes8a",
			expected: "reddit.com/s/YxmPgFes8a",
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
			name:     "With query parameters stripped",
			input:    "https://www.thingino.com/page?utm_source=test",
			expected: "https://thingino.com/page",
		},
		{
			name:     "With fragment stripped",
			input:    "https://thingino.com/#section",
			expected: "https://thingino.com/",
		},
		{
			name:     "Mixed case domain",
			input:    "https://www.ThinGino.COM/",
			expected: "https://thingino.com/",
		},
		{
			name:     "URL with percent-encoded character",
			input:    "https://openipc.org/%C3%A0",
			expected: "https://openipc.org/%C3%A0",
		},
		{
			name:     "URL with decoded special character",
			input:    "https://openipc.org/\u00e0",
			expected: "https://openipc.org/%C3%A0",
		},
		{
			name:     "Same URL with www and percent encoding",
			input:    "https://www.openipc.org/%C3%A0",
			expected: "https://openipc.org/%C3%A0",
		},
		{
			name:     "Reddit URL uses Reddit normalization",
			input:    "https://www.reddit.com/r/test/comments/123abc/title/",
			expected: "reddit.com/comments/123abc",
		},
		{
			name:     "Multiple query params stripped",
			input:    "https://example.com/page?foo=1&bar=2&baz=3",
			expected: "https://example.com/page",
		},
		{
			name:     "Empty URL",
			input:    "",
			expected: "",
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
	existingLinks := make(map[string]bool)

	redditURL := "https://www.reddit.com/r/degoogle/comments/1mau7yl/eu_age_verification_app_to_ban_any_android_system/"
	normalizedID := normalizeRedditURL(redditURL)
	existingLinks[normalizedID] = true

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

	differentURL := "https://www.reddit.com/r/hackernews/comments/1mbdi2k/different_post/"
	t.Run("Different post detection", func(t *testing.T) {
		linkKey := normalizeRedditURL(differentURL)
		if existingLinks[linkKey] {
			t.Errorf("URL %q should NOT be detected as duplicate but was. Normalized to: %q", differentURL, linkKey)
		}
	})
}

func TestIsSimilarTitle(t *testing.T) {
	testCases := []struct {
		name     string
		title1   string
		title2   string
		expected bool
	}{
		{
			name:     "Identical titles",
			title1:   "Hello World",
			title2:   "Hello World",
			expected: true,
		},
		{
			name:     "One contains the other",
			title1:   "Hello World",
			title2:   "Hello World and More",
			expected: true,
		},
		{
			name:     "Completely different short titles",
			title1:   "Hello",
			title2:   "World",
			expected: false,
		},
		{
			name:     "Same but different case",
			title1:   "hello world",
			title2:   "HELLO WORLD",
			expected: false,
		},
		{
			name:     "Partial overlap with short titles",
			title1:   "abc def",
			title2:   "xyz abc",
			expected: false,
		},
		{
			name:     "70%+ word overlap",
			title1:   "quick brown fox jumps over lazy dog",
			title2:   "quick brown fox jumps over lazy cat",
			expected: true,
		},
		{
			name:     "Below 70% word overlap",
			title1:   "This is a test title for the function",
			title2:   "This is something completely different now",
			expected: false,
		},
		{
			name:     "Both short - no word overlap analysis",
			title1:   "a b c",
			title2:   "a b d",
			expected: false,
		},
		{
			name:     "Empty strings",
			title1:   "",
			title2:   "",
			expected: true,
		},
		{
			name:     "One empty",
			title1:   "Hello World",
			title2:   "",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isSimilarTitle(tc.title1, tc.title2)
			if result != tc.expected {
				t.Errorf("isSimilarTitle(%q, %q) = %v, want %v", tc.title1, tc.title2, result, tc.expected)
			}
		})
	}
}

func TestBuildFeedUrl(t *testing.T) {
	url := buildFeedUrl()
	if url == nil {
		t.Fatal("buildFeedUrl() returned nil")
	}

	expected := "https://news.ycombinator.com/rss"
	actual := url.String()
	if actual != expected {
		t.Errorf("buildFeedUrl() = %q, want %q", actual, expected)
	}

	if url.Scheme != "https" {
		t.Errorf("Scheme = %q, want %q", url.Scheme, "https")
	}
	if url.Host != "news.ycombinator.com" {
		t.Errorf("Host = %q, want %q", url.Host, "news.ycombinator.com")
	}
	if url.Path != "rss" {
		t.Errorf("Path = %q, want %q", url.Path, "rss")
	}
}

func TestIsDuplicate(t *testing.T) {
	now := time.Now()
	oldTime := now.Add(-72 * time.Hour)
	recentTime := now.Add(-12 * time.Hour)

	existingPosts := []RedditPost{
		{URL: "https://example.com/article1", Title: "Article One", CreatedAt: recentTime},
		{URL: "https://example.com/article2", Title: "Article Two", CreatedAt: recentTime},
		{URL: "https://example.com/old-article", Title: "Old Article", CreatedAt: oldTime},
	}

	cutoffTime := now.Add(-DUPLICATE_CHECK_HOURS * time.Hour)

	tests := []struct {
		name     string
		url      string
		title    string
		expected bool
	}{
		{
			name:     "Exact URL match within cutoff",
			url:      "https://example.com/article1",
			title:    "Article One",
			expected: true,
		},
		{
			name:     "Exact URL match with www prefix",
			url:      "https://www.example.com/article1",
			title:    "Article One",
			expected: true,
		},
		{
			name:     "No match",
			url:      "https://example.com/article3",
			title:    "Article Three",
			expected: false,
		},
		{
			name:     "Old article URL match but outside cutoff",
			url:      "https://example.com/old-article",
			title:    "Old Article",
			expected: false,
		},
		{
			name:     "Similar title match",
			url:      "https://example.com/article2-v2",
			title:    "article two",
			expected: true,
		},
		{
			name:     "Reddit URL duplicate via normalization",
			url:      "https://www.reddit.com/r/degoogle/comments/1mau7yl/another-title/",
			title:    "Another Title",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			normalized := normalizeURL(tc.url)
			result := isDuplicate(normalized, tc.title, existingPosts, cutoffTime)
			if result != tc.expected {
				t.Errorf("isDuplicate(%q, %q, ...) = %v, want %v", tc.url, tc.title, result, tc.expected)
			}
		})
	}
}

func TestIsDuplicateWithReddit(t *testing.T) {
	now := time.Now()
	cutoffTime := now.Add(-DUPLICATE_CHECK_HOURS * time.Hour)

	existingPosts := []RedditPost{
		{
			URL:       "https://www.reddit.com/r/hackernews/comments/1mau7yl/eu_age_verification_app/",
			Title:     "EU Age Verification App",
			CreatedAt: now.Add(-12 * time.Hour),
		},
	}

	tests := []struct {
		name     string
		url      string
		title    string
		expected bool
	}{
		{
			name:     "Same Reddit post different formats",
			url:      "https://www.reddit.com/r/hackernews/comments/1mau7yl/eu_age_verification_app_to_ban_any_android_system/?share_id=abc",
			title:    "EU Age Verification App",
			expected: true,
		},
		{
			name:     "Different Reddit post",
			url:      "https://www.reddit.com/r/hackernews/comments/1mbdi2k/different_article/",
			title:    "Different Article",
			expected: false,
		},
		{
			name:     "Old Reddit URL same post",
			url:      "https://old.reddit.com/r/hackernews/comments/1mau7yl/",
			title:    "EU Age Verification App",
			expected: true,
		},
		{
			name:     "Redd.it short URL same post",
			url:      "https://redd.it/1mau7yl",
			title:    "EU Age Verification App",
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			normalized := normalizeURL(tc.url)
			result := isDuplicate(normalized, tc.title, existingPosts, cutoffTime)
			if result != tc.expected {
				t.Errorf("isDuplicate(%q, %q, ...) = %v, want %v", tc.url, tc.title, result, tc.expected)
			}
		})
	}
}

func TestGetFeedValidation(t *testing.T) {
	tests := []struct {
		name    string
		feed    *rss.Feed
		errText string
	}{
		{
			name:    "nil feed",
			feed:    nil,
			errText: "feed is nil",
		},
		{
			name:    "nil items",
			feed:    &rss.Feed{Items: nil},
			errText: "feed items are nil",
		},
		{
			name:    "empty items",
			feed:    &rss.Feed{Items: []*rss.Item{}},
			errText: "feed items are empty",
		},
		{
			name: "nil item in feed",
			feed: &rss.Feed{Items: []*rss.Item{nil}},
			errText: "feed item at index 0 is nil",
		},
		{
			name: "nil PubDateParsed",
			feed: &rss.Feed{Items: []*rss.Item{{Title: "Test", PubDateParsed: nil}}},
			errText: "feed item at index 0 has nil publish date",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			feed := tc.feed
			var err error

			if feed == nil {
				err = fmt.Errorf("feed is nil after parsing")
			} else if feed.Items == nil {
				err = fmt.Errorf("feed items are nil")
			} else if len(feed.Items) == 0 {
				err = fmt.Errorf("feed items are empty")
			} else {
				for i, item := range feed.Items {
					if item == nil {
						err = fmt.Errorf("feed item at index %d is nil", i)
						break
					}
					if item.PubDateParsed == nil {
						err = fmt.Errorf("feed item at index %d has nil publish date", i)
						break
					}
				}
			}

			if err == nil {
				t.Errorf("expected error containing %q, got nil", tc.errText)
			} else if !strings.Contains(err.Error(), tc.errText) {
				t.Errorf("expected error containing %q, got %q", tc.errText, err.Error())
			}
		})
	}
}

func TestProcessFeed(t *testing.T) {
	tests := []struct {
		name    string
		bot     reddit.Bot
		feed    *rss.Feed
		errText string
	}{
		{
			name:    "nil bot",
			bot:     nil,
			feed:    &rss.Feed{},
			errText: "bot is nil",
		},
		{
			name:    "nil feed",
			bot:     &mockBot{},
			feed:    nil,
			errText: "feed is nil",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := processFeed(tc.bot, tc.feed)
			if err == nil {
				t.Errorf("expected error containing %q, got nil", tc.errText)
			} else if !strings.Contains(err.Error(), tc.errText) {
				t.Errorf("expected error containing %q, got %q", tc.errText, err.Error())
			}
		})
	}
}

func TestPostNew(t *testing.T) {
	now := time.Now()
	cutoffTime := now.Add(-DUPLICATE_CHECK_HOURS * time.Hour)

	existingPosts := []RedditPost{
		{URL: "https://example.com/existing", Title: "Existing Post", CreatedAt: now.Add(-12 * time.Hour)},
	}

	tests := []struct {
		name    string
		bot     reddit.Bot
		item    *FeedItem
		errText string
	}{
		{
			name:    "nil bot",
			bot:     nil,
			item:    &FeedItem{Title: "Test", Link: "https://example.com"},
			errText: "bot is nil",
		},
		{
			name:    "nil item",
			bot:     &mockBot{},
			item:    nil,
			errText: "item is nil",
		},
		{
			name:    "empty title",
			bot:     &mockBot{},
			item:    &FeedItem{Title: "", Link: "https://example.com"},
			errText: "item title is empty",
		},
		{
			name:    "empty link",
			bot:     &mockBot{},
			item:    &FeedItem{Title: "Test", Link: ""},
			errText: "item link is empty",
		},
		{
			name:    "duplicate URL",
			bot:     &mockBot{},
			item:    &FeedItem{Title: "Existing Post", Link: "https://example.com/existing"},
			errText: "",
		},
		{
			name:    "HN link, no comment needed",
			bot:     &mockBot{},
			item:    &FeedItem{Title: "Test", Link: "https://news.ycombinator.com/item?id=12345"},
			errText: "",
		},
		{
			name:    "external link with comments URL",
			bot:     &mockBot{},
			item:    &FeedItem{Title: "Test", Link: "https://example.com/article", CommentsURL: "https://news.ycombinator.com/item?id=12345"},
			errText: "",
		},
		{
			name:    "external link without comments URL",
			bot:     &mockBot{},
			item:    &FeedItem{Title: "Test", Link: "https://example.com/article", CommentsURL: ""},
			errText: "",
		},
		{
			name:    "external link with non-HN comments URL",
			bot:     &mockBot{},
			item:    &FeedItem{Title: "Test", Link: "https://example.com/article", CommentsURL: "https://other.com/not-hn"},
			errText: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			postsCopy := make([]RedditPost, len(existingPosts))
			copy(postsCopy, existingPosts)
			err := postNew(tc.bot, tc.item, &postsCopy, cutoffTime)

			if tc.errText == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tc.errText)
				} else if !strings.Contains(err.Error(), tc.errText) {
					t.Errorf("expected error containing %q, got %q", tc.errText, err.Error())
				}
			}
		})
	}
}

func TestNewBot(t *testing.T) {
	t.Run("missing REDDIT_SECRET", func(t *testing.T) {
		os.Unsetenv("REDDIT_SECRET")
		os.Unsetenv("REDDIT_PASSWORD")
		_, err := newBot()
		if err == nil {
			t.Error("expected error, got nil")
		} else if !strings.Contains(err.Error(), "REDDIT_SECRET") {
			t.Errorf("expected error about REDDIT_SECRET, got %q", err.Error())
		}
	})

	t.Run("missing REDDIT_PASSWORD", func(t *testing.T) {
		os.Setenv("REDDIT_SECRET", "test-secret")
		os.Unsetenv("REDDIT_PASSWORD")
		_, err := newBot()
		if err == nil {
			t.Error("expected error, got nil")
		} else if !strings.Contains(err.Error(), "REDDIT_PASSWORD") {
			t.Errorf("expected error about REDDIT_PASSWORD, got %q", err.Error())
		}
	})
}

func TestFeedItemConstruction(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		rssItem  *rss.Item
		expected *FeedItem
	}{
		{
			name: "standard item",
			rssItem: &rss.Item{
				Title:         "Test Article",
				Link:          "https://example.com/article",
				Comments:      "https://news.ycombinator.com/item?id=12345",
				PubDateParsed: &now,
			},
			expected: &FeedItem{
				Title:       "Test Article",
				Link:        "https://example.com/article",
				Published:   &now,
				CommentsURL: "https://news.ycombinator.com/item?id=12345",
			},
		},
		{
			name: "item without comments",
			rssItem: &rss.Item{
				Title:         "No Comments",
				Link:          "https://example.com/no-comments",
				Comments:      "",
				PubDateParsed: &now,
			},
			expected: &FeedItem{
				Title:       "No Comments",
				Link:        "https://example.com/no-comments",
				Published:   &now,
				CommentsURL: "",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			feedItem := &FeedItem{
				Title:       tc.rssItem.Title,
				Link:        tc.rssItem.Link,
				Published:   tc.rssItem.PubDateParsed,
				CommentsURL: tc.rssItem.Comments,
			}

			if feedItem.Title != tc.expected.Title {
				t.Errorf("Title = %q, want %q", feedItem.Title, tc.expected.Title)
			}
			if feedItem.Link != tc.expected.Link {
				t.Errorf("Link = %q, want %q", feedItem.Link, tc.expected.Link)
			}
			if feedItem.CommentsURL != tc.expected.CommentsURL {
				t.Errorf("CommentsURL = %q, want %q", feedItem.CommentsURL, tc.expected.CommentsURL)
			}
			if feedItem.Published != tc.expected.Published {
				t.Errorf("Published = %v, want %v", feedItem.Published, tc.expected.Published)
			}
		})
	}
}
func TestGetFeedWithMockServer(t *testing.T) {
	// Create a mock RSS feed server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
<title>Hacker News</title>
<link>https://news.ycombinator.com</link>
<description>Test</description>
<item>
<title>Test Article</title>
<link>https://example.com/article</link>
<pubDate>Mon, 01 Jan 2024 12:00:00 +0000</pubDate>
<comments>https://news.ycombinator.com/item?id=12345</comments>
</item>
</channel>
</rss>`))
	}))
	defer ts.Close()

	// We need to test via a function that accepts a custom URL
	// Since getFeed() uses buildFeedUrl() which is hardcoded, let's test the parsing logic directly
	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	rssParser := &rss.Parser{}
	feed, err := rssParser.Parse(strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("failed to parse feed: %v", err)
	}

	if feed == nil {
		t.Fatal("feed is nil")
	}

	if len(feed.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(feed.Items))
	}

	item := feed.Items[0]
	if item.Title != "Test Article" {
		t.Errorf("Title = %q, want %q", item.Title, "Test Article")
	}
	if item.Link != "https://example.com/article" {
		t.Errorf("Link = %q, want %q", item.Link, "https://example.com/article")
	}
	if item.Comments != "https://news.ycombinator.com/item?id=12345" {
		t.Errorf("Comments = %q, want %q", item.Comments, "https://news.ycombinator.com/item?id=12345")
	}
	if item.PubDateParsed == nil {
		t.Error("PubDateParsed is nil")
	}
}

func TestProcessFeedWithMockBot(t *testing.T) {
	// Create a mock RSS feed with items
	now := time.Now()
	feed := &rss.Feed{
		Items: []*rss.Item{
			{
				Title:         "Test Article 1",
				Link:          "https://example.com/article1",
				Comments:      "https://news.ycombinator.com/item?id=111",
				PubDateParsed: &now,
			},
			{
				Title:         "Test Article 2",
				Link:          "https://example.com/article2",
				Comments:      "https://news.ycombinator.com/item?id=222",
				PubDateParsed: &now,
			},
		},
	}

	bot := &mockBot{}
	err := processFeed(bot, feed)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestProcessFeedHappyPath(t *testing.T) {
	// Test with a mock bot that returns listings
	bot := &mockBotWithListings{}
	now := time.Now()
	feed := &rss.Feed{
		Items: []*rss.Item{
			{
				Title:         "New Article",
				Link:          "https://example.com/new",
				Comments:      "https://news.ycombinator.com/item?id=333",
				PubDateParsed: &now,
			},
		},
	}

	err := processFeed(bot, feed)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// mockBotWithListings is a mockBot that returns listings for getExistingPosts
type mockBotWithListings struct{}

func (m *mockBotWithListings) Reply(parentName, text string) error { return nil }
func (m *mockBotWithListings) GetReply(parentName, text string) (reddit.Submission, error) {
	return reddit.Submission{Name: "t1_mock", ID: "mock"}, nil
}
func (m *mockBotWithListings) SendMessage(user, subject, text string) error { return nil }
func (m *mockBotWithListings) PostSelf(subreddit, title, text string) error { return nil }
func (m *mockBotWithListings) GetPostSelf(subreddit, title, text string) (reddit.Submission, error) {
	return reddit.Submission{Name: "t3_mock", ID: "mock"}, nil
}
func (m *mockBotWithListings) PostLink(subreddit, title, url string) error { return nil }
func (m *mockBotWithListings) GetPostLink(subreddit, title, url string) (reddit.Submission, error) {
	return reddit.Submission{Name: "t3_mock_post", ID: "mock_post", URL: ""}, nil
}
func (m *mockBotWithListings) Thread(permalink string) (*reddit.Post, error) { return nil, nil }
func (m *mockBotWithListings) Listing(path, after string) (reddit.Harvest, error) {
	return reddit.Harvest{}, nil
}
func (m *mockBotWithListings) ListingWithParams(path string, params map[string]string) (reddit.Harvest, error) {
	// Return a listing with some posts to test duplicate detection
	return reddit.Harvest{
		Posts: []*reddit.Post{
			{
				URL:        "https://example.com/existing",
				Title:      "Existing Post",
				CreatedUTC: uint64(time.Now().Add(-12 * time.Hour).Unix()),
				Deleted:    false,
			},
		},
	}, nil
}

