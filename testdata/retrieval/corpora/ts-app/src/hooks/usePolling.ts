import { useEffect, useRef } from 'react';

export function usePolling(callback: () => Promise<void>, intervalMs: number, enabled = true): void {
  const saved = useRef(callback);
  saved.current = callback;

  useEffect(() => {
    if (!enabled) {
      return;
    }
    let cancelled = false;
    const id = setInterval(() => {
      if (!cancelled) {
        void saved.current();
      }
    }, intervalMs);
    return () => {
      cancelled = true;
      clearInterval(id);
    };
  }, [intervalMs, enabled]);
}
