#pragma once

#include <cstddef>
#include <optional>
#include <span>
#include <string>
#include <string_view>
#include <vector>

namespace codec {

// Decodes base64 text (RFC 4648, padding optional). Returns std::nullopt on bad input.
std::optional<std::vector<std::byte>> decode(std::string_view text);

// Decodes a base64 payload that arrived as raw bytes, e.g. straight from a socket buffer.
std::optional<std::vector<std::byte>> decode(std::span<const std::byte> encoded);

std::string encode(std::span<const std::byte> data);

}  // namespace codec
