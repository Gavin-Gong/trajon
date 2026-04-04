import { getCurrentWindow } from "@tauri-apps/api/window";
import { useEffect, useMemo, useRef, useState } from "react";

import {
  availableEventTypes,
  composeTradeEvent,
  currentJournal as selectCurrentJournal,
  defaultCalendarSelection,
  groupJournalsByDay,
  isZero,
  netPosition,
  visibleMonthDays,
  type TradeEvent,
  type TradeEventComposerError,
  type TradeEventDraft,
  type TradeEventType,
  type TradeJournal,
} from "./tradingPanelDomain";
import {
  openPanelWindow,
  PANEL_BOUNDS_STORAGE_KEY,
  serializePanelBounds,
} from "./panelWindow";
import { useTradingPanelState } from "./tradingPanelStore";

type AppView = "main" | "panel";
type AppSection = "analytics" | "calendar" | "settings";

export function TradingPanelDesktop({ view }: { view: AppView }) {
  const store = useTradingPanelState();

  if (view === "panel") {
    return <PanelSurface store={store} />;
  }

  return <MainSurface store={store} />;
}

function MainSurface({ store }: { store: ReturnType<typeof useTradingPanelState> }) {
  const [section, setSection] = useState<AppSection>("analytics");

  return (
    <main className="desktop-shell main-shell">
      <aside className="sidebar">
        <div className="sidebar-brand">
          <p className="eyebrow">Trading Panel</p>
          <h1>Trading Panel</h1>
          <p className="supporting-copy">
            Tauri migration of the Swift trading helper: panel capture, calendar
            history, and settings in one desktop app.
          </p>
        </div>

        <nav className="sidebar-nav" aria-label="Sections">
          <SidebarButton
            active={section === "analytics"}
            label="分析"
            icon="◌"
            onClick={() => setSection("analytics")}
          />
          <SidebarButton
            active={section === "calendar"}
            label="日历"
            icon="▦"
            onClick={() => setSection("calendar")}
          />
          <SidebarButton
            active={section === "settings"}
            label="设置"
            icon="⚙"
            onClick={() => setSection("settings")}
          />
        </nav>

        <div className="sidebar-footer">
          <button
            type="button"
            className="primary-button"
            onClick={() => {
              void openPanelWindow();
            }}
          >
            Open Panel
          </button>
          <p className="hint-text">Panel stays focused on active journal capture.</p>
        </div>
      </aside>

      <section className="main-content">
        {section === "analytics" ? <AnalyticsPlaceholder /> : null}
        {section === "calendar" ? <CalendarHistorySection store={store} /> : null}
        {section === "settings" ? <SettingsSection store={store} /> : null}
      </section>
    </main>
  );
}

function PanelSurface({ store }: { store: ReturnType<typeof useTradingPanelState> }) {
  const { state, actions } = store;
  const journal = selectCurrentJournal(state.journals);
  const reminder = useReminderClock(
    state.config.timerEnabled,
    state.config.timerInterval,
  );
  const [showStartJournal, setShowStartJournal] = useState(false);
  const [showFinishJournal, setShowFinishJournal] = useState(false);
  const [isPinned, setIsPinned] = useState(state.panelPinned);

  useEffect(() => {
    setIsPinned(state.panelPinned);
  }, [state.panelPinned]);

  useEffect(() => {
    if (!isTauriRuntime()) {
      return;
    }

    const currentWindow = getCurrentWindow();
    void currentWindow.setAlwaysOnTop(isPinned).catch(() => undefined);
  }, [isPinned]);

  usePersistPanelBounds();

  return (
    <main className="desktop-shell panel-shell">
      <section className="panel-frame">
        <header className="panel-card compact-card top-bar">
          <button
            type="button"
            className={`pin-button ${isPinned ? "is-pinned" : ""}`}
            title={
              isPinned
                ? "Always on top. Click to allow overlap."
                : "Can be covered by other windows. Click to keep on top."
            }
            onClick={() => {
              const nextPinned = !isPinned;
              setIsPinned(nextPinned);
              actions.setPanelPinned(nextPinned);
            }}
          >
            {isPinned ? "📌" : "⊙"}
          </button>

          <div className="top-bar-meta">
            <p className="top-bar-title">TRADING PANEL</p>
            <p className={`reminder-label ${state.config.timerEnabled ? "is-on" : ""}`}>
              {formatReminderStatus(
                state.config.timerEnabled,
                state.config.timerInterval,
              )}
            </p>
          </div>

          <div className={`clock-badge ${reminder.isFlashing ? "is-flashing" : ""}`}>
            {formatClock(reminder.now)}
          </div>
        </header>

        <section className="panel-card compact-card journal-summary-card">
          {journal ? (
            <>
              <div className="journal-summary-header">
                <div>
                  <div className="journal-summary-title-row">
                    <h2>{journal.symbol || "No Symbol"}</h2>
                    <span className={`status-pill ${journal.status}`}>
                      {journal.status.toUpperCase()}
                    </span>
                  </div>

                  <div className="summary-meta-row">
                    {journal.market ? <span>{journal.market}</span> : null}
                    <span>
                      Pos {formatNumber(netPosition(journal.events))}
                    </span>
                  </div>
                </div>

                {journal.status === "open" ? (
                  <button
                    type="button"
                    className="secondary-button"
                    onClick={() => setShowFinishJournal(true)}
                  >
                    End Journal
                  </button>
                ) : null}
              </div>

              {journal.summary ? <p className="summary-copy">{journal.summary}</p> : null}
            </>
          ) : (
            <div className="empty-inline-state">
              <div>
                <h2>No active journal</h2>
                <p>Create a journal to start recording trade events.</p>
              </div>
              <button
                type="button"
                className="primary-button"
                onClick={() => setShowStartJournal(true)}
              >
                Start Journal
              </button>
            </div>
          )}
        </section>

        <TimelineSection
          journal={journal}
          title="Recent Notes"
          className="panel-card panel-grow"
        />

        <QuickCaptureCard
          journal={journal}
          onCreateJournal={() => setShowStartJournal(true)}
          onRecord={(draft) =>
            journal ? store.actions.recordEvent(journal.id, draft) : { ok: false, error: "missingContent" }
          }
        />
      </section>

      <StartJournalModal
        open={showStartJournal}
        onClose={() => setShowStartJournal(false)}
        onCreate={(symbol, market) => {
          actions.createJournal(symbol, market);
          setShowStartJournal(false);
        }}
      />

      <FinishJournalModal
        open={showFinishJournal && journal !== undefined}
        initialSummary={journal?.summary ?? ""}
        onClose={() => setShowFinishJournal(false)}
        onConfirm={(summary) => {
          if (!journal) {
            return;
          }

          actions.finishJournal(journal.id, summary);
          setShowFinishJournal(false);
        }}
      />
    </main>
  );
}

function AnalyticsPlaceholder() {
  return (
    <section className="content-panel placeholder-panel">
      <p className="eyebrow">分析</p>
      <h2>分析功能待实现</h2>
      <p className="supporting-copy">Current Swift app keeps analytics as a placeholder, so the Tauri version does the same.</p>
    </section>
  );
}

function CalendarHistorySection({
  store,
}: {
  store: ReturnType<typeof useTradingPanelState>;
}) {
  const journals = store.state.journals;
  const [displayedMonth, setDisplayedMonth] = useState(() => startOfMonth(new Date()));
  const [selectedJournalId, setSelectedJournalId] = useState<string | undefined>();
  const monthDays = useMemo(() => visibleMonthDays(displayedMonth), [displayedMonth]);
  const grouped = useMemo(() => groupJournalsByDay(journals), [journals]);
  const groupedMap = useMemo(
    () => new Map(grouped.map((group) => [group.dayKey, group.journals])),
    [grouped],
  );
  const monthJournals = useMemo(
    () =>
      journals
        .filter((journal) => isSameMonth(new Date(journal.createdAt), displayedMonth))
        .sort((left, right) =>
          new Date(left.createdAt).getTime() - new Date(right.createdAt).getTime(),
        ),
    [displayedMonth, journals],
  );

  useEffect(() => {
    const selection = defaultCalendarSelection(displayedMonth, journals);

    setSelectedJournalId((currentSelection) => {
      if (!currentSelection) {
        return selection?.id;
      }

      const stillVisible = monthJournals.some((journal) => journal.id === currentSelection);
      return stillVisible ? currentSelection : selection?.id;
    });
  }, [displayedMonth, journals, monthJournals]);

  const selectedJournal = journals.find((journal) => journal.id === selectedJournalId);

  return (
    <div className="calendar-layout">
      <section className="content-panel calendar-panel">
        <div className="section-header-block">
          <h2>Calendar History</h2>
          <p>Browse archived trade journals by creation date and inspect a selected session on the right.</p>
        </div>

        <div className="month-toolbar">
          <button
            type="button"
            className="icon-button"
            onClick={() => setDisplayedMonth(addMonths(displayedMonth, -1))}
          >
            ‹
          </button>

          <div className="month-toolbar-copy">
            <strong>{displayedMonth.toLocaleDateString("en-US", { month: "long", year: "numeric", timeZone: "UTC" })}</strong>
            <span>
              {monthJournals.length === 1
                ? "1 journal in this month"
                : `${monthJournals.length} journals in this month`}
            </span>
          </div>

          <button
            type="button"
            className="icon-button"
            onClick={() => setDisplayedMonth(addMonths(displayedMonth, 1))}
          >
            ›
          </button>
        </div>

        {monthJournals.length === 0 ? (
          <div className="soft-banner">No journals were created in this month.</div>
        ) : null}

        <div className="weekday-grid">
          {WEEKDAY_SYMBOLS.map((symbol) => (
            <span key={symbol}>{symbol}</span>
          ))}
        </div>

        <div className="calendar-grid">
          {monthDays.map((day) => {
            const dayKey = formatDayKey(day.date);
            const dayJournals = groupedMap.get(dayKey) ?? [];
            const isToday = formatDayKey(day.date) === formatDayKey(new Date());

            return (
              <div
                key={dayKey}
                className={`day-cell ${day.isInDisplayedMonth ? "" : "is-outside"} ${isToday ? "is-today" : ""}`}
              >
                <span className="day-number">{day.date.getUTCDate()}</span>
                <div className="day-chip-stack">
                  {dayJournals.map((journal) => (
                    <button
                      type="button"
                      key={journal.id}
                      className={`day-chip ${selectedJournalId === journal.id ? "is-selected" : ""}`}
                      onClick={() => setSelectedJournalId(journal.id)}
                    >
                      {journal.symbol || "No Symbol"}
                    </button>
                  ))}
                </div>
              </div>
            );
          })}
        </div>
      </section>

      <JournalDetailCard journal={selectedJournal} />
    </div>
  );
}

function JournalDetailCard({ journal }: { journal: TradeJournal | undefined }) {
  if (!journal) {
    return (
      <section className="content-panel detail-panel empty-detail-panel">
        <h2>Trade Journal</h2>
        <p>Select a journal from the calendar to inspect its timeline here.</p>
      </section>
    );
  }

  return (
    <section className="detail-panel">
      <div className="content-panel detail-header-panel">
        <div className="journal-summary-title-row">
          <h2>{journal.symbol || "No Symbol"}</h2>
          <span className={`status-pill ${journal.status}`}>
            {journal.status === "open" ? "Open" : "Done"}
          </span>
        </div>

        <div className="summary-meta-row">
          {journal.market ? <span>{journal.market}</span> : null}
          <span>{formatJournalDate(journal.createdAt)}</span>
        </div>

        {journal.summary ? <p className="summary-copy">{journal.summary}</p> : null}
      </div>

      <TimelineSection journal={journal} title="Journal Timeline" className="content-panel detail-timeline-panel" />
    </section>
  );
}

function SettingsSection({
  store,
}: {
  store: ReturnType<typeof useTradingPanelState>;
}) {
  const { config } = store.state;

  return (
    <section className="content-panel settings-panel">
      <div className="section-header-block">
        <h2>Settings</h2>
        <p>Reminder settings persist locally and apply to the floating panel clock.</p>
      </div>

      <div className="settings-group">
        <label className="toggle-row">
          <span>Enable Reminder</span>
          <input
            type="checkbox"
            checked={config.timerEnabled}
            onChange={(event) => {
              store.actions.updateConfig({ timerEnabled: event.currentTarget.checked });
            }}
          />
        </label>

        <label className="field-stack">
          <span>Interval</span>
          <select
            value={config.timerInterval}
            disabled={!config.timerEnabled}
            onChange={(event) => {
              store.actions.updateConfig({ timerInterval: Number(event.currentTarget.value) });
            }}
          >
            {REMINDER_INTERVALS.map((preset) => (
              <option key={preset.value} value={preset.value}>
                {preset.label}
              </option>
            ))}
          </select>
        </label>
      </div>

      <div className="settings-group">
        <label className="field-stack">
          <span>Data</span>
          <button type="button" className="secondary-button" onClick={() => undefined}>
            Export CSV
          </button>
        </label>
      </div>
    </section>
  );
}

function TimelineSection({
  journal,
  title,
  className,
}: {
  journal: TradeJournal | undefined;
  title: string;
  className: string;
}) {
  const events = useMemo(
    () =>
      journal
        ? [...journal.events].sort(
            (left, right) =>
              new Date(right.timestamp).getTime() - new Date(left.timestamp).getTime(),
          )
        : [],
    [journal],
  );
  const highlightedId = useHighlightedTopEvent(events);

  return (
    <section className={`${className} timeline-panel`}>
      <div className="timeline-header">
        <h3>{title}</h3>
        <span className="count-pill">{events.length}</span>
      </div>

      {events.length === 0 ? (
        <div className="timeline-empty-state">
          <span className="timeline-empty-icon">≈</span>
          <strong>Ready for the next setup</strong>
          <p>Your latest trade events will appear here as a clean session timeline.</p>
        </div>
      ) : (
        <div className="timeline-list">
          {events.map((event) => (
            <article
              key={event.id}
              className={`timeline-row ${highlightedId === event.id ? "is-highlighted" : ""}`}
            >
              <span className={`accent-bar ${event.type}`} />
              <div className="timeline-row-body">
                <div className="timeline-row-topline">
                  <span className={`event-pill ${event.type}`}>
                    {EVENT_SYMBOLS[event.type]} {EVENT_LABELS[event.type]}
                  </span>
                  <strong>{journal?.symbol || "Unknown Instrument"}</strong>
                </div>

                <div className="timeline-row-meta">
                  <span>{formatClock(new Date(event.timestamp))}</span>
                  {journal?.market ? <span>{journal.market}</span> : null}
                  {event.quantity !== undefined ? (
                    <span className={`quantity-pill ${event.type}`}>
                      {formatNumber(event.quantity)}
                    </span>
                  ) : null}
                  {event.price !== undefined ? <span>@ {formatNumber(event.price)}</span> : null}
                </div>

                {event.content ? <p>{event.content}</p> : null}
              </div>
            </article>
          ))}
        </div>
      )}
    </section>
  );
}

function QuickCaptureCard({
  journal,
  onCreateJournal,
  onRecord,
}: {
  journal: TradeJournal | undefined;
  onCreateJournal: () => void;
  onRecord: (draft: TradeEventDraft) => ReturnType<typeof composeTradeEvent>;
}) {
  const currentNetPosition = journal ? netPosition(journal.events) : 0;
  const status = journal?.status ?? "open";
  const types = availableEventTypes(status, currentNetPosition);

  const [selectedType, setSelectedType] = useState<TradeEventType>(types[0] ?? "log");
  const [quantityInput, setQuantityInput] = useState("");
  const [priceInput, setPriceInput] = useState("");
  const [content, setContent] = useState("");
  const [didSubmit, setDidSubmit] = useState(false);

  useEffect(() => {
    if (!types.includes(selectedType)) {
      setSelectedType(types[0] ?? "log");
    }
  }, [selectedType, types]);

  const draft = useMemo<TradeEventDraft>(
    () => ({
      type: selectedType,
      quantityInput,
      priceInput,
      content,
    }),
    [content, priceInput, quantityInput, selectedType],
  );

  const validation = journal
    ? composeTradeEvent(draft, currentNetPosition, status)
    : ({ ok: false, error: "missingContent" } as const);

  const canSubmit = journal !== undefined && validation.ok;

  return (
    <section className="panel-card compact-card composer-card">
      <div className="composer-header">
        <div>
          <p className="eyebrow small">Quick Capture</p>
          <p className="supporting-copy small">
            {journal
              ? journal.status === "done"
                ? "This journal is done. Only logs can be added now."
                : "Capture the next event before attention moves back to the chart."
              : "Start a trade journal before capturing events."}
          </p>
        </div>
        <span className="keyboard-hint">⌘↩ Submit</span>
      </div>

      {journal ? (
        <>
          <div className="chip-row">
            {types.map((type) => (
              <button
                type="button"
                key={type}
                className={`type-chip ${type === selectedType ? "is-selected" : ""}`}
                onClick={() => setSelectedType(type)}
              >
                {EVENT_LABELS[type]}
              </button>
            ))}
          </div>

          {selectedType !== "log" ? (
            <div className="field-grid two-up">
              <label className="field-stack">
                <span>Quantity</span>
                {selectedType === "exit" ? (
                  <div className="readonly-field">{formatNumber(-currentNetPosition)}</div>
                ) : (
                  <input
                    value={quantityInput}
                    placeholder="e.g. 1.25 or -1.25"
                    onChange={(event) => setQuantityInput(event.currentTarget.value)}
                  />
                )}
              </label>

              <label className="field-stack">
                <span>Price</span>
                <input
                  value={priceInput}
                  placeholder="Optional"
                  onChange={(event) => setPriceInput(event.currentTarget.value)}
                />
              </label>
            </div>
          ) : null}

          {selectedType !== "log" ? (
            <p className="hint-text compact">{quantityHint(selectedType, status, currentNetPosition)}</p>
          ) : null}

          <label className="field-stack">
            <span>{selectedType === "log" ? "Log" : "Note"}</span>
            <textarea
              value={content}
              placeholder={selectedType === "log" ? "What changed, what did you see, what is the risk?" : "Optional context"}
              rows={3}
              onChange={(event) => setContent(event.currentTarget.value)}
            />
          </label>

          <div className="composer-footer">
            <p className={`feedback-text ${didSubmit ? "is-success" : ""}`}>
              {didSubmit
                ? "Recorded"
                : feedbackMessage(selectedType, journal, validation, canSubmit)}
            </p>

            <button
              type="button"
              className="primary-button"
              disabled={!canSubmit}
              onClick={() => {
                const result = onRecord(draft);
                if (!result.ok) {
                  return;
                }

                setDidSubmit(true);
                setSelectedType("log");
                setQuantityInput("");
                setPriceInput("");
                setContent("");

                window.setTimeout(() => {
                  setDidSubmit(false);
                }, 1200);
              }}
            >
              {didSubmit ? "Recorded" : "Record Event"}
            </button>
          </div>
        </>
      ) : (
        <div className="empty-composer-state">
          <p>Create a journal before adding events.</p>
          <button type="button" className="primary-button" onClick={onCreateJournal}>
            Start Journal
          </button>
        </div>
      )}
    </section>
  );
}

function StartJournalModal({
  open,
  onClose,
  onCreate,
}: {
  open: boolean;
  onClose: () => void;
  onCreate: (symbol: string, market: string) => void;
}) {
  const [symbol, setSymbol] = useState("");
  const [market, setMarket] = useState("");

  useEffect(() => {
    if (!open) {
      setSymbol("");
      setMarket("");
    }
  }, [open]);

  if (!open) {
    return null;
  }

  return (
    <div className="modal-backdrop" role="presentation">
      <section className="modal-card" role="dialog" aria-modal="true" aria-label="Start Journal">
        <h3>Start Journal</h3>
        <label className="field-stack">
          <span>Symbol</span>
          <input value={symbol} onChange={(event) => setSymbol(event.currentTarget.value)} />
        </label>
        <label className="field-stack">
          <span>Market</span>
          <input value={market} onChange={(event) => setMarket(event.currentTarget.value)} />
        </label>
        <div className="modal-actions">
          <button type="button" className="text-button" onClick={onClose}>
            Cancel
          </button>
          <button
            type="button"
            className="primary-button"
            disabled={symbol.trim().length === 0}
            onClick={() => onCreate(symbol, market)}
          >
            Create
          </button>
        </div>
      </section>
    </div>
  );
}

function FinishJournalModal({
  open,
  initialSummary,
  onClose,
  onConfirm,
}: {
  open: boolean;
  initialSummary: string;
  onClose: () => void;
  onConfirm: (summary: string) => void;
}) {
  const [summary, setSummary] = useState(initialSummary);

  useEffect(() => {
    setSummary(initialSummary);
  }, [initialSummary]);

  if (!open) {
    return null;
  }

  return (
    <div className="modal-backdrop" role="presentation">
      <section className="modal-card" role="dialog" aria-modal="true" aria-label="Finish Journal">
        <h3>Finish Journal</h3>
        <label className="field-stack">
          <span>Summary</span>
          <textarea
            rows={4}
            value={summary}
            placeholder="Optional summary"
            onChange={(event) => setSummary(event.currentTarget.value)}
          />
        </label>
        <div className="modal-actions">
          <button type="button" className="text-button" onClick={onClose}>
            Cancel
          </button>
          <button type="button" className="primary-button" onClick={() => onConfirm(summary)}>
            Confirm
          </button>
        </div>
      </section>
    </div>
  );
}

function SidebarButton({
  active,
  icon,
  label,
  onClick,
}: {
  active: boolean;
  icon: string;
  label: string;
  onClick: () => void;
}) {
  return (
    <button type="button" className={`sidebar-button ${active ? "is-active" : ""}`} onClick={onClick}>
      <span>{icon}</span>
      <span>{label}</span>
    </button>
  );
}

function useHighlightedTopEvent(events: TradeEvent[]) {
  const previousIds = useRef<string[]>([]);
  const [highlightedId, setHighlightedId] = useState<string>();

  useEffect(() => {
    const currentIds = events.map((event) => event.id);
    const currentTop = currentIds[0];

    if (
      currentTop &&
      currentTop !== previousIds.current[0] &&
      !previousIds.current.includes(currentTop)
    ) {
      setHighlightedId(currentTop);
      const timer = window.setTimeout(() => setHighlightedId(undefined), 1400);
      previousIds.current = currentIds;
      return () => {
        window.clearTimeout(timer);
      };
    }

    previousIds.current = currentIds;
    return undefined;
  }, [events]);

  return highlightedId;
}

function useReminderClock(enabled: boolean, interval: number) {
  const [now, setNow] = useState(() => new Date());
  const [isFlashing, setIsFlashing] = useState(false);

  useEffect(() => {
    const timer = window.setInterval(() => {
      setNow(new Date());
    }, 1000);

    return () => {
      window.clearInterval(timer);
    };
  }, []);

  useEffect(() => {
    if (!enabled) {
      setIsFlashing(false);
      return;
    }

    const intervalMs = interval * 1000;
    const delay = intervalMs - (Date.now() % intervalMs);
    let repeatingTimer = 0;

    const trigger = () => {
      setIsFlashing(true);
      window.setTimeout(() => setIsFlashing(false), 1000);
    };

    const initialTimer = window.setTimeout(() => {
      trigger();
      repeatingTimer = window.setInterval(trigger, intervalMs);
    }, delay);

    return () => {
      window.clearTimeout(initialTimer);
      if (repeatingTimer) {
        window.clearInterval(repeatingTimer);
      }
    };
  }, [enabled, interval]);

  return { now, isFlashing };
}

function usePersistPanelBounds() {
  useEffect(() => {
    if (!isTauriRuntime()) {
      return;
    }

    const currentWindow = getCurrentWindow();
    const persistBounds = async () => {
      const [position, size] = await Promise.all([
        currentWindow.outerPosition(),
        currentWindow.outerSize(),
      ]);

      window.localStorage.setItem(
        PANEL_BOUNDS_STORAGE_KEY,
        serializePanelBounds({
          x: position.x,
          y: position.y,
          width: size.width,
          height: size.height,
        }),
      );
    };

    void persistBounds().catch(() => undefined);
    const unsubscribers: Array<() => void> = [];

    void currentWindow
      .onMoved(() => {
        void persistBounds().catch(() => undefined);
      })
      .then((unlisten) => {
        unsubscribers.push(unlisten);
      });

    void currentWindow
      .onResized(() => {
        void persistBounds().catch(() => undefined);
      })
      .then((unlisten) => {
        unsubscribers.push(unlisten);
      });

    return () => {
      unsubscribers.forEach((unlisten) => unlisten());
    };
  }, []);
}

function feedbackMessage(
  selectedType: TradeEventType,
  journal: TradeJournal,
  validation: ReturnType<typeof composeTradeEvent>,
  canSubmit: boolean,
) {
  if (journal.status === "done") {
    return "Done journals only accept logs.";
  }

  if (!canSubmit) {
    return validation.ok
      ? "Complete the required fields to record this event."
      : validationMessage(selectedType, validation.error);
  }

  if (selectedType === "exit") {
    return "Ready to record exit.";
  }

  if (selectedType === "log") {
    return "Ready to record log.";
  }

  return "Ready to record event.";
}

function validationMessage(
  type: TradeEventType,
  error: TradeEventComposerError,
) {
  switch (error) {
    case "missingContent":
      return type === "log"
        ? "Log requires text."
        : "Complete the required fields to record this event.";
    case "missingQuantity":
      return type === "entry"
        ? "Enter a signed quantity to open the position."
        : "Enter a quantity before recording this event.";
    case "invalidPrecision":
      return "Use no more than two decimal places.";
    case "zeroQuantity":
      return "Quantity cannot be zero.";
    case "wrongDirection":
      return type === "add"
        ? "Add must stay in the same direction as the current position."
        : "Reduce must move back toward zero.";
    case "exceedsPosition":
      return "Reduce cannot exceed the remaining position.";
    case "doneOnlyLogs":
      return "Done journals only accept logs.";
  }
}

function quantityHint(
  type: TradeEventType,
  status: TradeJournal["status"],
  currentNetPosition: number,
) {
  if (status === "done") {
    return "Done journals only accept logs.";
  }

  switch (type) {
    case "entry":
      return "Positive = long, negative = short. Max two decimals.";
    case "add":
      return currentNetPosition < 0
        ? "Add must stay negative to extend the short."
        : "Add must stay positive to extend the long.";
    case "reduce":
      return currentNetPosition < 0
        ? "Reduce must be positive to cover the short."
        : "Reduce must be negative to trim the long.";
    case "exit":
      return "Exit uses the full remaining position automatically.";
    case "log":
      return "";
  }
}

function formatClock(date: Date) {
  return date.toLocaleTimeString("en-GB", {
    hour12: false,
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function formatReminderStatus(enabled: boolean, interval: number) {
  return enabled ? `On · ${formatReminderInterval(interval)}` : "Off";
}

function formatReminderInterval(interval: number) {
  const match = REMINDER_INTERVALS.find((preset) => preset.value === interval);
  return match ? match.label : `${Math.round(interval / 60)}m`;
}

function formatJournalDate(value: string) {
  return new Date(value).toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

function formatNumber(value: number) {
  return value.toLocaleString("en-US", {
    maximumFractionDigits: 2,
    minimumFractionDigits: isZero(value % 1) ? 0 : 0,
  });
}

function addMonths(date: Date, delta: number) {
  return new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth() + delta, 1));
}

function startOfMonth(date: Date) {
  return new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), 1));
}

function isSameMonth(left: Date, right: Date) {
  return (
    left.getUTCFullYear() === right.getUTCFullYear() &&
    left.getUTCMonth() === right.getUTCMonth()
  );
}

function formatDayKey(date: Date) {
  const year = date.getUTCFullYear();
  const month = `${date.getUTCMonth() + 1}`.padStart(2, "0");
  const day = `${date.getUTCDate()}`.padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function isTauriRuntime(): boolean {
  return typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;
}

const REMINDER_INTERVALS = [
  { label: "1m", value: 60 },
  { label: "5m", value: 300 },
  { label: "15m", value: 900 },
  { label: "30m", value: 1800 },
  { label: "1h", value: 3600 },
];

const EVENT_LABELS: Record<TradeEventType, string> = {
  entry: "Entry",
  add: "Add",
  reduce: "Reduce",
  exit: "Exit",
  log: "Log",
};

const EVENT_SYMBOLS: Record<TradeEventType, string> = {
  entry: "↗",
  add: "+",
  reduce: "−",
  exit: "×",
  log: "☰",
};

const WEEKDAY_SYMBOLS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
