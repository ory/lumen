<?php

namespace App\Billing;

enum Currency: string
{
    case Euro = 'EUR';
    case Dollar = 'USD';
    case Yen = 'JPY';

    public function minorUnits(): int
    {
        return match ($this) {
            self::Yen => 0,
            default => 2,
        };
    }

    public function symbol(): string
    {
        return match ($this) {
            self::Euro => '€',
            self::Dollar => '$',
            self::Yen => '¥',
        };
    }
}
