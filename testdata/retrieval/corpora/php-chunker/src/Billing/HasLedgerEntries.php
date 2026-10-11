<?php

namespace App\Billing;

trait HasLedgerEntries
{
    protected array $ledgerEntries = [];

    public function recordLedgerEntry(string $account, int $amountCents): void
    {
        $this->ledgerEntries[] = [
            'account' => $account,
            'amount' => $amountCents,
            'recorded_at' => time(),
        ];
    }

    public function ledgerBalance(): int
    {
        return array_sum(array_column($this->ledgerEntries, 'amount'));
    }
}
