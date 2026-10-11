void greet(String name) {
  print('Hello, $name');
}

class Animal {
  final String name;
  Animal(this.name);

  void speak() {
    print('...');
  }
}

mixin Swimming {
  void swim() {
    print('swimming');
  }
}

enum Color { red, green, blue }

extension StringHelper on String {
  bool get isBlank => trim().isEmpty;
}

typedef IntCallback = void Function(int);

abstract class Repository {
  Future<void> save(Object entity);
}
