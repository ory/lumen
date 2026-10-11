// Top-level functions
void topLevelFn() {}

Future<void> asyncFn() async {}

Stream<int> generatorFn() async* { yield 1; }

Iterable<int> syncGenerator() sync* { yield 1; }

// Classes
class BaseClass {
  final String name;
  BaseClass(this.name);
  BaseClass.named(String n) : name = n;

  void method() {}
  static void staticMethod() {}

  factory BaseClass.create() => BaseClass('default');

  String get label => name;
  set label(String v) {}
}

abstract class AbstractBase {
  void abstractMethod();
}

class Child extends BaseClass {
  Child() : super('child');
}

// Mixins
mixin Printable {
  void printSelf() {}
}

// Enums (enhanced, Dart 2.17+)
enum Status {
  active, inactive;

  String get display => name.toUpperCase();
}

// Extensions
extension NumHelper on num {
  bool get isPositive => this > 0;
}

// Typedefs
typedef JsonMap = Map<String, dynamic>;
typedef Callback<T> = void Function(T);

// Top-level variable
final version = '1.0.0';
