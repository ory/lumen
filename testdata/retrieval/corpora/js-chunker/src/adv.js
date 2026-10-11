// Regular declarations
function regularFn() {}
function* generatorFn() { yield 1; }

// Variable-based functions
const arrowFn = () => {};
const funcExpr = function namedExpr() {};
var legacyFn = function() {};
let mutableFn = function() {};

// Export wrappers
export function exportedFn() {}
export const exportedArrow = () => {};
export class ExportedClass {}
export default function defaultExport() {}

// Classes
class Animal {
  speak() {}
  get name() { return ""; }
}

// Edge cases: NOT definitions
(function iife() {})();
const result = iife;
// function commented() {}

// Object methods (debatable - documenting behavior)
const handlers = {
  onClick() {},
  onHover: () => {},
};
