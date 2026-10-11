export const greet = (name) => {
  return "hello " + name;
};

const handler = function() {
  return 42;
};

var legacy = function() {
  return 0;
};

export const gen = function*() {
  yield 1;
};

export function namedFn() {}

export class Animal {
  speak() {}
}
