#pragma once

struct Vec3 {
    float x = 0.0f;
    float y = 0.0f;
    float z = 0.0f;

    float length() const;
    void normalize();
    Vec3 cross(const Vec3& other) const;
};
