# Examples

The current v1alpha4 sample used by the CLI and browser parity tests is [`web/parity-success`](../web/parity-success). Run it from the repository root:

```bash
skillc check web/parity-success
skillc inspect web/parity-success
skillc build web/parity-success --out ./dist/skillc-output
```

The directories under `examples/` are reference material from earlier development and migration work. They are not current v1alpha4 positive Build samples.

| Directory | Contents |
|---|---|
| `ca-agent-demo-project/` | Pinned source snapshot from ca-agent-demo, retained for migration analysis |
| `ca-skills-v0.2/` | Historical CA-SL v0.2 Skill source snapshot |
| `minimal-project/` | Synthetic fixture for the pre-v1alpha4 compiler |
| `script-contract-project/` | Synthetic pre-v1alpha4 Script Contract fixture |

The historical snapshots are kept for reference. They should not be modified to make them pass the v1alpha4 compiler. Use `web/parity-success` for a current compiler smoke test.

For compiler conformance coverage, see [tests/conformance/v1alpha4](../tests/conformance/v1alpha4) and the [acceptance record](../docs/COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md).
