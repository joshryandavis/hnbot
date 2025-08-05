package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/turnage/graw/reddit"
)

const (
	REDDIT_TIMEOUT        = 30
	REDDIT_SUBREDDIT      = "hackernews"
	REDDIT_AGENT          = "hackernews:hnmod:0.1.0"
	REDDIT_USERNAME       = "hnmod"
	REDDIT_ID             = "v7eIyAVMwtcKG00ahocIXg"
	HN_BASE_URL           = "news.ycombinator.com"
	RSS_PROTOCOL          = "https"
	RSS_BASE_URL          = "hnrss.org"
	RSS_FEED              = "frontpage"
	RSS_COUNT             = 50
	HN_POINTS_THRESHOLD   = 100
	HN_COMMENTS_THRESHOLD = 10
	DUPLICATE_CHECK_HOURS = 48
)

type RedditPost struct {
	URL       string
	Title     string
	CreatedAt time.Time
}

func normalizeURL(rawURL string) string {
	if rawURL == "" {
		return rawURL
	}

	if strings.Contains(rawURL, "reddit.com") || strings.Contains(rawURL, "redd.it") {
		return normalizeRedditURL(rawURL)
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	parsedURL.Scheme = strings.ToLower(parsedURL.Scheme)
	parsedURL.Host = strings.ToLower(parsedURL.Host)

	if parsedURL.Scheme == "" || parsedURL.Scheme == "http" {
		parsedURL.Scheme = "https"
	}

	if strings.HasPrefix(parsedURL.Host, "www.") {
		parsedURL.Host = strings.TrimPrefix(parsedURL.Host, "www.")
	}

	// Decode and re-encode the path to normalize percent-encoding
	if decodedPath, err := url.QueryUnescape(parsedURL.Path); err == nil {
		parsedURL.Path = decodedPath
	}

	parsedURL.Path = strings.TrimSuffix(parsedURL.Path, "/")
	if parsedURL.Path == "" {
		parsedURL.Path = "/"
	}

	parsedURL.Fragment = ""
	
	// Re-encode the URL to ensure consistent encoding
	parsedURL.RawPath = ""  // Let Go re-encode the path

	return parsedURL.String()
}

func main() {
	fmt.Println("Starting")

	bot, err := newBot()
	if err != nil {
		fmt.Println("Error creating Reddit bot:", err)
		return
	}

	if bot == nil {
		fmt.Println("Error: Reddit bot is nil")
		return
	}

	feed, err := getFeed()
	if err != nil {
		fmt.Println("Error getting feed:", err)
		return
	}

	if feed == nil {
		fmt.Println("Error: feed is nil")
		return
	}

	err = processFeed(bot, feed)
	if err != nil {
		fmt.Println("Error processing:", err)
		return
	}

	fmt.Println("Done")
}

func buildFeedUrl() *url.URL {
	rssURL := &url.URL{
		Scheme: RSS_PROTOCOL,
		Host:   RSS_BASE_URL,
		Path:   RSS_FEED,
	}

	query := rssURL.Query()
	query.Set("count", fmt.Sprintf("%d", RSS_COUNT))
	query.Set("points", fmt.Sprintf("%d", HN_POINTS_THRESHOLD))
	query.Set("comments", fmt.Sprintf("%d", HN_COMMENTS_THRESHOLD))

	rssURL.RawQuery = query.Encode()

	return rssURL
}

func getFeed() (*gofeed.Feed, error) {
	fmt.Println("Getting feed")

	rssURL := buildFeedUrl()

	fmt.Println("RSS URL:", rssURL.String())

	fp := gofeed.NewParser()
	if fp == nil {
		return nil, errors.New("failed to create feed parser")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*REDDIT_TIMEOUT)
	defer cancel()

	feed, err := fp.ParseURLWithContext(rssURL.String(), ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed URL: %w", err)
	}

	if feed == nil {
		return nil, errors.New("feed is nil after parsing")
	}

	if feed.Items == nil {
		return nil, errors.New("feed items are nil")
	}

	if len(feed.Items) == 0 {
		return nil, errors.New("feed items are empty")
	}

	for i, item := range feed.Items {
		if item == nil {
			return nil, fmt.Errorf("feed item at index %d is nil", i)
		}
		if item.PublishedParsed == nil {
			return nil, fmt.Errorf("feed item at index %d has nil publish date", i)
		}
	}

	return feed, nil
}

func processFeed(bot reddit.Bot, feed *gofeed.Feed) error {
	if bot == nil {
		return errors.New("bot is nil")
	}

	if feed == nil {
		return errors.New("feed is nil")
	}

	fmt.Println("Processing feed")
	processedCount := 0
	errorCount := 0

	existingPosts, err := getExistingPosts(bot)
	if err != nil {
		return fmt.Errorf("error getting existing posts: %w", err)
	}

	cutoffTime := time.Now().Add(-DUPLICATE_CHECK_HOURS * time.Hour)

	for i, item := range feed.Items {
		if item == nil {
			fmt.Printf("Warning: skipping nil item at index %d\n", i)
			continue
		}

		if item.PublishedParsed == nil {
			fmt.Printf("Warning: skipping item with nil publish date: %s\n", item.Title)
			continue
		}

		if item.Link == "" {
			fmt.Printf("Warning: skipping item with empty link: %s\n", item.Title)
			continue
		}

		normalizedLink := normalizeURL(item.Link)

		if isDuplicate(normalizedLink, item.Title, existingPosts, cutoffTime) {
			fmt.Printf("Post already exists, skipping: %s\n", item.Link)
			continue
		}

		err := postNew(bot, item, &existingPosts)
		if err != nil {
			errorCount++
			fmt.Printf("Error posting item %d (%s): %v\n", i, item.Title, err)
			if errorCount >= 3 {
				return fmt.Errorf("too many posting errors (%d): aborting", errorCount)
			}
			continue
		}
		processedCount++
	}

	fmt.Printf("Successfully processed %d items\n", processedCount)
	return nil
}

func normalizeRedditURL(rawURL string) string {
	if rawURL == "" {
		return rawURL
	}

	cleanURL := rawURL
	if idx := strings.Index(cleanURL, "?"); idx != -1 {
		cleanURL = cleanURL[:idx]
	}

	redditIDRegex := regexp.MustCompile(`(?:reddit\.com/r/[^/]+/comments/|redd\.it/)([a-zA-Z0-9]+)`)
	if matches := redditIDRegex.FindStringSubmatch(cleanURL); len(matches) > 1 {
		return matches[1]
	}

	if strings.Contains(cleanURL, "/s/") {
		return cleanURL
	}

	return rawURL
}

func getExistingPosts(bot reddit.Bot) ([]RedditPost, error) {
	if bot == nil {
		return nil, errors.New("bot is nil")
	}

	fmt.Println("Getting existing posts from subreddit")
	var allPosts []RedditPost
	
	pageTypes := []string{"new", "hot", "top"}
	for _, pageType := range pageTypes {
		postUrl := fmt.Sprintf("/r/%s/%s", REDDIT_SUBREDDIT, pageType)
		postOpts := map[string]string{
			"limit": "100",
		}
		
		if pageType == "top" {
			postOpts["t"] = "week"
		}

		posts, err := bot.ListingWithParams(postUrl, postOpts)
		if err != nil {
			fmt.Printf("Warning: failed to get %s listings: %v\n", pageType, err)
			continue
		}

		if posts.Posts == nil {
			continue
		}

		for _, post := range posts.Posts {
			if post.URL != "" && !post.Deleted {
				allPosts = append(allPosts, RedditPost{
					URL:       post.URL,
					Title:     post.Title,
					CreatedAt: time.Unix(int64(post.CreatedUTC), 0),
				})
			}
		}
	}

	if len(allPosts) == 0 {
		fmt.Println("No existing posts found")
		return []RedditPost{}, nil
	}

	fmt.Printf("Found %d existing posts across new/hot/top\n", len(allPosts))
	return allPosts, nil
}

func isDuplicate(normalizedURL string, title string, existingPosts []RedditPost, cutoffTime time.Time) bool {
	titleLower := strings.ToLower(title)
	
	for _, post := range existingPosts {
		if post.CreatedAt.Before(cutoffTime) {
			continue
		}
		
		normalizedExisting := normalizeURL(post.URL)
		if normalizedExisting == normalizedURL {
			return true
		}
		
		if isSimilarTitle(titleLower, strings.ToLower(post.Title)) {
			fmt.Printf("Similar title found: '%s' vs '%s'\n", title, post.Title)
			return true
		}
	}
	
	return false
}

func isSimilarTitle(title1, title2 string) bool {
	if title1 == title2 {
		return true
	}
	
	if len(title1) > 20 && len(title2) > 20 {
		if strings.Contains(title1, title2) || strings.Contains(title2, title1) {
			return true
		}
		
		commonLength := min(len(title1), len(title2))
		if commonLength > 30 && title1[:30] == title2[:30] {
			return true
		}
	}
	
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func postNew(bot reddit.Bot, item *gofeed.Item, existingPosts *[]RedditPost) error {
	if bot == nil {
		return errors.New("bot is nil")
	}

	if item == nil {
		return errors.New("item is nil")
	}

	if item.Title == "" {
		return errors.New("item title is empty")
	}

	if item.Link == "" {
		return errors.New("item link is empty")
	}

	fmt.Println("Posting:", item.Title)

	isHn := false
	if item.Link != "" && strings.Contains(item.Link, HN_BASE_URL) {
		isHn = true
	}
	fmt.Println("HN link:", isHn)

	normalizedLink := normalizeURL(item.Link)
	cutoffTime := time.Now().Add(-DUPLICATE_CHECK_HOURS * time.Hour)

	if isDuplicate(normalizedLink, item.Title, *existingPosts, cutoffTime) {
		fmt.Println("Post already exists (double-check), skipping:", item.Link)
		return nil
	}

	submission, err := bot.GetPostLink(REDDIT_SUBREDDIT, item.Title, item.Link)
	if err != nil {
		return fmt.Errorf("failed to create Reddit post: %w", err)
	}

	if submission.Name == "" {
		return errors.New("no post id returned")
	}

	*existingPosts = append(*existingPosts, RedditPost{
		URL:       item.Link,
		Title:     item.Title,
		CreatedAt: time.Now(),
	})

	if isHn {
		return nil
	}

	hnLink := item.GUID
	if hnLink == "" {
		return errors.New("no HN link found in GUID")
	}

	if !strings.Contains(hnLink, HN_BASE_URL) {
		return fmt.Errorf("GUID not a HN link: %s", hnLink)
	}

	commentTxt := "Discussion on HN: " + hnLink
	reply, err := bot.GetReply(submission.Name, commentTxt)
	if err != nil {
		return fmt.Errorf("failed to post comment: %w", err)
	}

	if reply.Name == "" {
		return errors.New("no comment id returned")
	}

	return nil
}

func newBot() (reddit.Bot, error) {
	fmt.Println("Getting Reddit bot")

	secret := os.Getenv("REDDIT_SECRET")
	if secret == "" {
		return nil, errors.New("no Reddit secret provided in environment variable REDDIT_SECRET")
	}

	password := os.Getenv("REDDIT_PASSWORD")
	if password == "" {
		return nil, errors.New("no Reddit password provided in environment variable REDDIT_PASSWORD")
	}

	transport := &http.Transport{
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		DisableCompression:    false,
		DisableKeepAlives:     false,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Second * REDDIT_TIMEOUT,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("too many redirects")
			}
			return nil
		},
	}

	cfg := reddit.BotConfig{
		Agent: REDDIT_AGENT,
		App: reddit.App{
			ID:       REDDIT_ID,
			Username: REDDIT_USERNAME,
			Secret:   secret,
			Password: password,
		},
		Rate:   1 * time.Second,
		Client: client,
	}

	bot, err := reddit.NewBot(cfg)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			if urlErr.Timeout() {
				return nil, fmt.Errorf("Reddit API connection timed out: %w", err)
			}
			return nil, fmt.Errorf("Reddit API connection error: %w", err)
		}
		return nil, fmt.Errorf("failed to create Reddit bot: %w", err)
	}

	if bot == nil {
		return nil, errors.New("bot is nil after creation")
	}

	return bot, nil
}