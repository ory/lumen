#ifndef NETLIB_RECORDS_H
#define NETLIB_RECORDS_H

#include <stddef.h>

#define RECORD_NAME_LEN 32

struct record {
    unsigned int id;
    int priority;
    char name[RECORD_NAME_LEN];
};

/* qsort-compatible ordering callback. */
typedef int (*CompareFn)(const void *lhs, const void *rhs);

void sort_records(struct record *records, size_t count, CompareFn cmp);
void sort_records_by_name(struct record *records, size_t count);
const struct record *find_record(const struct record *records, size_t count, unsigned int id);

#endif /* NETLIB_RECORDS_H */
