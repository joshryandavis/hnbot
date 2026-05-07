# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Reddit bot that automatically posts Hacker News front page stories to r/hackernews. It fetches stories from the hnrss.org RSS feed and posts them to Reddit, adding a comment with the HN discussion link for non-HN URLs.

Two implementations exist:
- **Go version** (`main.go`) - Original implementation using graw library (legacy, blocked by Reddit API changes)
- **Devvit version** (`devvit/`) - Port to Reddit's Developer Platform (TypeScript)

## Build and Run Commands

### Go Version (Legacy)

```bash
# Run the bot (requires environment variables)
REDDIT_SECRET=... REDDIT_PASSWORD=... go run .

# Run tests
go test ./...

# Run a specific test
go test -run TestNormalizeURL

# Build binary
go build -o hnbot .
```

### Devvit Version

```bash
cd devvit

# Install dependencies
npm install

# Login to Devvit
devvit login

# Test locally on a subreddit
devvit playtest r/hackernews

# Upload to Reddit
devvit upload

# Install on subreddit
devvit install r/hackernews
```

## Environment Variables (Go Version Only)

- `REDDIT_SECRET` - Reddit API client secret
- `REDDIT_PASSWORD` - Reddit account password for u/hnmod

The Devvit version handles authentication automatically through the platform.

## Architecture

Both versions share the same logic flow:

1. **Feed fetching** (`getFeed`) - Fetches HN front page stories from hnrss.org RSS feed with filters (100+ points, 10+ comments)
2. **Feed processing** (`processFeed`) - Iterates through feed items, checks for duplicates, posts new stories
3. **Duplicate detection** (`isDuplicate`, `getExistingPosts`) - Fetches recent posts from r/hackernews (new/hot/top) and compares URLs and titles
4. **Posting** (`postNew`) - Creates Reddit link post; for non-HN URLs, adds a comment linking to the HN discussion

### Go Version Structure

Single-file application (`main.go`) using graw library for Reddit API.

### Devvit Version Structure

```
devvit/src/
├── main.ts           # Entry point, Devvit config, triggers
├── constants.ts      # Configuration constants
├── types.ts          # TypeScript interfaces
├── url.ts            # URL normalization functions
├── duplicate.ts      # Duplicate detection logic
├── feed.ts           # RSS feed fetching and parsing
├── posts.ts          # Reddit post operations
└── scheduler.ts      # Scheduled job and feed processing
```

The Devvit version uses:
- Built-in scheduler with cron (`*/15 * * * *`) instead of GitHub Actions
- `fetch()` API for RSS instead of gofeed library
- Devvit Reddit API (`context.reddit`) instead of graw library

## Key Constants

- `HN_POINTS_THRESHOLD = 100` - Minimum HN points to post
- `HN_COMMENTS_THRESHOLD = 10` - Minimum HN comments to post
- `DUPLICATE_CHECK_HOURS = 48` - Hours to look back for duplicates
- `RSS_COUNT = 50` - Number of RSS items to fetch

## URL Normalization

The bot normalizes URLs to detect duplicates across different URL formats:
- Strips `www.` prefix, normalizes to HTTPS
- Reddit URLs are normalized to their post ID (e.g., `reddit.com/comments/1mau7yl`)
- Query parameters and fragments are stripped for comparison

## CI/CD

### Go Version
GitHub Actions workflow (`.github/workflows/run.yml`) runs the bot every 15 minutes via cron schedule.

### Devvit Version
No CI/CD needed - the bot runs on Reddit's infrastructure with a built-in scheduler.
