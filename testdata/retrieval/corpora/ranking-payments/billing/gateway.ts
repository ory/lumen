export interface PaymentResult {
  id: string;
  status: 'captured' | 'declined';
}

// Calls the card network through the acquiring bank.
export async function chargeCard(token: string, amountCents: number): Promise<PaymentResult> {
  const res = await fetch('https://acquirer.example/charges', {
    method: 'POST',
    body: JSON.stringify({ token, amountCents }),
  });
  return res.json();
}
