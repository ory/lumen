#pragma once

struct Vec2 {
    float x = 0.0f;
    float y = 0.0f;

    float length() const;
    // Scales the vector to unit length in place; a zero vector is left unchanged.
    void normalize();
    Vec2 normalized() const;

    bool operator==(const Vec2& other) const;
    bool operator!=(const Vec2& other) const { return !(*this == other); }
};

float dot(const Vec2& a, const Vec2& b);
