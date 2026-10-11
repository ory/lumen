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

func TestRubyMethodNamesKeepPunctuation(t *testing.T) {
	src := `class Metal
  def response_body=(body)
    @body = body
  end

  def valid?
    true
  end

  def save!
    persist
  end

  def self.config=(value)
    @config = value
  end

  def ==(other)
    other.id == id
  end
end
`
	chunks := chunkSource(t, "metal.rb", src)
	assertDecls(t, chunks, []wantDecl{
		{symbol: "Metal.response_body=", kind: "function", start: 2, end: 4, contains: []string{"@body = body"}},
		{symbol: "Metal.valid?", kind: "function", start: 6, end: 8},
		{symbol: "Metal.save!", kind: "function", start: 10, end: 12},
		{symbol: "Metal.config=", kind: "function", start: 14, end: 16},
		{symbol: "Metal.==", kind: "function", start: 18, end: 20},
	})
}
