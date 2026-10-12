# AGENTS.md

## Comments

- Do not add comments to methods or functions that are not necessary. Well-named code explains itself.
- Comment only what the code cannot say: a non-obvious reason, a constraint, a trade-off, or a known pending item (`TODO`).
- Do not restate what the code does, and do not add a comment on every exported symbol by default.

## Simplicity

- Do not over-engineer. Prefer the simplest code that solves the current requirement and reads easily.
- Do not add abstractions, layers, options or indirection for needs that do not exist yet.
- Prefer one consistent way to express the same idea. Inject dependencies as small, named interfaces rather than a mix of interfaces and function fields.
- When two options are equally valid, choose the one a new reader understands fastest.
- Keep logic inside the function that uses it. Do not add package-level variables, helpers or constants used by a single function; declare them inside it.
- Do not wrap one line in a new function or split a short function into several. Extract only when the logic is reused or the function stops reading top to bottom.
- Do not optimize without evidence. Write the simplest version first and optimize only when a benchmark or profile shows a real bottleneck.
