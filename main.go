package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/turnage/graw/reddit"
)

const (
	TIMEOUT         = 30
	HN_BASE_URL     = "news.ycombinator.com"
	SUBREDDIT       = "hackernews"
	REDDIT_USERNAME = "hnmod"
	REDDIT_ID       = "v7eIyAVMwtcKG00ahocIXg"
	RSS_URL         = "https://hnrss.org/frontpage"
	MAX_TIME_WINDOW = 24 * 60
	MIN_TIME_WINDOW = 1
)

func main() {
	fmt.Println("Starting")

	if len(os.Args) < 2 {
		fmt.Println("Error: missing time window argument")
		fmt.Println("Usage: program <minutes>")
		return
	}

	timeWindow, err := getTimeWindow()
	if err != nil {
		fmt.Println("Error getting time window:", err)
		return
	}

	if timeWindow == 0 {
		fmt.Println("Error: time window cannot be zero")
		return
	}

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

	err = processFeed(bot, feed, timeWindow)
	if err != nil {
		fmt.Println("Error processing:", err)
		return
	}

	fmt.Println("Done")
}

func getTimeWindow() (time.Duration, error) {
	fmt.Println("Getting time window")
	args := os.Args[1:]
	if len(args) != 1 {
		return 0, errors.New("invalid arguments: expected exactly one argument")
	}

	if args[0] == "" {
		return 0, errors.New("invalid argument: time window cannot be empty")
	}

	minutesInt, err := strconv.Atoi(args[0])
	if err != nil {
		return 0, fmt.Errorf("invalid time format - must be an integer: %v", err)
	}

	if minutesInt < MIN_TIME_WINDOW {
		return 0, fmt.Errorf("invalid time window: must be at least %d minute(s)", MIN_TIME_WINDOW)
	}

	if minutesInt > MAX_TIME_WINDOW {
		return 0, fmt.Errorf("invalid time window: must be at most %d minutes", MAX_TIME_WINDOW)
	}

	timeWindow := time.Duration(minutesInt) * time.Minute
	fmt.Println("Time window:", timeWindow)
	return timeWindow, nil
}

func getFeed() (*gofeed.Feed, error) {
	fmt.Println("Getting feed")

	fp := gofeed.NewParser()
	if fp == nil {
		return nil, errors.New("failed to create feed parser")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*TIMEOUT)
	defer cancel()

	feed, err := fp.ParseURLWithContext(RSS_URL, ctx)
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

func processFeed(bot reddit.Bot, feed *gofeed.Feed, timeWindow time.Duration) error {
	if bot == nil {
		return errors.New("bot is nil")
	}

	if feed == nil {
		return errors.New("feed is nil")
	}

	if timeWindow <= 0 {
		return errors.New("timeWindow must be positive")
	}

	fmt.Println("Processing feed")
	processedCount := 0
	errorCount := 0

	cutoffTime := time.Now().UTC().Add(-timeWindow)

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

		if item.PublishedParsed.UTC().After(cutoffTime) {
			err := postNew(bot, item)
			if err != nil {
				errorCount++
				fmt.Printf("Error posting item %d (%s): %v\n", i, item.Title, err)
				if errorCount >= 3 {
					return fmt.Errorf("too many posting errors (%d): aborting", errorCount)
				}
				continue
			}
			processedCount++
		} else {
			fmt.Println("Skipping outdated item:", item.Title)
		}
	}

	fmt.Printf("Successfully processed %d items\n", processedCount)
	return nil
}

func postNew(bot reddit.Bot, item *gofeed.Item) error {
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

	exists, err := checkPostExists(bot, item.Link)
	if err != nil {
		return fmt.Errorf("error checking if post exists: %w", err)
	}

	if exists {
		fmt.Println("Post already exists, skipping:", item.Title)
		return nil
	}

	if SUBREDDIT == "" {
		return errors.New("SUBREDDIT constant is empty")
	}

	submission, err := bot.GetPostLink(SUBREDDIT, item.Title, item.Link)
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

func checkPostExists(bot reddit.Bot, link string) (bool, error) {
	if bot == nil {
		return false, errors.New("bot is nil")
	}

	if link == "" {
		return false, errors.New("link is empty")
	}

	if SUBREDDIT == "" {
		return false, errors.New("SUBREDDIT constant is empty")
	}

	fmt.Println("Checking if post exists")
	postUrl := fmt.Sprintf("/r/%s/new", SUBREDDIT)

	postOpts := map[string]string{
		"limit": "100",
	}

	posts, err := bot.ListingWithParams(postUrl, postOpts)
	if err != nil {
		return false, fmt.Errorf("failed to get Reddit listings: %w", err)
	}

	if posts.Posts == nil {
		return false, errors.New("posts.Posts is nil")
	}

	for _, post := range posts.Posts {
		if post.URL == link {
			return true, nil
		}
	}

	return false, nil
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

	cfg := reddit.BotConfig{
		Agent: "hakernews:hnmod:0.1.0",
		App: reddit.App{
			ID:       REDDIT_ID,
			Username: REDDIT_USERNAME,
			Secret:   secret,
			Password: password,
		},
		Rate:   5 * time.Second,
		Client: &http.Client{Timeout: time.Second * TIMEOUT},
	}

	bot, err := reddit.NewBot(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Reddit bot: %w", err)
	}

	if bot == nil {
		return nil, errors.New("bot is nil after creation")
	}

	return bot, nil
}
