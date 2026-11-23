> **HISTORICAL / NOT CURRENT INPUT (v2.1.0).** Retained at its original path as frozen older-version or design evidence, not an active BuildRequest, syntax or CLI target. The current Compiler accepts **only CA-SL v1alpha5**; v1alpha4 inherited *semantics* remain implemented, but v1alpha4 API inputs are rejected. See [documentation](../README.md), [Development contracts](../../contracts/development/v1alpha1/README.md) and [v2.1.0 Release](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0). Statements below about old implementation states are historical. <!-- historical-docs-v2.1.0 -->

---

> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](../COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# Dependency Snapshot Digest v1

**Status:** FROZEN Compiler/Studio interoperability algorithm for BuildRequest v1alpha4. Any framing change requires a new digest domain separator and coordinated API revision.

This document proposes the snapshot integrity algorithm owned by the BuildRequest v1alpha4 transport contract. It makes no compatibility claim with Compiler v0.1: that earlier design explicitly left digest protocol out of scope. The algorithm identity is the domain separator below, not the CA-SL language version.

## 1. Digest value

The wire value is sha256: followed by 64 lowercase hexadecimal characters. The digest is SHA-256 over a canonical binary preimage. It is independent of Dependency Alias / dependencyId, file-array order, JSON property order, base64 spelling, Git URL, branch, commit, timestamp or source provider.

Raw file bytes and normalized snapshot-relative UTF-8 paths are authoritative.

## 2. Input validation

Before hashing, require:
- 1–512 files in a nonempty snapshot;
- each path is valid UTF-8, normalized POSIX relative path, 1–240 UTF-8 bytes;
- no absolute path, empty segment, . / .. segment, backslash, colon or NUL;
- no duplicate path;
- no duplicate dependencyId in one BuildRequest;
- no more than 512 total source + dependency + trusted bundle files in one request;
- no more than 2 MiB raw bytes per file and 16 MiB total raw bytes per request.

Decode base64 to raw bytes before hashing. Native callers hash the same raw bytes. Reject invalid base64 and invalid paths before digest calculation.

## 3. Canonical binary preimage

All integers are unsigned big-endian.

1. ASCII domain separator: CA-SKILL-DEPENDENCY-SNAPSHOT\0v1\0.
2. File count: uint32.
3. Sort files by unsigned lexicographic order of their UTF-8 path bytes.
4. For each sorted file append path byte length (uint32), path UTF-8 bytes, raw content byte length (uint64), and raw content bytes.

No separator, terminator, JSON serialization, newline normalization, YAML canonicalization or text decoding is inserted between records. Empty file content is valid; empty paths are not.

The dependencyId is excluded. Two aliases with byte-identical normalized file sets have the same digest.

The same file-set framing is used for CapabilityContracts and EvidenceTrustBindings bundles; their separate deployment identity, publisher authorization and bundle kind are validated outside the file-set digest.

## 4. Golden vector

Files (each content shown as UTF-8; trailing newlines are significant):

| Snapshot-relative path | Exact content |
|---|---|
| schemas/a.json | {"type":"string"} followed by LF |
| tools/catalog.yaml | apiVersion: v1 followed by LF |

Canonical preimage length: 125 bytes.

~~~text
43412d534b494c4c2d444550454e44454e43592d534e415053484f5400763100000000020000000e736368656d61732f612e6a736f6e00000000000000127b2274797065223a22737472696e67227d0a00000012746f6f6c732f636174616c6f672e79616d6c000000000000000f61706956657273696f6e3a2076310a
~~~

Expected digest:

~~~text
sha256:f9a26e8cceb4db7d36eb7e3905c73b1b166ff03bcb96a77297571e715020bfbc
~~~

Machine-readable vectors: [dependency-snapshot-digest-vectors.json](../../contracts/v1alpha4/dependency-snapshot-digest-vectors.json).

Required tests: reversed file order yields the same digest; changing one content byte or path changes it; changing only dependencyId does not; duplicate paths, traversal paths, invalid UTF-8 and invalid base64 are rejected; Native Go and WASM match every vector byte-for-byte.

## 5. Ownership and versioning

This digest is only an integrity check for the exact immutable snapshot bytes supplied by the trusted caller. It is not a resolver, package manager, lock lifecycle, registry, cache or release feature. Studio / the trusted snapshot source computes and retains the digest with its immutable snapshot record; Compiler recomputes it before linking. Compiler does not persist it or return it in BuildResult.

Compiler owns this algorithm and golden vectors. Studio implements the exact same algorithm for immutable dependency snapshots and locks. No side may change path normalization, sorting, framing or content-byte handling under this algorithm identity. Any change requires a new digest domain separator and coordinated BuildRequest contract revision.
