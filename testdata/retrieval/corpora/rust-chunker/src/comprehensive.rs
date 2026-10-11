fn bare_fn() {}

pub struct MyStruct {
    x: i32,
}

pub enum MyEnum {
    A,
    B,
}

pub trait MyTrait {
    fn required(&self);
}

pub const MY_CONST: i32 = 42;

impl MyStruct {
    fn method(&self) {}
}

impl<T> MyStruct {
    fn generic_method(&self) {}
}

type Alias = i32;

static MY_STATIC: i32 = 10;

mod my_module {}

union MyUnion {
    x: i32,
    y: f64,
}
