<?php

namespace App\Billing\Gateways;

use App\Billing\Invoice;

class StripeGateway
{
    public function __construct(private string $secretKey)
    {
    }

    public function charge(Invoice $invoice, string $paymentMethod): string
    {
        $payload = [
            'amount' => $invoice->total(),
            'payment_method' => $paymentMethod,
            'metadata' => ['invoice' => $invoice->reference],
        ];

        return hash_hmac('sha256', json_encode($payload), $this->secretKey);
    }
}
