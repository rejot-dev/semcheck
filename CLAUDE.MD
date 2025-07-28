# semcheck

Semcheck (semantic checker) is a lightweight CLI tool that verifies whether code changes remain consistent with their associated specifications.
It detects when documentation, comments, or spec files are modified—or should have been—and ensures that implementation and
specification stay in sync. Ideal for pre-commit hooks or CI pipelines, Semcheck helps prevent spec drift with minimal setup.

## Implementation Cycle

When prompted for making changes to this code base, start by checking if the specification needs to be updated, if so, apply those changes first.


## Building project

Build the two binaries in this project through invocations of `just build` for semcheck and `just build-eval` for eval.

## Site Development

The `/site` directory contains an Astro-based website for the project:

- **Framework**: Astro with Tailwind CSS
- **Development server**: `npm run dev` (from `/site` directory)
- **Build**: `npm run build` (from `/site` directory)
- **Format code**: `npm run format` (from `/site` directory)

The site includes:
- Landing page (`src/pages/index.astro`)
- Reusable components in `src/components/`
- Code samples in `src/code-samples/`
- Static assets in `public/`

## Specification Files

Specification files are located in the `/specs` directory.

## Evaluations (evals)

The `/evals` directory contains the evaluation framework:

- `eval-config.yaml`: Configuration for running evaluations
- `expectations.csv`: Expected results for test cases
- `cases/`: Individual test cases, each containing:
  - `impl.go`: Implementation file to be checked
  - `spec.md`: Specification file to check against

### Creating New Eval Cases with Intentional Issues

Creating new evaluation test cases always start from a "correct" implementation and issues that violate the specification are introduces systematically, one by one.

1. **Identify Target Issues**: Research the specification and current implementation as defined by the case rule in eval-config.yaml to get context on potential spec violations that can be introduced in the implementation files.

2. **Introduce Code Issues**: Modify the implementation files to introduce the specific violations:
   - NEVER include comments documenting spec violations in the implementation

3. **Update Configuration Files**:
   - DO add a description of the violation introduced in the `eval-config.yaml` rule description key.
   - Update `expectations.csv` to reflect the expected number of errors, warnings, and notices
