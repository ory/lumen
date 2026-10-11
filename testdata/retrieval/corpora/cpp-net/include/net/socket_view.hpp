#pragma once

#include "net/socket_handle.hpp"

namespace net {

// Borrows a descriptor owned elsewhere. Copying is cheap and the descriptor is never closed.
class SocketView {
public:
    explicit SocketView(int fd) noexcept : fd_(fd) {}
    SocketView(const SocketHandle& handle) noexcept : fd_(handle.get()) {}

    int get() const noexcept { return fd_; }
    bool valid() const noexcept { return fd_ >= 0; }
    void shutdown_both() const;

private:
    int fd_;
};

// Non-owning wrapper used by the poller; it only records readiness for a descriptor.
struct PollEntry {
    SocketView socket;
    short events = 0;
    short revents = 0;
};

}  // namespace net
