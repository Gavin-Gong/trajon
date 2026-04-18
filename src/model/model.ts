export type TradeJournalStatus = "open" | "done";

export type TradeEventType = "entry" | "add" | "reduce" | "exit" | "log";

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

export type TradeJournal = {
  id: string;
  createdAt: string;
  updatedAt: string;
  symbol: string;
  market: string;
  summary: string;
  status: TradeJournalStatus;
  events: TradeEvent[];
};
