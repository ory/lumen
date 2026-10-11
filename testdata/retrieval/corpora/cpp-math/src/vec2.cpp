#include "vec2.hpp"

#include <cmath>

static constexpr float kEpsilon = 1e-6f;

float Vec2::length() const {
    return std::sqrt(x * x + y * y);
}

void Vec2::normalize() {
    const float len = length();
    if (len < kEpsilon) {
        return;
    }
    x /= len;
    y /= len;
}

Vec2 Vec2::normalized() const {
    Vec2 copy = *this;
    copy.normalize();
    return copy;
}

bool Vec2::operator==(const Vec2& other) const {
    return std::fabs(x - other.x) < kEpsilon && std::fabs(y - other.y) < kEpsilon;
}

float dot(const Vec2& a, const Vec2& b) {
    return a.x * b.x + a.y * b.y;
}
