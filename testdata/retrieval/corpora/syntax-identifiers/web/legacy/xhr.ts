// Transport used by browsers without fetch support.
export function sendXMLHTTPRequest(url: string, body: string): Promise<string> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open('POST', url);
    xhr.onload = () => resolve(xhr.responseText);
    xhr.onerror = () => reject(new Error(`request to ${url} failed`));
    xhr.send(body);
  });
}
