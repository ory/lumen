#include "vec2.hpp"
#include "vec3.hpp"

struct Camera {
    Vec3 forward;
    Vec2 pan;
};

void aim_camera(Camera& camera, const Vec3& target) {
    camera.forward = target;
    camera.forward.normalize();
    camera.pan.normalize();
}
