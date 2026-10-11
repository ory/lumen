import { useEffect, useState } from 'react';
import { fetchSuggestions } from '../api/searchClient';

export const useSearchSuggestions = (prefix: string, limit = 8): string[] => {
  const [suggestions, setSuggestions] = useState<string[]>([]);

  useEffect(() => {
    if (prefix.trim().length < 2) {
      setSuggestions([]);
      return;
    }

    const controller = new AbortController();
    fetchSuggestions(prefix, controller.signal)
      .then((items) => setSuggestions(items.slice(0, limit)))
      .catch(() => setSuggestions([]));

    return () => {
      controller.abort();
    };
  }, [prefix, limit]);

  return suggestions;
};
