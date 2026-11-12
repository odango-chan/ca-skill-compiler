# Dependency Snapshot Digest v0.1

Status: **PROPOSED NORMATIVE ALGORITHM FOR CROSS-LANGUAGE IMPLEMENTATION**
Scope: deterministic content integrity for one resolved dependency snapshot. This is not a source URL, Git commit, lockfile, or package identity.

## 1. Digest value

The wire value is exactly:

`sha256:` followed by 64 lowercase hexadecimal characters.

The digest is computed over the canonical binary preimage below. It is independent of JSON property order, base64 formatting, dependency ID/alias, source URL, source revision, timestamps, and input file-array order. Raw file bytes and normalized snapshot-relative paths are authoritative.

## 2. Input validation before hashing

For each dependency snapshot:

- Require 1..512 files.
- Require each path to be valid UTF-8, normalized POSIX relative path, 1..240 UTF-8 bytes.
- Reject absolute paths, empty segments, `.` / `..` segments, backslashes, colons, NUL, and duplicate paths.
- Reject duplicate dependency IDs within one BuildRequest.
- Apply the BuildRequest aggregate limits across Skill source and all dependency files: at most 512 files, 2 MiB per file, 16 MiB total raw bytes.
- Decode base64 transport to raw bytes before hashing. Native callers hash the same raw bytes.

The Compiler must recompute and compare the digest before parsing/linking a snapshot. A mismatch is a hard error. Studio must calculate the exact same value before persisting the lock.

## 3. Canonical binary preimage

All integers are unsigned big-endian.

1. ASCII domain separator: `CA-SKILL-DEPENDENCY-SNAPSHOT\0v1\0`.
2. File count: uint32.
3. Sort files by the unsigned lexicographic order of their UTF-8 path bytes.
4. For each sorted file, append:
   - path byte length: uint32;
   - path UTF-8 bytes;
   - raw content byte length: uint64;
   - raw content bytes.

No separator, terminator, JSON serialization, newline normalization, YAML canonicalization, or text decoding is added between records. Empty file content is valid; empty paths are not.

The dependency ID is deliberately excluded: it is a Workspace Configuration alias, not content. Two aliases pointing to byte-identical normalized file sets have the same digest.

## 4. Golden vector 1

Files (shown as UTF-8 text; the trailing newline in each file is significant):

| Snapshot-relative path | Exact content |
|---|---|
| `schemas/a.json` | `{"type":"string"}\n` |
| `tools/catalog.yaml` | `apiVersion: v1\n` |

The canonical preimage is 125 bytes and its hex representation is:

```text
43412d534b494c4c2d444550454e44454e43592d534e415053484f5400763100000000020000000e736368656d61732f612e6a736f6e00000000000000127b2274797065223a22737472696e67227d0a00000012746f6f6c732f636174616c6f672e79616d6c000000000000000f61706956657273696f6e3a2076310a
```

Expected digest:

```text
sha256:f9a26e8cceb4db7d36eb7e3905c73b1b166ff03bcb96a77297571e715020bfbc
```

Implementations must additionally test:
- reversed input array order gives the same digest;
- changing one byte changes the digest;
- changing a path changes the digest;
- changing only dependencyId does not change the digest;
- duplicate paths, traversal paths, invalid UTF-8 and invalid base64 are rejected before hashing.

## 5. Ownership and versioning

Compiler owns this canonical algorithm and golden vectors. Studio implements it for Workspace Configuration lock creation and validation. Neither side may silently change the preimage under v0.1. Any change to path normalization, file ordering, framing, or content bytes requires a new digest version and coordinated contract revision.
