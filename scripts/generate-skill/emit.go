// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type generator struct {
	entities        []*entity
	assetsDir       string
	outDir          string
	providerVersion string
	log             io.Writer
	verbose         bool
}

func (g *generator) emit() error {
	if err := os.RemoveAll(g.outDir); err != nil {
		return fmt.Errorf("clean out dir: %w", err)
	}
	dirs := []string{
		g.outDir,
		filepath.Join(g.outDir, "references"),
		filepath.Join(g.outDir, "references", "resources"),
		filepath.Join(g.outDir, "references", "data-sources"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}

	// Copy hand-editable templates first (SKILL.md, index.md, context-checklist.md,
	// provider.md, gotchas.md). writeIndex then replaces the {{ENTITIES}} marker
	// in index.md with the generated entity tables.
	if err := g.copyStaticAssets(); err != nil {
		return err
	}
	if err := g.writeIndex(); err != nil {
		return err
	}
	if err := g.writePerEntityRefs(); err != nil {
		return err
	}
	if err := g.writeProvenance(); err != nil {
		return err
	}
	return nil
}

// indexEntitiesPlaceholder is replaced with the generated entity tables in
// the hand-editable references/index.md template.
const indexEntitiesPlaceholder = "{{ENTITIES}}"

// writeIndex renders references/index.md by replacing {{ENTITIES}} in the
// copied template with a Markdown table of every entity grouped by subcategory.
// Only the table is generated; the surrounding prose is fully editable in
// assets/references/index.md.
func (g *generator) writeIndex() error {
	path := filepath.Join(g.outDir, "references", "index.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read index template (expected copy of assets/references/index.md at %q): %w", path, err)
	}
	if !strings.Contains(string(raw), indexEntitiesPlaceholder) {
		return fmt.Errorf("references/index.md template is missing %s placeholder", indexEntitiesPlaceholder)
	}
	rendered := strings.Replace(string(raw), indexEntitiesPlaceholder, g.renderEntityTables(), 1)
	return os.WriteFile(path, []byte(rendered), 0o644)
}

// renderEntityTables produces the grouped Markdown tables that slot into the
// index.md template's {{ENTITIES}} placeholder.
func (g *generator) renderEntityTables() string {
	groups := map[string][]*entity{}
	for _, e := range g.entities {
		cat := e.Subcategory
		if cat == "" {
			cat = inferSubcategory(e)
		}
		groups[cat] = append(groups[cat], e)
	}
	cats := make([]string, 0, len(groups))
	for c := range groups {
		cats = append(cats, c)
	}
	sort.Strings(cats)

	var b strings.Builder
	for _, cat := range cats {
		b.WriteString("## " + cat + "\n\n")
		b.WriteString("| Entity | Kind | Summary | Reference |\n")
		b.WriteString("|---|---|---|---|\n")
		ents := groups[cat]
		sort.Slice(ents, func(i, j int) bool { return ents[i].Name < ents[j].Name })
		for _, e := range ents {
			kind := kindLabel(e.Kinds)
			summary := oneLine(e.DocsSummary)
			if summary == "" {
				summary = oneLine(e.Purpose)
			}
			summary = truncate(summary, 120)
			link := entityRefPath(e)
			fmt.Fprintf(&b, "| `%s` | %s | %s | [%s](%s) |\n",
				e.Name, kind, escapePipe(summary), filepath.Base(link), link)
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// writePerEntityRefs writes one reference file per entity. When an entity is
// both resource and data source, we produce two files with cross-links.
func (g *generator) writePerEntityRefs() error {
	for _, e := range g.entities {
		if e.Kinds.has(kindResource) {
			if err := g.writeEntityRef(e, kindResource); err != nil {
				return err
			}
		}
		if e.Kinds.has(kindDataSource) {
			if err := g.writeEntityRef(e, kindDataSource); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *generator) writeEntityRef(e *entity, kind entityKind) error {
	var b strings.Builder
	label := "Resource"
	impl := e.ResourceImpl
	example := e.DocsExampleResource
	if kind == kindDataSource {
		label = "Data source"
		impl = e.DataSourceImpl
		example = e.DocsExampleDataSource
	}

	fmt.Fprintf(&b, "# `%s` (%s)\n\n", e.Name, strings.ToLower(label))

	summary := oneLine(e.DocsSummary)
	if summary == "" {
		summary = oneLine(e.Purpose)
	}
	if summary != "" {
		b.WriteString(summary + "\n\n")
	}

	b.WriteString("## Facts\n\n")
	if e.Subcategory != "" {
		b.WriteString("- Subcategory: " + e.Subcategory + "\n")
	}
	b.WriteString("- Kind: " + label + "\n")
	if impl != "" {
		b.WriteString("- Implementation: `" + impl + "`\n")
	}
	if kind == kindResource && e.Kinds.has(kindDataSource) {
		b.WriteString("- See also: [data source](../data-sources/" + e.ShortName + ".md)\n")
	}
	if kind == kindDataSource && e.Kinds.has(kindResource) {
		b.WriteString("- See also: [resource](../resources/" + e.ShortName + ".md)\n")
	}
	b.WriteString("- Spec: `openspec/specs/" + e.SpecCapability + "/spec.md`\n")
	b.WriteString("\n")

	if e.Schema != "" {
		b.WriteString("## Schema\n\n")
		b.WriteString(e.Schema)
		b.WriteString("\n\n")
	}

	if example != "" {
		b.WriteString("## Example\n\n")
		b.WriteString("```terraform\n")
		b.WriteString(example)
		b.WriteString("\n```\n\n")
	}

	// Surface lifecycle / gotcha signals mined from the spec requirements.
	if notes := mineSpecSignals(e.SpecBody); notes != "" {
		b.WriteString("## Lifecycle & gotchas (from spec)\n\n")
		b.WriteString(notes)
		b.WriteString("\n\n")
	}

	b.WriteString("## Further reading\n\n")
	b.WriteString("- Full requirements: `openspec/specs/" + e.SpecCapability + "/spec.md`\n")
	if kind == kindResource && e.DocsResourcePath != "" {
		b.WriteString("- Generated docs: `" + e.DocsResourcePath + "`\n")
	}
	if kind == kindDataSource && e.DocsDataSourcePath != "" {
		b.WriteString("- Generated docs: `" + e.DocsDataSourcePath + "`\n")
	}

	sub := "resources"
	if kind == kindDataSource {
		sub = "data-sources"
	}
	path := filepath.Join(g.outDir, "references", sub, e.ShortName+".md")
	return writeFile(path, b.String())
}

// copyStaticAssets copies hand-seeded content from the assets dir into the
// skill output, applying template substitutions (currently {{VERSION}}) on
// Markdown files. Non-Markdown files are copied byte-for-byte.
func (g *generator) copyStaticAssets() error {
	if _, err := os.Stat(g.assetsDir); os.IsNotExist(err) {
		fmt.Fprintf(g.log, "note: assets dir %q does not exist; skipping static content\n", g.assetsDir)
		return nil
	}
	return filepath.WalkDir(g.assetsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(g.assetsDir, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(g.outDir, rel)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".md") {
			data = []byte(g.applySubstitutions(string(data)))
		}
		return os.WriteFile(dest, data, 0o644)
	})
}

// applySubstitutions replaces template placeholders in hand-editable Markdown
// assets. {{VERSION}} is replaced with the provider version when one was
// supplied via -provider-version; otherwise a zero-dev sentinel is used so the
// output still satisfies semver consumers (e.g. sandbox lint).
func (g *generator) applySubstitutions(s string) string {
	version := g.providerVersion
	if version == "" {
		version = "0.0.0-dev"
	}
	return strings.ReplaceAll(s, "{{VERSION}}", version)
}

// writeProvenance records what the skill was generated from. Intentionally the
// only file with a mutable field (commit/version), kept separate so diffs on
// the main content remain stable when only metadata changes.
func (g *generator) writeProvenance() error {
	var b strings.Builder
	b.WriteString("# Generated skill — provenance\n\n")
	b.WriteString("This skill is auto-generated from the elastic/terraform-provider-elasticstack repository.\n\n")
	fmt.Fprintf(&b, "- Entities: %d (resources: %d, data sources: %d)\n",
		len(g.entities), countKind(g.entities, kindResource), countKind(g.entities, kindDataSource))
	if g.providerVersion != "" {
		b.WriteString("- Provider version: " + g.providerVersion + "\n")
	}
	b.WriteString("- Source of truth: `openspec/specs/` + `docs/`\n")
	b.WriteString("- Generator: `scripts/generate-skill`\n\n")
	b.WriteString("Do not edit files in this directory by hand. Re-run `make skill-generate` from the provider repo instead.\n")
	return writeFile(filepath.Join(g.outDir, "GENERATED.md"), b.String())
}

// --- helpers ---

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func kindLabel(k entityKind) string {
	switch {
	case k.has(kindResource) && k.has(kindDataSource):
		return "resource + data source"
	case k.has(kindResource):
		return "resource"
	case k.has(kindDataSource):
		return "data source"
	}
	return "unknown"
}

func countKind(es []*entity, k entityKind) int {
	n := 0
	for _, e := range es {
		if e.Kinds.has(k) {
			n++
		}
	}
	return n
}

func entityRefPath(e *entity) string {
	if e.Kinds.has(kindResource) {
		return "resources/" + e.ShortName + ".md"
	}
	return "data-sources/" + e.ShortName + ".md"
}

// inferSubcategory produces a best-effort bucket from the entity short name
// when the docs frontmatter didn't give us one (e.g. data sources sometimes
// omit it).
func inferSubcategory(e *entity) string {
	name := e.ShortName
	switch {
	case strings.HasPrefix(name, "elasticsearch_ingest_processor_"):
		return "Ingest processors"
	case strings.HasPrefix(name, "elasticsearch_security_"):
		return "Security"
	case strings.HasPrefix(name, "elasticsearch_ml_"):
		return "Machine Learning"
	case strings.HasPrefix(name, "elasticsearch_snapshot"):
		return "Snapshot"
	case strings.HasPrefix(name, "elasticsearch_watcher"):
		return "Watcher"
	case strings.HasPrefix(name, "elasticsearch_transform"):
		return "Transform"
	case strings.HasPrefix(name, "elasticsearch_"):
		return "Elasticsearch"
	case strings.HasPrefix(name, "kibana_security_"):
		return "Kibana Security"
	case strings.HasPrefix(name, "kibana_synthetics_"):
		return "Kibana Synthetics"
	case strings.HasPrefix(name, "kibana_agentbuilder_"):
		return "Kibana Agent Builder"
	case strings.HasPrefix(name, "kibana_"):
		return "Kibana"
	case strings.HasPrefix(name, "fleet_"):
		return "Fleet"
	case strings.HasPrefix(name, "apm_"):
		return "APM"
	}
	return "Other"
}

// oneLine returns the first sentence of s, trimmed, with trailing "See: URL"
// noise stripped so the index table stays scannable.
func oneLine(s string) string {
	var line string
	for l := range strings.SplitSeq(s, "\n") {
		l = strings.TrimSpace(l)
		if l != "" {
			line = l
			break
		}
	}
	if line == "" {
		return ""
	}
	// Cut at first ". " to keep just the first sentence.
	if i := strings.Index(line, ". "); i > 0 {
		line = line[:i+1]
	}
	// Strip trailing "See: https://..." or "see the X documentation https://..." noise.
	for _, marker := range []string{" See:", " see:", " See the", " see the"} {
		if i := strings.Index(line, marker); i > 0 {
			line = line[:i]
		}
	}
	return strings.TrimSpace(line)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func escapePipe(s string) string {
	return strings.ReplaceAll(s, "|", `\|`)
}

// mineSpecSignals pulls bullet-points out of the spec for common gotcha
// categories. We detect headings and SHALL/MUST sentences that mention
// force-new, deletion protection, version compatibility, JSON normalization,
// or import behavior.
func mineSpecSignals(body string) string {
	signals := []struct {
		label   string
		needles []string
	}{
		{"Forces replacement", []string{"force new", "requires replacement", "RequiresReplace"}},
		{"Deletion / protection", []string{"deletion_protection", "delete.*refuses", "refuses to delete"}},
		{"Version gates", []string{"requires Elasticsearch >=", "requires Kibana >=", "requires Fleet", "version >=", "Unsupported Feature"}},
		{"JSON handling", []string{"jsonencode", "JSON (normalized)", "normalized JSON", "preserve.*unknown"}},
		{"Import", []string{"import ", "ImportState"}},
		{"Connection", []string{"elasticsearch_connection", "kibana_connection", "fleet_connection"}},
	}

	lines := strings.Split(body, "\n")
	var out strings.Builder
	for _, sig := range signals {
		var hits []string
		seen := map[string]struct{}{}
		for _, l := range lines {
			low := strings.ToLower(l)
			match := false
			for _, n := range sig.needles {
				if strings.Contains(low, strings.ToLower(n)) {
					match = true
					break
				}
			}
			if !match {
				continue
			}
			trim := strings.TrimSpace(l)
			// Skip headings and fenced markers.
			if strings.HasPrefix(trim, "#") || strings.HasPrefix(trim, "```") {
				continue
			}
			if trim == "" {
				continue
			}
			// De-duplicate and cap.
			if _, ok := seen[trim]; ok {
				continue
			}
			seen[trim] = struct{}{}
			hits = append(hits, trim)
			if len(hits) >= 4 {
				break
			}
		}
		if len(hits) == 0 {
			continue
		}
		out.WriteString("**" + sig.label + "**\n")
		for _, h := range hits {
			// Already-bulleted lines keep their marker; otherwise bullet them.
			if strings.HasPrefix(h, "- ") || strings.HasPrefix(h, "* ") {
				out.WriteString(h + "\n")
			} else {
				out.WriteString("- " + truncate(h, 220) + "\n")
			}
		}
		out.WriteString("\n")
	}
	return strings.TrimRight(out.String(), "\n")
}
