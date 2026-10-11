#include "codec/base64.hpp"

#include <cstdint>

namespace {

constexpr std::string_view kAlphabet =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

int sextet(char c) {
    const auto pos = kAlphabet.find(c);
    return pos == std::string_view::npos ? -1 : static_cast<int>(pos);
}

}  // namespace

std::optional<std::vector<std::byte>> codec::decode(std::string_view text) {
    std::vector<std::byte> out;
    out.reserve(text.size() / 4 * 3);
    std::uint32_t buffer = 0;
    int bits = 0;
    for (char c : text) {
        if (c == '=') {
            break;
        }
        const int value = sextet(c);
        if (value < 0) {
            return std::nullopt;
        }
        buffer = (buffer << 6) | static_cast<std::uint32_t>(value);
        bits += 6;
        if (bits >= 8) {
            bits -= 8;
            out.push_back(static_cast<std::byte>((buffer >> bits) & 0xFF));
        }
    }
    return out;
}

std::optional<std::vector<std::byte>> codec::decode(std::span<const std::byte> encoded) {
    const auto* chars = reinterpret_cast<const char*>(encoded.data());
    return codec::decode(std::string_view(chars, encoded.size()));
}

std::string codec::encode(std::span<const std::byte> data) {
    std::string out;
    out.reserve((data.size() + 2) / 3 * 4);
    std::uint32_t buffer = 0;
    int bits = 0;
    for (std::byte b : data) {
        buffer = (buffer << 8) | std::to_integer<std::uint32_t>(b);
        bits += 8;
        while (bits >= 6) {
            bits -= 6;
            out.push_back(kAlphabet[(buffer >> bits) & 0x3F]);
        }
    }
    if (bits > 0) {
        out.push_back(kAlphabet[(buffer << (6 - bits)) & 0x3F]);
    }
    while (out.size() % 4 != 0) {
        out.push_back('=');
    }
    return out;
}
