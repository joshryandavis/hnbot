import { Feed, FeedItem } from "./types.js";
import {
  RSS_PROTOCOL,
  RSS_BASE_URL,
  RSS_FEED,
  RSS_COUNT,
  HN_POINTS_THRESHOLD,
  HN_COMMENTS_THRESHOLD,
} from "./constants.js";

export function buildFeedUrl(): string {
  const params = new URLSearchParams({
    count: RSS_COUNT.toString(),
    points: HN_POINTS_THRESHOLD.toString(),
    comments: HN_COMMENTS_THRESHOLD.toString(),
  });

  return `${RSS_PROTOCOL}://${RSS_BASE_URL}/${RSS_FEED}?${params.toString()}`;
}

export async function getFeed(): Promise<Feed> {
  console.log("Getting feed");

  const rssURL = buildFeedUrl();
  console.log("RSS URL:", rssURL);

  const response = await fetch(rssURL);
  if (!response.ok) {
    throw new Error(`Failed to fetch RSS feed: ${response.status}`);
  }

  const xmlText = await response.text();
  const feed = parseRSSFeed(xmlText);

  if (!feed) {
    throw new Error("Feed is nil after parsing");
  }

  if (!feed.items || feed.items.length === 0) {
    throw new Error("Feed items are empty");
  }

  for (let i = 0; i < feed.items.length; i++) {
    const item = feed.items[i];
    if (!item) {
      throw new Error(`Feed item at index ${i} is nil`);
    }
    if (!item.publishedParsed) {
      throw new Error(`Feed item at index ${i} has nil publish date`);
    }
  }

  return feed;
}

function parseRSSFeed(xmlText: string): Feed {
  const items: FeedItem[] = [];

  const itemRegex = /<item>([\s\S]*?)<\/item>/g;
  let match;

  while ((match = itemRegex.exec(xmlText)) !== null) {
    const itemXml = match[1];

    const title = extractTag(itemXml, "title");
    const link = extractTag(itemXml, "link");
    const guid = extractTag(itemXml, "guid");
    const pubDate = extractTag(itemXml, "pubDate");

    items.push({
      title: title || "",
      link: link || "",
      guid: guid || "",
      publishedParsed: pubDate ? new Date(pubDate) : null,
    });
  }

  return { items };
}

function extractTag(xml: string, tagName: string): string | null {
  const cdataRegex = new RegExp(
    `<${tagName}[^>]*><!\\[CDATA\\[([\\s\\S]*?)\\]\\]></${tagName}>`
  );
  const cdataMatch = xml.match(cdataRegex);
  if (cdataMatch) {
    return cdataMatch[1].trim();
  }

  const plainRegex = new RegExp(`<${tagName}[^>]*>([\\s\\S]*?)</${tagName}>`);
  const plainMatch = xml.match(plainRegex);
  if (plainMatch) {
    return plainMatch[1].trim();
  }

  return null;
}
