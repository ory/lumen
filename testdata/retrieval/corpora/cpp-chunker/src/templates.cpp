template <typename T>
T max_val(T a, T b) {
    return a > b ? a : b;
}

template <typename T>
class Container {
public:
    T value;
};

union MyUnion {
    int i;
    float f;
};
