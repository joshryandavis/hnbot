# HN Bot - Devvit Version

This is a port of the original Go bot to [Reddit's Developer Platform (Devvit)](https://developers.reddit.com). It maintains the same logic, function names, and coding style as the original.

## Why Devvit?

After Reddit's API changes in 2023, traditional bots using the old API face restrictions. Devvit is Reddit's official developer platform that:

- Provides guaranteed API access for approved apps
- Hosts your bot on Reddit's infrastructure (no external hosting needed)
- Includes built-in scheduling (no GitHub Actions or cron required)
- Handles authentication automatically

## Prerequisites

- Node.js 22.2.0 or later
- A Reddit account
- Moderator access to r/hackernews (or a test subreddit)

## Setup

### 1. Install Dependencies

```bash
cd devvit
npm install
```

### 2. Install Devvit CLI

```bash
npm install -g devvit
```

### 3. Login to Reddit

```bash
devvit login
```

This will open a browser window for Reddit OAuth authentication.

## Testing

### Local Playtest

Test the bot on a subreddit without deploying:

```bash
devvit playtest r/your-test-subreddit
```

This runs the app locally while connected to Reddit. You can trigger the bot manually using the "Run HN Bot Now" menu item in the subreddit's mod menu.

### View Logs

During playtest, logs appear in your terminal. After deployment, view logs with:

```bash
devvit logs r/hackernews
```

## Deployment

### 1. Upload the App

```bash
devvit upload
```

This uploads your app to Reddit's servers.

### 2. Install on Subreddit

```bash
devvit install r/hackernews
```

After installation, the bot will automatically:
- Schedule itself to run every 15 minutes
- Start posting HN stories that meet the threshold (100+ points, 10+ comments)

## Project Structure

```
devvit/
├── devvit.yaml          # App configuration
├── package.json         # Dependencies
├── tsconfig.json        # TypeScript config
└── src/
    ├── main.ts          # Entry point, Devvit setup, triggers
    ├── constants.ts     # Configuration (thresholds, intervals)
    ├── types.ts         # TypeScript interfaces
    ├── url.ts           # URL normalization (same as Go version)
    ├── duplicate.ts     # Duplicate detection (same as Go version)
    ├── feed.ts          # RSS fetching from hnrss.org
    ├── posts.ts         # Reddit post/comment operations
    └── scheduler.ts     # Feed processing and scheduled job
```

## Function Mapping (Go → TypeScript)

| Go Function | TypeScript Location |
|-------------|---------------------|
| `normalizeURL()` | `url.ts` |
| `normalizeRedditURL()` | `url.ts` |
| `buildFeedUrl()` | `feed.ts` |
| `getFeed()` | `feed.ts` |
| `isDuplicate()` | `duplicate.ts` |
| `isSimilarTitle()` | `duplicate.ts` |
| `getExistingPosts()` | `posts.ts` |
| `postNew()` | `posts.ts` |
| `processFeed()` | `scheduler.ts` |
| `main()` | `scheduler.ts` (onRun handler) |
| `newBot()` | N/A (Devvit handles auth) |

## Configuration

All constants are in `src/constants.ts` with the same names as the Go version:

- `HN_POINTS_THRESHOLD = 100`
- `HN_COMMENTS_THRESHOLD = 10`
- `DUPLICATE_CHECK_HOURS = 48`
- `RSS_COUNT = 50`
- `SCHEDULER_CRON = "*/15 * * * *"`

## Manual Trigger

After installation, you can manually trigger the bot:

1. Go to r/hackernews
2. Click the mod menu (shield icon)
3. Select "Run HN Bot Now"

## Troubleshooting

### App not running?

Check if the scheduled job is registered:
```bash
devvit logs r/hackernews --since 1h
```

### Permission errors?

Ensure the app has the required permissions by reinstalling:
```bash
devvit uninstall r/hackernews
devvit install r/hackernews
```

### Rate limiting?

The bot includes a 2-second delay between posts (same as the Go version). If you're still hitting limits, Reddit's platform handles backoff automatically.

## Differences from Go Version

1. **No environment variables** - Devvit handles authentication
2. **Built-in scheduler** - No GitHub Actions needed
3. **RSS parsing** - Uses simple regex parser instead of gofeed (works fine for hnrss.org's clean XML)
4. **Hosting** - Runs on Reddit's servers, not yours
