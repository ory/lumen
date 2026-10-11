<?php

namespace App\Billing;

use DateTimeImmutable;

interface InvoiceRepository
{
    public function find(string $reference): ?Invoice;

    /**
     * @return Invoice[]
     */
    public function findOverdue(DateTimeImmutable $asOf): array;

    public function save(Invoice $invoice): void;
}
