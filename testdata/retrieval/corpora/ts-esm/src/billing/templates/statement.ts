export interface LineItem {
  description: string;
  amountCents: number;
}

export default function formatStatement(customer: string, lines: LineItem[]): string {
  const total = lines.reduce((sum, line) => sum + line.amountCents, 0);
  const body = lines.map((line) => `${line.description}\t${(line.amountCents / 100).toFixed(2)}`).join('\n');
  return `Statement for ${customer}\n${body}\nTotal\t${(total / 100).toFixed(2)}`;
}
