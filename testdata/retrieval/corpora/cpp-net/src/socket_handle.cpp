#include "net/socket_handle.hpp"

#include <unistd.h>

namespace net {

SocketHandle::~SocketHandle() {
    if (fd_ >= 0) {
        ::close(fd_);
    }
}

SocketHandle& SocketHandle::operator=(SocketHandle&& other) noexcept {
    if (this != &other) {
        reset(std::exchange(other.fd_, -1));
    }
    return *this;
}

void SocketHandle::reset(int fd) noexcept {
    if (fd_ >= 0 && fd_ != fd) {
        ::close(fd_);
    }
    fd_ = fd;
}

}  // namespace net
