import { RedditPost } from "./types.js";
import { normalizeURL } from "./url.js";

export function isDuplicate(
  normalizedURL: string,
  title: string,
  existingPosts: RedditPost[],
  cutoffTime: Date
): boolean {
  const titleLower = title.toLowerCase();

  for (const post of existingPosts) {
    if (post.createdAt < cutoffTime) {
      continue;
    }

    const normalizedExisting = normalizeURL(post.url);
    if (normalizedExisting === normalizedURL) {
      return true;
    }

    if (isSimilarTitle(titleLower, post.title.toLowerCase())) {
      console.log(`Similar title found: '${title}' vs '${post.title}'`);
      return true;
    }
  }

  return false;
}

export function isSimilarTitle(title1: string, title2: string): boolean {
  if (title1 === title2) {
    return true;
  }

  if (title1.includes(title2) || title2.includes(title1)) {
    return true;
  }

  const words1 = title1.split(/\s+/);
  const words2 = title2.split(/\s+/);

  if (words1.length >= 4 && words2.length >= 4) {
    let commonWords = 0;
    const wordSet = new Set<string>();

    for (const w of words1) {
      if (w.length > 2) {
        wordSet.add(w);
      }
    }

    for (const w of words2) {
      if (w.length > 2 && wordSet.has(w)) {
        commonWords++;
      }
    }

    const minWords = Math.min(words1.length, words2.length);
    if (minWords > 0 && commonWords / minWords > 0.7) {
      return true;
    }
  }

  return false;
}
