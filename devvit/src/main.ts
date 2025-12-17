import { Devvit } from "@devvit/public-api";
import {
  SCHEDULER_JOB_NAME,
  SCHEDULER_CRON,
  REDDIT_SUBREDDIT,
} from "./constants.js";
import { registerSchedulerJob } from "./scheduler.js";

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

export default Devvit;
