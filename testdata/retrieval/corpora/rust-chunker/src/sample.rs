pub fn add(a: i32, b: i32) -> i32 {
    a + b
}

pub struct Point {
    pub x: f64,
    pub y: f64,
}

pub trait Drawable {
    fn draw(&self);
}

pub const MAX_SIZE: usize = 1024;
