export interface SearchResult {
  id: string;
  title: string;
  url: string;
  score: number;
}

const SEARCH_ENDPOINT = '/api/search';
const SUGGEST_ENDPOINT = '/api/suggest';

export async function fetchSearchResults(query: string, signal?: AbortSignal): Promise<SearchResult[]> {
  const response = await fetch(`${SEARCH_ENDPOINT}?q=${encodeURIComponent(query)}`, { signal });
  if (!response.ok) {
    throw new Error(`search failed with status ${response.status}`);
  }
  return (await response.json()) as SearchResult[];
}

export async function fetchSuggestions(prefix: string, signal?: AbortSignal): Promise<string[]> {
  const response = await fetch(`${SUGGEST_ENDPOINT}?prefix=${encodeURIComponent(prefix)}`, { signal });
  if (!response.ok) {
    return [];
  }
  const body = (await response.json()) as { suggestions: string[] };
  return body.suggestions;
}
