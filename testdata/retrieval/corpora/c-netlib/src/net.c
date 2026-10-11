#include "net.h"

#include <stdio.h>
#include <string.h>

#ifdef _WIN32
#include <winsock2.h>
#include <ws2tcpip.h>
#pragma comment(lib, "ws2_32.lib")
#else
#include <fcntl.h>
#include <netdb.h>
#include <sys/socket.h>
#include <unistd.h>
#endif

#ifdef _WIN32
static int winsock_ready = 0;

static int ensure_winsock(void) {
    WSADATA wsa;
    if (winsock_ready) {
        return 0;
    }
    if (WSAStartup(MAKEWORD(2, 2), &wsa) != 0) {
        return -1;
    }
    winsock_ready = 1;
    return 0;
}

/* Windows: Winsock must be initialised before the first socket call. */
int open_socket(const char *host, int port) {
    struct addrinfo hints, *res, *ai;
    char service[16];
    SOCKET s = INVALID_SOCKET;

    if (ensure_winsock() != 0) {
        return -1;
    }
    memset(&hints, 0, sizeof hints);
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    _snprintf(service, sizeof service, "%d", port);
    if (getaddrinfo(host, service, &hints, &res) != 0) {
        return -1;
    }
    for (ai = res; ai != NULL; ai = ai->ai_next) {
        s = socket(ai->ai_family, ai->ai_socktype, ai->ai_protocol);
        if (s == INVALID_SOCKET) {
            continue;
        }
        if (connect(s, ai->ai_addr, (int)ai->ai_addrlen) == 0) {
            break;
        }
        closesocket(s);
        s = INVALID_SOCKET;
    }
    freeaddrinfo(res);
    return s == INVALID_SOCKET ? -1 : (int)s;
}

void close_socket(int fd) {
    closesocket((SOCKET)fd);
}
#else
/* POSIX: the descriptor is marked close-on-exec so children do not inherit it. */
int open_socket(const char *host, int port) {
    struct addrinfo hints, *res, *ai;
    char service[16];
    int fd = -1;

    memset(&hints, 0, sizeof hints);
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    snprintf(service, sizeof service, "%d", port);
    if (getaddrinfo(host, service, &hints, &res) != 0) {
        return -1;
    }
    for (ai = res; ai != NULL; ai = ai->ai_next) {
        fd = socket(ai->ai_family, ai->ai_socktype, ai->ai_protocol);
        if (fd < 0) {
            continue;
        }
        fcntl(fd, F_SETFD, FD_CLOEXEC);
        if (connect(fd, ai->ai_addr, ai->ai_addrlen) == 0) {
            break;
        }
        close(fd);
        fd = -1;
    }
    freeaddrinfo(res);
    return fd;
}

void close_socket(int fd) {
    close(fd);
}
#endif

ssize_t send_all(int fd, const void *buf, size_t len) {
    const char *p = buf;
    size_t left = len;
    while (left > 0) {
        ssize_t n = send(fd, p, left, 0);
        if (n <= 0) {
            return -1;
        }
        p += n;
        left -= (size_t)n;
    }
    return (ssize_t)len;
}
