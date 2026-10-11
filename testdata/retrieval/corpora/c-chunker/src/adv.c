#include <stdio.h>

#define MAX_SIZE 1024
#define SQUARE(x) ((x) * (x))

typedef int MyInt;
typedef struct { int x; int y; } Point2D;

struct Point {
    int x;
    int y;
};

enum Color { RED, GREEN, BLUE };

union Data {
    int i;
    float f;
};

int add(int a, int b) {
    return a + b;
}

int *get_ptr(void) {
    return 0;
}

static void internal_fn(void) {}

// Forward declaration (should be extractable as declaration)
void forward_declared(int x);

// Global variable declaration
int global_var = 42;

/* int commented_fn(void) {} */
