#include "codec/header.hpp"

#include <algorithm>
#include <cctype>

namespace {

bool iequals(const std::string& a, const std::string& b) {
    return a.size() == b.size() &&
           std::equal(a.begin(), a.end(), b.begin(), [](unsigned char x, unsigned char y) {
               return std::tolower(x) == std::tolower(y);
           });
}

}  // namespace

// Header names compare case-insensitively (RFC 9110); values compare exactly.
bool codec::operator==(const Header& lhs, const Header& rhs) noexcept {
    return iequals(lhs.name, rhs.name) && lhs.value == rhs.value;
}

bool codec::operator!=(const Header& lhs, const Header& rhs) noexcept {
    return !(lhs == rhs);
}
