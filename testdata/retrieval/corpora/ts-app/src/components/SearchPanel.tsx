import { useState } from 'react';
import { useDebouncedValue } from '../hooks/useDebouncedValue';
import { useSearchResults } from '../hooks/useSearchResults';
import { useSearchSuggestions } from '../hooks/useSearchSuggestions';
import { ErrorBanner } from './ErrorBanner';

export function SearchPanel() {
  const [input, setInput] = useState('');
  const [attempt, setAttempt] = useState(0);
  const query = useDebouncedValue(input);
  const suggestions = useSearchSuggestions(input);
  const { results, error, loading } = useSearchResults(query);

  return (
    <section>
      <input value={input} onChange={(e) => setInput(e.target.value)} list="search-suggestions" />
      <datalist id="search-suggestions">
        {suggestions.map((s) => (
          <option key={s} value={s} />
        ))}
      </datalist>
      {error && <ErrorBanner error={error} attempt={attempt} onRetry={() => setAttempt(attempt + 1)} />}
      {loading ? <p>Loading…</p> : (
        <ul>
          {results.map((r) => (
            <li key={r.id}><a href={r.url}>{r.title}</a></li>
          ))}
        </ul>
      )}
    </section>
  );
}
