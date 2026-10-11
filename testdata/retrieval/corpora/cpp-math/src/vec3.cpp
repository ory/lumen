#include "vec3.hpp"

#include <cmath>

float Vec3::length() const {
    return std::sqrt(x * x + y * y + z * z);
}

void Vec3::normalize() {
    const float len = length();
    if (len == 0.0f) {
        return;
    }
    const float inv = 1.0f / len;
    x *= inv;
    y *= inv;
    z *= inv;
}

Vec3 Vec3::cross(const Vec3& other) const {
    return Vec3{y * other.z - z * other.y, z * other.x - x * other.z, x * other.y - y * other.x};
}
