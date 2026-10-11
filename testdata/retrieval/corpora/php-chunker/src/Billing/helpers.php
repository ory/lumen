<?php

namespace App\Billing;

// function legacy_round_minor(float $amount): int
// {
//     return (int) floor($amount * 100);
// }

function round_minor(float $amount, Currency $currency): int
{
    $factor = 10 ** $currency->minorUnits();

    return (int) round($amount * $factor, 0, PHP_ROUND_HALF_EVEN);
}

function format_money(int $minor, Currency $currency): string
{
    $divisor = 10 ** $currency->minorUnits();

    return $currency->symbol().number_format($minor / $divisor, $currency->minorUnits());
}
