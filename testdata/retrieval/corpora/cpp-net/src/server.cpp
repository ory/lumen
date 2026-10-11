#include "net/socket_handle.hpp"
#include "net/socket_view.hpp"

#include <sys/socket.h>
#include <vector>

namespace net {

std::vector<SocketHandle> accept_pending(const SocketHandle& listener, int max_clients) {
    std::vector<SocketHandle> clients;
    SocketView view(listener);
    for (int i = 0; i < max_clients; ++i) {
        const int fd = ::accept(view.get(), nullptr, nullptr);
        if (fd < 0) {
            break;
        }
        clients.emplace_back(fd);
    }
    return clients;
}

}  // namespace net
