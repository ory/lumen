#pragma once

#include <cstddef>
#include <optional>
#include <string_view>
#include <vector>

namespace codec::hex {

std::optional<std::vector<std::byte>> decode(std::string_view text);

}  // namespace codec::hex
