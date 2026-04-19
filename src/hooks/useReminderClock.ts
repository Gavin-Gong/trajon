import { useEffect, useState } from "react";

export function useReminderClock(enabled: boolean, interval: number) {
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
