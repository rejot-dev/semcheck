# semcheck

Semcheck (semantic checker) is a lightweight CLI tool that verifies whether code changes remain consistent with their associated specifications.
It detects when documentation, comments, or spec files are modified—or should have been—and ensures that implementation and
specification stay in sync. Ideal for pre-commit hooks or CI pipelines, Semcheck helps prevent spec drift with minimal setup.

## Implementation plan

And iterative plan for the implementation is in PLAN.md, only execute a single phase at once.

## Building project

Build the two binaries in this project through invocations of `just build` for semcheck and `just build-eval` for eval.
