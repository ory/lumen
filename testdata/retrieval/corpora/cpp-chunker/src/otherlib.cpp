#include <string>
#include <vector>

namespace OtherLib {
    int helper_calls = 0;

    void helper() {
        ++helper_calls;
    }

    template <typename T>
    T min_val(T a, T b) {
        return a < b ? a : b;
    }

    template <typename T>
    class Registry {
    public:
        std::vector<T> items;
    };
}
