#include <iostream>

class Vec2 {
public:
    float x;
    float y;
    void normalize();
};

struct Point {
    int x;
    int y;
};

enum Color { RED, GREEN, BLUE };

int add(int a, int b) {
    return a + b;
}

namespace MyLib {
    void helper() {}
    int libVar = 42;
}

using IntAlias = int;

void Vec2::normalize() {
    // normalize impl
}

template <typename T>
T max_val(T a, T b) {
    return a > b ? a : b;
}

template <typename T>
class Container {
public:
    T value;
};

// C++20 concept
template <typename T>
concept Addable = requires(T a, T b) { a + b; };

union Data {
    int i;
    float f;
};

// function commented() {}
