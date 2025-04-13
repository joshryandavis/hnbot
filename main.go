package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
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
	REDDIT_MAX_POSTS      = "100"
	REDDIT_ID             = "v7eIyAVMwtcKG00ahocIXg"
	HN_BASE_URL           = "news.ycombinator.com"
	RSS_PROTOCOL          = "https"
	RSS_BASE_URL          = "hnrss.org"
	RSS_FEED              = "frontpage"
	RSS_COUNT             = 50
	HN_POINTS_THRESHOLD   = 100
	HN_COMMENTS_THRESHOLD = 10
)

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

	existingLinks, err := getExistingPosts(bot)
	if err != nil {
		return fmt.Errorf("error getting existing posts: %w", err)
	}

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

		if _, exists := existingLinks[item.Link]; exists {
			fmt.Printf("Post already exists, skipping: %s\n", item.Link)
			continue
		}

		err := postNew(bot, item, existingLinks)
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

func getExistingPosts(bot reddit.Bot) (map[string]bool, error) {
	if bot == nil {
		return nil, errors.New("bot is nil")
	}

	fmt.Println("Getting existing posts from subreddit")
	postUrl := fmt.Sprintf("/r/%s/new", REDDIT_SUBREDDIT)

	postOpts := map[string]string{
		"limit": REDDIT_MAX_POSTS,
	}

	posts, err := bot.ListingWithParams(postUrl, postOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to get Reddit listings: %w", err)
	}

	if posts.Posts == nil {
		return nil, errors.New("posts.Posts is nil")
	}

	existingLinks := make(map[string]bool)
	for _, post := range posts.Posts {
		if post.URL != "" && !post.Deleted {
			existingLinks[post.URL] = true
		}
	}

	if len(existingLinks) == 0 {
		fmt.Println("No existing posts found")
		return nil, errors.New("no existing posts found")
	}

	fmt.Printf("Found %d existing posts\n", len(existingLinks))
	return existingLinks, nil
}

func postNew(bot reddit.Bot, item *gofeed.Item, existingLinks map[string]bool) error {
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

	if exists := existingLinks[item.Link]; exists {
		fmt.Println("Post already exists, skipping:", item.Link)
		return nil
	}

	submission, err := bot.GetPostLink(REDDIT_SUBREDDIT, item.Title, item.Link)
	if err != nil {
		return fmt.Errorf("failed to create Reddit post: %w", err)
	}

	if submission.Name == "" {
		return errors.New("no post id returned")
	}

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
