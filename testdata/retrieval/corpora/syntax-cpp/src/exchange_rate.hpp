#pragma once
#include <string>

// ExchangeRate converts between two currencies at a quoted mid-market rate.
class ExchangeRate {
 public:
  ExchangeRate(std::string from, std::string to, double rate)
      : from_(from), to_(to), rate_(rate) {}

  bool operator!=(const ExchangeRate& other) const {
    return from_ != other.from_ || to_ != other.to_ || rate_ != other.rate_;
  }

  bool operator<(const ExchangeRate& other) const { return rate_ < other.rate_; }

  double apply(double amount) const { return amount * rate_; }

 private:
  std::string from_;
  std::string to_;
  double rate_;
};
