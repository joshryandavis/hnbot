export interface RedditPost {
  url: string;
  title: string;
  createdAt: Date;
}

export interface FeedItem {
  title: string;
  link: string;
  guid: string;
  publishedParsed: Date | null;
}

export interface Feed {
  items: FeedItem[];
}
