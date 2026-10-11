import formatStatement, { type LineItem } from './templates/statement.js';

export function issueInvoice(customer: string, lines: LineItem[]): { customer: string; text: string } {
  return { customer, text: formatStatement(customer, lines) };
}
