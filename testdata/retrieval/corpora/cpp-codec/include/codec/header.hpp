#pragma once

#include <string>

namespace codec {

struct Header {
    std::string name;
    std::string value;
};

bool operator==(const Header& lhs, const Header& rhs) noexcept;
bool operator!=(const Header& lhs, const Header& rhs) noexcept;

}  // namespace codec
