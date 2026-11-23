# Archived compiler documentation (read-only)

The active tree intentionally contains only the currently maintained Compiler
v2.1.0 entry points and the latest architecture acceptance evidence. Older
v0/v0.1/v1alpha4 specifications, implementation work packages, per-run screenshot
copies and former release-state diaries were removed from the active checkout,
**not erased from Git history**.

- [Browse the exact full archive at commit `72b41548a9c4`](https://github.com/odango-chan/ca-skill-compiler/tree/72b41548a9c4d29c6c49abed117a04a963454f22/docs)
- [Historical work packages](https://github.com/odango-chan/ca-skill-compiler/tree/72b41548a9c4d29c6c49abed117a04a963454f22/docs/workstreams)
- [Historical design contracts](https://github.com/odango-chan/ca-skill-compiler/tree/72b41548a9c4d29c6c49abed117a04a963454f22/docs/design)
- [Historical Archify browser evidence](https://github.com/odango-chan/ca-skill-compiler/tree/72b41548a9c4d29c6c49abed117a04a963454f22/docs/architecture/archify/evidence/history)

Current machine-readable contracts remain in `/contracts/` and must not be
replaced by archival prose. Historical frozen semantics implemented by the one
v1alpha5 Compiler Core are retained; only historical **documentation** was
pruned, never source code, fixtures or tests.

If a past decision must be amended, add a small explicit current ADR/release
note; do not copy the full old work-package log into `main` again.
