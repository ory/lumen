#ifndef NETLIB_NET_H
#define NETLIB_NET_H

#include <stddef.h>
#include <sys/types.h>

#define NET_CONNECT_TIMEOUT_MS 5000
#define NET_MAX_HOSTNAME 255

/* Resolves host and opens a connected TCP socket. Returns the descriptor or -1. */
int open_socket(const char *host, int port);

/* Releases a descriptor returned by open_socket. */
void close_socket(int fd);

/* Writes the whole buffer, retrying short writes. Returns len or -1. */
ssize_t send_all(int fd, const void *buf, size_t len);

#endif /* NETLIB_NET_H */
