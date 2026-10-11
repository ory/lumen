// Default transport for evergreen browsers.
export async function sendFetchRequest(url: string, body: string): Promise<string> {
  const res = await fetch(url, { method: 'POST', body });
  if (!res.ok) {
    throw new Error(`request to ${url} failed: ${res.status}`);
  }
  return res.text();
}
