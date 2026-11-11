# Skill Compiler

A small, **stateless** compiler/linker for multi-Skill projects.

## Project layout

```text
project/
  skills/       # editable SKILL sources, multiple Skills
  lib/          # imported read-only Tool contracts
  output/       # generated file tree, not a source input
```

Target commands: `skillc check <project>` and `skillc build <project> --out <directory>`. The same core should be callable from browser WebAssembly for Skill Studio on GitHub Pages.

No database, HTTP backend, Qualification, Release, Runtime execution, or automatic Tool execution.

**Status:** bootstrap in progress; source code and tests are being added separately. See [the design](docs/design/skill-compiler-v0.md) when available.

Historical design and rationale: [ca-skill-toolchain #133](https://github.com/odango-chan/ca-skill-toolchain/issues/133).
