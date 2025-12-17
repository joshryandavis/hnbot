import { Devvit } from "@devvit/public-api";
import { Feed } from "./types.js";
import {
  DUPLICATE_CHECK_HOURS,
  MAX_ERROR_COUNT,
  POST_DELAY_MS,
  SCHEDULER_JOB_NAME,
} from "./constants.js";
import { normalizeURL } from "./url.js";
import { isDuplicate } from "./duplicate.js";
import { getFeed } from "./feed.js";
import { getExistingPosts, postNew } from "./posts.js";

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export async function processFeed(
  context: Devvit.Context,
  feed: Feed
): Promise<void> {
  console.log("Processing feed");
  let processedCount = 0;
  let errorCount = 0;

  const existingPosts = await getExistingPosts(context);

  const cutoffTime = new Date(
    Date.now() - DUPLICATE_CHECK_HOURS * 60 * 60 * 1000
  );

  for (let i = 0; i < feed.items.length; i++) {
    const item = feed.items[i];

    if (!item) {
      console.log(`Warning: skipping nil item at index ${i}`);
      continue;
    }

    if (!item.publishedParsed) {
      console.log(
        `Warning: skipping item with nil publish date: ${item.title}`
      );
      continue;
    }

    if (!item.link) {
      console.log(`Warning: skipping item with empty link: ${item.title}`);
      continue;
    }

    const normalizedLink = normalizeURL(item.link);

    if (isDuplicate(normalizedLink, item.title, existingPosts, cutoffTime)) {
      console.log(`Post already exists, skipping: ${item.link}`);
      continue;
    }

    try {
      await postNew(context, item, existingPosts, cutoffTime);
      processedCount++;
      await sleep(POST_DELAY_MS);
    } catch (err) {
      errorCount++;
      console.log(`Error posting item ${i} (${item.title}): ${err}`);
      if (errorCount >= MAX_ERROR_COUNT) {
        throw new Error(`Too many posting errors (${errorCount}): aborting`);
      }
    }
  }

  console.log(`Successfully processed ${processedCount} items`);
}

export function registerSchedulerJob(): void {
  Devvit.addSchedulerJob({
    name: SCHEDULER_JOB_NAME,
    onRun: async (_event, context) => {
      console.log("Starting");

      try {
        const feed = await getFeed();

        if (!feed) {
          throw new Error("Error: feed is nil");
        }

        await processFeed(context, feed);

        console.log("Done");
      } catch (err) {
        console.error("Bot run failed:", err);
        throw err;
      }
    },
  });
}
