// Anonymous analytics visit identifier, unrelated to the login session.
export function visitId(): string {
  let id = sessionStorage.getItem('visit');
  if (!id) {
    id = crypto.randomUUID();
    sessionStorage.setItem('visit', id);
  }
  return id;
}
