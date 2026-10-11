import { readFileSync, writeFileSync } from 'node:fs';

// Joins third-party vendor files into one script, kept for the old pipeline.
export function concatenateVendorFiles(files: string[], out: string): void {
  writeFileSync(out, files.map((f) => readFileSync(f, 'utf8')).join(';\n'));
}
