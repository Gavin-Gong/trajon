import { Badge as AntBadge, Button, Calendar, Card } from "antd";
import type { BadgeProps as AntBadgeProps, CalendarProps } from "antd";
import dayjs, { type Dayjs } from "dayjs";
import { useEffect, useMemo, useState } from "react";

import {
  defaultCalendarSelection,
  groupJournalsByDay,
} from "../domain/calendar";
import type { TradeJournal } from "../domain/journal";
import { tagColorForVariant } from "./EventPresentation";
import { JournalDrawer } from "./JournalDrawer";
import { shellCardStyle } from "./PanelStyles";
import type { TradingJournalStore } from "./JournalStoreTypes";

export function CalendarHistorySection({ store }: { store: TradingJournalStore }) {
  const journals = store.state.journals;
  const [displayedMonth, setDisplayedMonth] = useState<Dayjs>(() => dayjs());
  const [selectedJournalId, setSelectedJournalId] = useState<string | undefined>();
  const [openedJournalId, setOpenedJournalId] = useState<string | undefined>();
  const grouped = useMemo(() => groupJournalsByDay(journals), [journals]);
  const groupedMap = useMemo(() => new Map(grouped.map((group) => [group.dayKey, group.journals])), [grouped]);
  const monthJournals = useMemo(
    () =>
      journals
        .filter((journal) => {
          const created = dayjs(journal.createdAt);
          return created.year() === displayedMonth.year() && created.month() === displayedMonth.month();
        })
        .sort((left, right) => new Date(left.createdAt).getTime() - new Date(right.createdAt).getTime()),
    [displayedMonth, journals],
  );

  useEffect(() => {
    const selection = defaultCalendarSelection(displayedMonth.toDate(), journals);

    setSelectedJournalId((currentSelection) => {
      if (!currentSelection) {
        return selection?.id;
      }

      const stillVisible = monthJournals.some((journal) => journal.id === currentSelection);
      return stillVisible ? currentSelection : selection?.id;
    });
  }, [displayedMonth, journals, monthJournals]);

  const journalBadgeStatus = (journal: TradeJournal): AntBadgeProps["status"] =>
    journal.status === "open" ? "processing" : "success";

  const dateCellRender = (current: Dayjs) => {
    const dayKey = current.format("YYYY-MM-DD");
    const dayJournals = groupedMap.get(dayKey) ?? [];

    if (dayJournals.length === 0) {
      return null;
    }

    return (
      <div style={{ margin: 0 }}>
        {dayJournals.map((journal) => (
          <div key={journal.id} style={{ marginBottom: 4 }}>
            <AntBadge
              status={journalBadgeStatus(journal)}
              text={
                <Button
                  type="link"
                  size="small"
                  style={{
                    fontWeight: selectedJournalId === journal.id ? 700 : 500,
                    color: selectedJournalId === journal.id ? "#2563eb" : "rgba(0, 0, 0, 0.65)",
                  }}
                   onClick={(event) => {
                     event.stopPropagation();
                     setSelectedJournalId(journal.id);
                     setOpenedJournalId(journal.id);
                   }}
                 >
                  {journal.symbol || "未命名"}
                </Button>
              }
            />
          </div>
        ))}
      </div>
    );
  };

  const cellRender: CalendarProps<Dayjs>["cellRender"] = (current, info) => {
    if (info.type === "date") {
      return dateCellRender(current);
    }

    return info.originNode;
  };

  return (
    <>
      <Card style={shellCardStyle}>
        <Calendar
          mode="month"
          value={displayedMonth}
          cellRender={cellRender}
          onPanelChange={(date) => setDisplayedMonth(date)}
          onSelect={(date) => setDisplayedMonth(date)}
        />
      </Card>

      <JournalDrawer
        store={store}
        open={Boolean(openedJournalId)}
        journalId={openedJournalId}
        onClose={() => setOpenedJournalId(undefined)}
      />
    </>
  );
}
