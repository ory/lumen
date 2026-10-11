#!/usr/bin/env python3
"""Regenerates nomic-embed-code-q4km-reference.json.gz.

Usage: gen-nomic-reference.py <llama-embedding binary> <nomic-embed-code.Q4_K_M.gguf>

The binary must be upstream llama.cpp's llama-embedding built from the same
revision as internal/llama/update-llama.sh. Embeddings are computed on the CPU
(--device none) with last-token pooling and L2 normalization; queries carry
the model's query instruction, documents do not.
"""
import gzip
import hashlib
import json
import subprocess
import sys

INSTRUCTION = "Represent this query for searching relevant code: "
DOCUMENTS = [
    "func Factorial(n int) int {\n\tif n <= 1 {\n\t\treturn 1\n\t}\n\treturn n * Factorial(n-1)\n}",
    "def read_json(path):\n    with open(path) as f:\n        return json.load(f)",
    "async function fetchUser(id) {\n  const res = await fetch(`/api/users/${id}`);\n  return res.json();\n}",
    "func (s *Server) Shutdown(ctx context.Context) error {\n\treturn s.http.Shutdown(ctx)\n}",
]
QUERIES = [
    ("compute the factorial of a number recursively", 0),
    ("load and parse a JSON file from disk", 1),
    ("fetch a user from the REST API", 2),
    ("gracefully stop the HTTP server", 3),
]
SEP = "<#lumen-sep#>"


def main():
    binary, model = sys.argv[1], sys.argv[2]
    texts = DOCUMENTS + [INSTRUCTION + q for q, _ in QUERIES]
    args = [binary, "-m", model, "--device", "none", "--pooling", "last",
            "--embd-normalize", "2", "--embd-output-format", "json",
            "--embd-separator", SEP, "-p", SEP.join(texts)]
    out = subprocess.run(args, check=True, capture_output=True, text=True).stdout
    vecs = [d["embedding"] for d in json.loads(out)["data"]]
    sha = hashlib.sha256()
    with open(model, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 24), b""):
            sha.update(chunk)
    ref = {
        "model_sha256": sha.hexdigest(),
        "command": "llama-embedding (llama.cpp v0.6.0 d812350) -m nomic-embed-code.Q4_K_M.gguf --device none "
                   "--pooling last --embd-normalize 2 --embd-output-format json",
        "documents": [{"text": t, "embedding": [round(x, 6) for x in v]}
                      for t, v in zip(DOCUMENTS, vecs)],
        "queries": [{"query": q, "relevant": r, "embedding": [round(x, 6) for x in v]}
                    for (q, r), v in zip(QUERIES, vecs[len(DOCUMENTS):])],
    }
    with gzip.GzipFile("nomic-embed-code-q4km-reference.json.gz", "wb", mtime=0) as f:
        f.write(json.dumps(ref, separators=(",", ":")).encode())


if __name__ == "__main__":
    main()
