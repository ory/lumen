// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package chunker_test

import "testing"

func TestCSharpOperatorsIndexersAndFinalizers(t *testing.T) {
	src := `namespace Geometry
{
    public class Calculator
    {
        /// <summary>Adds two calculators.</summary>
        public static Calculator operator +(Calculator a, Calculator b)
        {
            return a;
        }

        public static implicit operator int(Calculator c) => c.Value;

        public int this[int index] => index;

        ~Calculator()
        {
            Dispose();
        }
    }
}
`
	chunks := chunkSource(t, "Calculator.cs", src)
	assertDecls(t, chunks, []wantDecl{
		{symbol: "operator +", kind: "method", start: 5, end: 9, contains: []string{"Adds two calculators."}},
		{symbol: "operator int", kind: "method", start: 11, end: 11},
		{symbol: "this[]", kind: "method", start: 13, end: 13},
		{symbol: "~Calculator", kind: "function", start: 15, end: 18},
	})
}
