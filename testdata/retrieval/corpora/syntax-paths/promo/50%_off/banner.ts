// Half-price campaign banner shown on the storefront.
export function renderHalfPriceBanner(endsAt: Date): string {
  return `<aside class="promo">Everything half price until ${endsAt.toDateString()}</aside>`;
}
