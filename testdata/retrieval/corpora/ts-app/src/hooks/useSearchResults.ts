import { useEffect, useState } from 'react';
import { fetchSearchResults, type SearchResult } from '../api/searchClient';

export interface SearchState {
  results: SearchResult[];
  error: Error | null;
  loading: boolean;
}

export function useSearchResults(query: string): SearchState {
  const [results, setResults] = useState<SearchResult[]>([]);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!query) {
      setResults([]);
      return;
    }

    const controller = new AbortController();
    setLoading(true);
    setError(null);

    fetchSearchResults(query, controller.signal)
      .then(setResults)
      .catch((err: unknown) => {
        if (err instanceof DOMException && err.name === 'AbortError') {
          return;
        }
        setError(err as Error);
      })
      .finally(() => setLoading(false));

    return () => controller.abort();
  }, [query]);

  return { results, error, loading };
}
