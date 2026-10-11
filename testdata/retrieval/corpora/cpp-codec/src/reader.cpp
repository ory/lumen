#include "codec/base64.hpp"
#include "codec/header.hpp"

#include <span>

namespace app {

std::optional<std::vector<std::byte>> read_attachment(const codec::Header& header,
                                                      std::span<const std::byte> body) {
    if (header != codec::Header{"Content-Transfer-Encoding", "base64"}) {
        return std::vector<std::byte>(body.begin(), body.end());
    }
    return codec::decode(body);
}

}  // namespace app
