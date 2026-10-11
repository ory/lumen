<?php

namespace App\Billing;

use DateTimeImmutable;
use PDO;

class DatabaseInvoiceRepository implements InvoiceRepository
{
    public function __construct(private PDO $pdo)
    {
    }

    public function find(string $reference): ?Invoice
    {
        $statement = $this->pdo->prepare('SELECT * FROM invoices WHERE reference = ?');
        $statement->execute([$reference]);
        $row = $statement->fetch(PDO::FETCH_ASSOC);

        return $row ? Invoice::fromRow($row) : null;
    }

    public function findOverdue(DateTimeImmutable $asOf): array
    {
        $statement = $this->pdo->prepare('SELECT * FROM invoices WHERE due_at < ? AND status = ?');
        $statement->execute([$asOf->format('Y-m-d'), Invoice::STATUS_OPEN]);

        return array_map([Invoice::class, 'fromRow'], $statement->fetchAll(PDO::FETCH_ASSOC));
    }

    public function save(Invoice $invoice): void
    {
        $statement = $this->pdo->prepare('REPLACE INTO invoices (reference, status, total) VALUES (?, ?, ?)');
        $statement->execute([$invoice->reference, $invoice->status(), $invoice->total()]);
    }
}
