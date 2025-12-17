import { Devvit } from "@devvit/public-api";
import { RedditPost, FeedItem } from "./types.js";
import { REDDIT_SUBREDDIT, HN_BASE_URL } from "./constants.js";
import { normalizeURL } from "./url.js";
import { isDuplicate } from "./duplicate.js";

export async function getExistingPosts(
  context: Devvit.Context
): Promise<RedditPost[]> {
  console.log("Getting existing posts from subreddit");

  const allPosts: RedditPost[] = [];
  let lastErr: Error | null = null;
  let successCount = 0;

  const pageTypes = ["new", "hot", "top"] as const;

  for (const pageType of pageTypes) {
    try {
      let posts;

      if (pageType === "new") {
        posts = await context.reddit
          .getNewPosts({
            subredditName: REDDIT_SUBREDDIT,
            limit: 100,
          })
          .all();
      } else if (pageType === "hot") {
        posts = await context.reddit
          .getHotPosts({
            subredditName: REDDIT_SUBREDDIT,
            limit: 100,
          })
          .all();
      } else {
        posts = await context.reddit
          .getTopPosts({
            subredditName: REDDIT_SUBREDDIT,
            timeframe: "week",
            limit: 100,
          })
          .all();
      }

      successCount++;

      for (const post of posts) {
        if (post.url && !post.removed) {
          allPosts.push({
            url: post.url,
            title: post.title,
            createdAt: post.createdAt,
          });
        }
      }
    } catch (err) {
      console.log(`Warning: failed to get ${pageType} listings: ${err}`);
      lastErr = err as Error;
    }
  }

  if (successCount === 0) {
    throw new Error(`Failed to fetch any listings: ${lastErr?.message}`);
  }

  if (allPosts.length === 0) {
    console.log("No existing posts found");
  } else {
    console.log(`Found ${allPosts.length} existing posts across new/hot/top`);
  }

  return allPosts;
}

export async function postNew(
  context: Devvit.Context,
  item: FeedItem,
  existingPosts: RedditPost[],
  cutoffTime: Date
): Promise<void> {
  if (!item.title) {
    throw new Error("Item title is empty");
  }

  if (!item.link) {
    throw new Error("Item link is empty");
  }

  console.log("Posting:", item.title);

  const isHn = item.link.includes(HN_BASE_URL);
  console.log("HN link:", isHn);

  const normalizedLink = normalizeURL(item.link);

  if (isDuplicate(normalizedLink, item.title, existingPosts, cutoffTime)) {
    console.log("Post already exists (double-check), skipping:", item.link);
    return;
  }

  const subreddit = await context.reddit.getSubredditByName(REDDIT_SUBREDDIT);
  const submission = await subreddit.submitPost({
    title: item.title,
    url: item.link,
  });

  if (!submission.id) {
    throw new Error("No post id returned");
  }

  existingPosts.push({
    url: item.link,
    title: item.title,
    createdAt: new Date(),
  });

  if (isHn) {
    return;
  }

  const hnLink = item.guid;
  if (!hnLink) {
    console.log(
      `Warning: no HN link found in GUID for '${item.title}', skipping comment`
    );
    return;
  }

  if (!hnLink.includes(HN_BASE_URL)) {
    console.log(
      `Warning: GUID is not an HN link for '${item.title}': ${hnLink}, skipping comment`
    );
    return;
  }

  const commentTxt = "Discussion on HN: " + hnLink;
  const reply = await submission.addComment({ text: commentTxt });

  if (!reply.id) {
    throw new Error("No comment id returned");
  }
}
