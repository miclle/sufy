# sufy.sandbox.toml Configuration Reference

The `sandbox template` commands can read `sufy.sandbox.toml` from the project root to
persist template parameters. Prefer the environment-agnostic `name` for locating
templates; existing configurations that use `template_id` continue to work.

## Full Field Reference

```toml
# Template identity: prefer name only, so different environments can auto-locate
# their own templates by name.
name = "my-template"

# Backward compatible: if template_id is set, it takes precedence.
# template_id = "tmpl-xxxxxxxxxxxx"

# Build inputs
dockerfile    = "./Dockerfile"
path          = "."
from_image    = ""
from_template = ""

# Runtime
start_cmd = ""
ready_cmd = ""

# Resources
cpu_count = 2
memory_mb = 2048

# Build options
no_cache = false
```

| Field | Type | Description |
|------|------|------|
| template_id | string | Template ID. Takes precedence when present and routes to rebuild. |
| name | string | Template name. When template_id is absent, build/get/publish/delete/unpublish locate the remote template by name. |
| dockerfile | string | Path to the Dockerfile. |
| path | string | Build context directory. |
| from_image | string | Base Docker image. |
| from_template | string | Base template. |
| start_cmd | string | Container startup command. |
| ready_cmd | string | Readiness check command. |
| cpu_count | int | CPU cores. |
| memory_mb | int | Memory in MiB. |
| no_cache | bool | Force a full rebuild, ignoring cache. |

## Precedence

`CLI flag > config file > built-in default`

When both are provided, the CLI value wins and an override notice is printed once to stderr.

## Build Input Combinations

`from_image` and `from_template` are mutually exclusive.

`dockerfile` can be used alone or combined with `from_image` or `from_template`:
- `dockerfile` only: the Dockerfile's `FROM` is used as the base image.
- `from_image + dockerfile`: `from_image` is the base image; the Dockerfile `FROM` is overridden.
- `from_template + dockerfile`: `from_template` is the base template; the Dockerfile `FROM` is parsed but not used as the real base image.

`from_template + dockerfile` is a good fit for layering small dependencies on top of a
shared base template — for example, multiple agent templates sharing one `agents-base`.

## Config File Lookup Rules

1. `--config <path>` (only supported by `build`)
2. `sufy.sandbox.toml` in the current working directory
3. If neither is found, run in pure CLI mode (backward compatible)

## Template Locator Rules

1. An explicitly passed `template_id`, or `template_id` from the config file, takes precedence.
2. When `template_id` is not provided, `name` is used to look up the remote template by alias.
3. `build` enters rebuild when name hits; otherwise it creates a new template.
4. `get` / `publish` / `delete` / `unpublish` without a template ID also read `template_id` from
   the local config first, then fall back to looking up by `name`.

## Auto-writeback Behavior

- After a first successful create (config file present, `template_id` empty), sufy writes
  the new `template_id` back to the file for backward compatibility with existing scripts.
- When a name lookup hits an existing template and triggers a rebuild, `template_id` is not
  written back — the config remains reusable across environments.
- Without `--wait`, writeback happens after the build successfully starts. With `--wait`,
  it happens after the build completes with status `ready`.
- The writer replaces any existing `template_id` assignment line (regardless of original value)
  or inserts a new line at the top of the file.
- Comments, field order, and whitespace are preserved.
- After writing, sufy prints to stdout: `Written template_id to <path> (please commit this file)`.

## Team Collaboration

Commit `sufy.sandbox.toml` to version control:
- Teammates can clone the repo and run `sufy sandbox template build` to locate or create a
  template in their current environment by `name`.
- CI scripts need only a single command — no scattered flag plumbing.
- Typically only `name` / `dockerfile` / resource fields need to be committed. Keeping
  `template_id` pins the config to a specific environment.

## Example: Minimal Project

```
my-template/
├── Dockerfile
└── sufy.sandbox.toml
```

`sufy.sandbox.toml`:
```toml
name = "my-template"
dockerfile = "./Dockerfile"
cpu_count = 2
memory_mb = 2048
```

Run:
```bash
sufy sandbox template build --wait
# First run: no template by that name → create. Subsequent runs: name hit → rebuild.
```

Rebuilding later:
```bash
sufy sandbox template build --wait  # name hit → rebuild, no 409
```
