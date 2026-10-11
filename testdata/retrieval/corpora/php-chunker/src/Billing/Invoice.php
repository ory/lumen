<?php

namespace App\Billing;

class Invoice
{
    use HasLedgerEntries;

    const STATUS_OPEN = 'open';
    const STATUS_PAID = 'paid';
    const STATUS_VOID = 'void';

    public string $reference;

    protected array $lineItems = [];

    private string $status = self::STATUS_OPEN;

    public function __construct(string $reference, private Currency $currency)
    {
        $this->reference = $reference;
    }

    public static function fromRow(array $row): self
    {
        $invoice = new self($row['reference'], Currency::from($row['currency']));
        $invoice->status = $row['status'];

        return $invoice;
    }

    public function addLineItem(string $description, int $quantity, int $unitPriceCents): void
    {
        $this->lineItems[] = compact('description', 'quantity', 'unitPriceCents');
    }

    public function total(): int
    {
        return array_sum(array_map(
            fn (array $item) => $item['quantity'] * $item['unitPriceCents'],
            $this->lineItems,
        ));
    }

    public function void(): void
    {
        $this->status = self::STATUS_VOID;
        $this->recordLedgerEntry('receivables', -$this->total());
    }

    public function status(): string
    {
        return $this->status;
    }
}
