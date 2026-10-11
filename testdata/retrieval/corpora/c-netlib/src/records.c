#include "records.h"

#include <stdlib.h>
#include <string.h>

/* Highest priority first; ties are broken by ascending id. */
static int compare_records(const void *lhs, const void *rhs) {
    const struct record *a = lhs;
    const struct record *b = rhs;

    if (a->priority != b->priority) {
        return b->priority - a->priority;
    }
    return (a->id > b->id) - (a->id < b->id);
}

static int compare_names(const void *lhs, const void *rhs) {
    const struct record *a = lhs;
    const struct record *b = rhs;

    return strncmp(a->name, b->name, RECORD_NAME_LEN);
}

void sort_records(struct record *records, size_t count, CompareFn cmp) {
    qsort(records, count, sizeof *records, cmp != NULL ? cmp : compare_records);
}

void sort_records_by_name(struct record *records, size_t count) {
    sort_records(records, count, compare_names);
}

const struct record *find_record(const struct record *records, size_t count, unsigned int id) {
    for (size_t i = 0; i < count; i++) {
        if (records[i].id == id) {
            return &records[i];
        }
    }
    return NULL;
}
