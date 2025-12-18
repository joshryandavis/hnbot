import { Devvit } from "@devvit/public-api";
import {
  SCHEDULER_JOB_NAME,
  SCHEDULER_CRON,
  REDDIT_SUBREDDIT,
} from "./constants.js";
import { registerSchedulerJob } from "./scheduler.js";
import { getFeed } from "./feed.js";

Devvit.configure({
  redditAPI: true,
  http: true,
});

registerSchedulerJob();

Devvit.addTrigger({
  event: "AppInstall",
  onEvent: async (_event, context) => {
    console.log(`HN Bot installed on r/${REDDIT_SUBREDDIT}`);

    await context.scheduler.runJob({
      name: SCHEDULER_JOB_NAME,
      cron: SCHEDULER_CRON,
    });

    console.log("Scheduled recurring job with cron:", SCHEDULER_CRON);
  },
});

Devvit.addMenuItem({
  label: "Run HN Bot Now",
  location: "subreddit",
  onPress: async (_event, context) => {
    console.log("Manual trigger requested");

    await context.scheduler.runJob({
      name: SCHEDULER_JOB_NAME,
      runAt: new Date(),
    });

    context.ui.showToast("HN Bot job triggered");
  },
});

// Dry-run test: only fetches RSS feed, doesn't post anything
Devvit.addMenuItem({
  label: "Test HN Feed Fetch (Dry Run)",
  location: "subreddit",
  onPress: async (_event, context) => {
    console.log("=== DRY RUN TEST START ===");
    console.log("Testing fetch to hnrss.org...");

    try {
      const feed = await getFeed();
      console.log(`SUCCESS: Fetched ${feed.items.length} items from hnrss.org`);
      console.log("First 3 items:");
      for (let i = 0; i < Math.min(3, feed.items.length); i++) {
        const item = feed.items[i];
        console.log(`  ${i + 1}. ${item.title}`);
        console.log(`     Link: ${item.link}`);
        console.log(`     HN: ${item.guid}`);
      }
      context.ui.showToast(`Success! Fetched ${feed.items.length} items from hnrss.org`);
    } catch (err) {
      console.error("FAILED to fetch from hnrss.org:", err);
      context.ui.showToast(`Failed: ${err}`);
    }

    console.log("=== DRY RUN TEST END ===");
  },
});

export default Devvit;
