export type TradeJournalStatus = "open" | "done";

export type TradeEventType = "entry" | "add" | "reduce" | "exit" | "log" | "plan";

export type TradeEvent = {
  id: string;
  createdAt: string;
  updatedAt: string;
  timestamp: string;
  type: TradeEventType;
  content: string;
  price?: number;
  quantity?: number;
};

export type JournalImage = {
  id: string;
  journalId: string;
  createdAt: string;
  updatedAt: string;
  fileName: string;
  relativePath: string;
  sortOrder: number;
};

export type TradeJournal = {
  id: string;
  createdAt: string;
  updatedAt: string;
  symbol: string;
  summary: string;
  tags: string[];
  status: TradeJournalStatus;
  events: TradeEvent[];
  images: JournalImage[];
};
