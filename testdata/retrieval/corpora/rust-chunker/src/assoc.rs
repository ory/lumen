trait Producer {
    type Output;
    fn required(&self) -> Self::Output;
}

struct Foo;

impl Producer for Foo {
    type Output = i32;
    fn required(&self) -> i32 { 42 }
}
