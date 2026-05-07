export function normalizeURL(rawURL: string): string {
  if (!rawURL) {
    return rawURL;
  }

  if (rawURL.includes("reddit.com") || rawURL.includes("redd.it")) {
    return normalizeRedditURL(rawURL);
  }

  let parsedURL: URL;
  try {
    parsedURL = new URL(rawURL);
  } catch {
    return rawURL;
  }

  parsedURL.protocol = parsedURL.protocol.toLowerCase();
  parsedURL.hostname = parsedURL.hostname.toLowerCase();

  if (parsedURL.protocol === "" || parsedURL.protocol === "http:") {
    parsedURL.protocol = "https:";
  }

  if (parsedURL.hostname.startsWith("www.")) {
    parsedURL.hostname = parsedURL.hostname.slice(4);
  }

  let path = parsedURL.pathname.replace(/\/$/, "");
  if (path === "") {
    path = "/";
  }
  parsedURL.pathname = path;

  parsedURL.hash = "";
  parsedURL.search = "";

  return parsedURL.toString();
}

export function normalizeRedditURL(rawURL: string): string {
  if (!rawURL) {
    return rawURL;
  }

  let cleanURL = rawURL;
  const queryIndex = cleanURL.indexOf("?");
  if (queryIndex !== -1) {
    cleanURL = cleanURL.slice(0, queryIndex);
  }

  const redditIDRegex = /(?:reddit\.com\/r\/[^/]+\/comments\/|redd\.it\/)([a-zA-Z0-9]+)/;
  const matches = cleanURL.match(redditIDRegex);
  if (matches && matches[1]) {
    return "reddit.com/comments/" + matches[1];
  }

  if (cleanURL.includes("/s/")) {
    const shareIDRegex = /\/s\/([a-zA-Z0-9]+)/;
    const shareMatches = cleanURL.match(shareIDRegex);
    if (shareMatches && shareMatches[1]) {
      return "reddit.com/s/" + shareMatches[1];
    }
  }

  return cleanURL;
}
