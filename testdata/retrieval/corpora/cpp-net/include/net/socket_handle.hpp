#pragma once

#include <utility>

namespace net {

// Owns a socket descriptor. Move-only; the descriptor is released on destruction.
class SocketHandle {
public:
    SocketHandle() = default;
    explicit SocketHandle(int fd) noexcept : fd_(fd) {}
    ~SocketHandle();

    SocketHandle(const SocketHandle&) = delete;
    SocketHandle& operator=(const SocketHandle&) = delete;
    SocketHandle(SocketHandle&& other) noexcept : fd_(std::exchange(other.fd_, -1)) {}
    SocketHandle& operator=(SocketHandle&& other) noexcept;

    int get() const noexcept { return fd_; }
    int release() noexcept { return std::exchange(fd_, -1); }
    void reset(int fd = -1) noexcept;

private:
    int fd_ = -1;
};

}  // namespace net
