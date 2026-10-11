#pragma once
#include <cstdint>
#include <string>
#include <utility>

// Money is a fixed-point amount in minor units with an ISO 4217 currency code.
class Money {
 public:
  Money(std::int64_t minor, std::string currency)
      : minor_(minor), currency_(std::move(currency)) {}

  bool operator==(const Money& other) const {
    return minor_ == other.minor_ && currency_ == other.currency_;
  }

  std::int64_t minor() const { return minor_; }
  const std::string& currency() const { return currency_; }

 private:
  std::int64_t minor_;
  std::string currency_;
};
