#include <cmath>
#include <cstdint>

// roundHalfEven rounds a fractional minor-unit amount using banker's rounding.
std::int64_t roundHalfEven(double value) {
  double floored = std::floor(value);
  double diff = value - floored;
  if (diff > 0.5) return static_cast<std::int64_t>(floored) + 1;
  if (diff < 0.5) return static_cast<std::int64_t>(floored);
  std::int64_t f = static_cast<std::int64_t>(floored);
  return (f % 2 == 0) ? f : f + 1;
}
