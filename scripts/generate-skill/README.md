# generate-skill

Generates an agent Skill directory (`dist/skill/elasticstack-terraform/`) that teaches coding agents how to write Terraform configuration against the `elastic/elasticstack` provider.

## Why

Manually maintaining a skill for 100+ resources and data sources is not tractable and would drift instantly. This generator treats `openspec/specs/` as the authoritative schema source, merges it with the `docs/` output from tfplugindocs, and layers in hand-seeded static content from `assets/`.

## Design principles

- **Progressive disclosure.** The top-level `SKILL.md` is a small router (< 150 lines). The per-entity detail, the entity index, the provider block reference, and the gotchas live in separate files that the agent loads only when a task calls for them.
- **Deterministic.** No timestamps or network calls in content. Rerunning the generator on the same inputs produces byte-identical output.
- **No third-party dependencies.** Standard library only.

## Output layout

```
dist/skill/elasticstack-terraform/
├── SKILL.md                        # Copied verbatim from assets/SKILL.md
├── GENERATED.md                    # Provenance (generated)
└── references/
    ├── index.md                    # assets/references/index.md with {{ENTITIES}} substituted
    ├── context-checklist.md        # Copied verbatim from assets/
    ├── provider.md                 # Copied verbatim from assets/
    ├── gotchas.md                  # Copied verbatim from assets/
    ├── resources/<short_name>.md   # Generated (one per resource)
    └── data-sources/<short_name>.md # Generated (one per data source)
```

## What you can hand-edit

Everything under `assets/` is hand-editable Markdown. The generator copies it into the output tree unchanged, except for `references/index.md` where `{{ENTITIES}}` is replaced with the Markdown entity table. Edit `assets/SKILL.md`, `assets/references/provider.md`, etc. freely — no Go changes required.

The per-entity files under `references/resources/` and `references/data-sources/` are fully generated from `openspec/specs/` + `docs/`. If you need to change their shape, edit `scripts/generate-skill/emit.go` (`writeEntityRef`).

## Running

```sh
make skill-generate      # writes to dist/skill/elasticstack-terraform/
make skill-test          # runs the parser unit tests
```

Custom invocation:

```sh
go run ./scripts/generate-skill \
  -specs openspec/specs \
  -docs  docs \
  -assets scripts/generate-skill/assets \
  -out   dist/skill/elasticstack-terraform \
  -provider-version 0.14.4 \
  -v
```

## Inputs

- `openspec/specs/<capability>/spec.md` — authoritative schema and requirements. Parser extracts:
  - H1 entity name (`elasticstack_*`)
  - `Resource implementation:` / `Data source implementation:` lines
  - `## Purpose` and `## Schema` sections
  - Requirement sentences matching gotcha needles (force-new, deletion, version gates, JSON handling, import, connection)
- `docs/resources/*.md` and `docs/data-sources/*.md` — tfplugindocs output. Parser extracts:
  - Frontmatter `subcategory` and `description`
  - First `terraform` fenced block after `## Example Usage`
- `scripts/generate-skill/assets/` — hand-edited Markdown, copied verbatim. `assets/references/index.md` must contain the `{{ENTITIES}}` placeholder.

## Editing the skill

- To change wording, add guidance, or restructure non-per-entity pages: edit the corresponding file under `assets/` and rerun `make skill-generate`.
- To add a new hand-authored page: drop a file into `assets/<rel_path>` and link to it from `assets/SKILL.md`. It will be copied to `dist/skill/elasticstack-terraform/<rel_path>`.
- To change how per-entity files look: edit `emit.go` (`writeEntityRef`).
- To change how the entity index tables are rendered: edit `emit.go` (`renderEntityTables`).

## Tests

`entities_test.go` pins the spec/docs parsers so changes to the upstream format fail loudly.
