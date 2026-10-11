#include <stdio.h>
#include <string.h>

#include "net.h"
#include "records.h"

int main(int argc, char **argv) {
    struct record queue[3] = {
        {1, 5, "backup"},
        {2, 9, "deploy"},
        {3, 5, "audit"},
    };
    const char *host = argc > 1 ? argv[1] : "localhost";
    int fd;

    sort_records(queue, 3, NULL);
    fd = open_socket(host, 8080);
    if (fd < 0) {
        fprintf(stderr, "connect to %s failed\n", host);
        return 1;
    }
    for (size_t i = 0; i < 3; i++) {
        send_all(fd, queue[i].name, strlen(queue[i].name));
    }
    close_socket(fd);
    return 0;
}
