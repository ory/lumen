#include "net/socket_view.hpp"

#include <sys/socket.h>

namespace net {

void SocketView::shutdown_both() const {
    if (fd_ >= 0) {
        ::shutdown(fd_, SHUT_RDWR);
    }
}

}  // namespace net
