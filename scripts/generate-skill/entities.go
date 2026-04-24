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
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// entityKind is the Terraform binding kind.
type entityKind int

const (
	kindResource entityKind = 1 << iota
	kindDataSource
)

func (k entityKind) has(other entityKind) bool { return k&other != 0 }

// entity is a Terraform resource and/or data source as described by an
// OpenSpec capability and (optionally) the matching tfplugindocs page.
type entity struct {
	// Name is the fully-qualified provider type, e.g. "elasticstack_elasticsearch_index".
	Name string

	// ShortName is Name with the "elasticstack_" prefix stripped.
	// Used for docs filename matching and display.
	ShortName string

	// Kinds indicates whether this entity is exposed as a resource, data source, or both.
	Kinds entityKind

	// SpecCapability is the OpenSpec capability directory name (e.g. "elasticsearch-index").
	SpecCapability string

	// SpecPath is the absolute path to spec.md.
	SpecPath string

	// SpecBody is the full spec.md contents.
	SpecBody string

	// Purpose is the ## Purpose section body (trimmed).
	Purpose string

	// Schema is the ## Schema section body, verbatim (may contain multiple HCL blocks).
	Schema string

	// Implementation notes from the spec header (Resource implementation / Data source implementation lines).
	ResourceImpl   string
	DataSourceImpl string

	// DocsResourcePath, DocsDataSourcePath point to tfplugindocs pages if present.
	DocsResourcePath   string
	DocsDataSourcePath string

	// DocsSummary is a one-line description extracted from docs frontmatter or the first
	// non-heading paragraph of the spec Purpose.
	DocsSummary string

	// DocsExampleResource, DocsExampleDataSource are the "Example Usage" HCL blocks from docs.
	DocsExampleResource   string
	DocsExampleDataSource string

	// Subcategory is the docs frontmatter "subcategory" value, e.g. "Security", "Fleet", "Ingest".
	Subcategory string
}

// loadEntities walks specsDir and docsDir and returns the merged entity list
// sorted by Name.
func loadEntities(specsDir, docsDir string) ([]*entity, error) {
	specDirs, err := os.ReadDir(specsDir)
	if err != nil {
		return nil, fmt.Errorf("read specs dir %q: %w", specsDir, err)
	}

	byName := map[string]*entity{}
	for _, de := range specDirs {
		if !de.IsDir() {
			continue
		}
		capability := de.Name()
		// Skip CI / acceptance-test housekeeping specs: these describe workflows,
		// not user-facing Terraform entities.
		if strings.HasPrefix(capability, "ci-") || strings.HasPrefix(capability, "acceptance-") {
			continue
		}
		specPath := filepath.Join(specsDir, capability, "spec.md")
		body, err := os.ReadFile(specPath)
		if err != nil {
			// Some capability directories may exist without spec.md yet; skip quietly.
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %q: %w", specPath, err)
		}
		ent := parseSpec(capability, specPath, string(body))
		if ent == nil {
			// Non-entity spec (e.g. provider-level guidance). Skip for now.
			continue
		}
		byName[ent.Name] = ent
	}

	// Merge docs/resources and docs/data-sources.
	if err := mergeDocs(byName, filepath.Join(docsDir, "resources"), kindResource); err != nil {
		return nil, err
	}
	if err := mergeDocs(byName, filepath.Join(docsDir, "data-sources"), kindDataSource); err != nil {
		return nil, err
	}

	out := make([]*entity, 0, len(byName))
	for _, e := range byName {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// parseSpec extracts the fields we care about from a spec.md. Returns nil if
// the spec does not describe a provider entity (no "elasticstack_*" H1 title).
func parseSpec(capability, path, body string) *entity {
	lines := strings.Split(body, "\n")
	if len(lines) == 0 {
		return nil
	}
	name, ok := extractEntityNameFromH1(lines[0])
	if !ok {
		return nil
	}

	ent := &entity{
		Name:           name,
		ShortName:      strings.TrimPrefix(name, "elasticstack_"),
		SpecCapability: capability,
		SpecPath:       path,
		SpecBody:       body,
	}

	for _, line := range lines[:min(len(lines), 10)] {
		if rest, ok := strings.CutPrefix(line, "Resource implementation:"); ok {
			ent.ResourceImpl = cleanImpl(rest)
			ent.Kinds |= kindResource
		}
		if rest, ok := strings.CutPrefix(line, "Data source implementation:"); ok {
			ent.DataSourceImpl = cleanImpl(rest)
			ent.Kinds |= kindDataSource
		}
	}

	ent.Purpose = extractSection(body, "## Purpose")
	ent.Schema = extractSection(body, "## Schema")

	// Fallback: if the spec did not declare implementation lines (older specs),
	// infer kind from docs presence during merge. For now assume resource.
	if ent.Kinds == 0 {
		ent.Kinds = kindResource
	}

	return ent
}

// extractEntityNameFromH1 parses lines like:
//
//	# `elasticstack_elasticsearch_index` — Schema and Functional Requirements
//
// Returns (name, true) when matched, otherwise ("", false).
func extractEntityNameFromH1(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "# ") {
		return "", false
	}
	rest := strings.TrimPrefix(line, "# ")
	// Expect backtick-quoted name.
	if !strings.HasPrefix(rest, "`") {
		return "", false
	}
	end := strings.Index(rest[1:], "`")
	if end < 0 {
		return "", false
	}
	name := rest[1 : 1+end]
	if !strings.HasPrefix(name, "elasticstack_") {
		return "", false
	}
	return name, true
}

func cleanImpl(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "`")
	return strings.TrimSpace(s)
}

// extractSection returns the body of a markdown section starting with the given
// heading (e.g. "## Purpose"), stopped by the next heading of equal or lesser
// depth. The heading itself is not included.
func extractSection(body, heading string) string {
	lines := strings.Split(body, "\n")
	depth := headingDepth(heading)
	if depth == 0 {
		return ""
	}
	start := -1
	for i, l := range lines {
		if strings.TrimRight(l, " \t") == heading {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return ""
	}
	end := len(lines)
	for i := start; i < len(lines); i++ {
		d := headingDepth(lines[i])
		if d > 0 && d <= depth {
			end = i
			break
		}
	}
	return strings.TrimSpace(strings.Join(lines[start:end], "\n"))
}

// headingDepth returns the ATX heading depth (1..6) or 0 if the line is not a
// markdown heading. Indented headings and setext headings are ignored.
func headingDepth(line string) int {
	// Don't count headings inside fenced code blocks; callers should strip code
	// fences first if that matters. For our purposes spec sections do not contain
	// fenced "##" lines so this is adequate.
	if !strings.HasPrefix(line, "#") {
		return 0
	}
	n := 0
	for n < len(line) && line[n] == '#' {
		n++
	}
	if n == 0 || n > 6 {
		return 0
	}
	if n < len(line) && line[n] != ' ' {
		return 0
	}
	return n
}

// mergeDocs scans a docs directory (either "resources" or "data-sources") and
// annotates the matching entity with docs metadata and the Example Usage block.
// Docs filenames strip the "elasticstack_" prefix, so we match by ShortName.
func mergeDocs(byName map[string]*entity, dir string, kind entityKind) error {
	des, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read docs dir %q: %w", dir, err)
	}
	// Build a shortName -> *entity lookup.
	byShort := map[string]*entity{}
	for _, e := range byName {
		byShort[e.ShortName] = e
	}

	for _, de := range des {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".md") {
			continue
		}
		shortName := strings.TrimSuffix(de.Name(), ".md")
		ent, ok := byShort[shortName]
		if !ok {
			// A docs page without a matching spec is a gap we surface as a warning
			// during emit, but we don't invent an entity here.
			continue
		}
		path := filepath.Join(dir, de.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %q: %w", path, err)
		}
		meta, example := parseDocsPage(string(raw))
		ent.Kinds |= kind
		if ent.Subcategory == "" {
			ent.Subcategory = meta.Subcategory
		}
		if ent.DocsSummary == "" {
			ent.DocsSummary = meta.Description
		}
		switch kind {
		case kindResource:
			ent.DocsResourcePath = path
			ent.DocsExampleResource = example
		case kindDataSource:
			ent.DocsDataSourcePath = path
			ent.DocsExampleDataSource = example
		}
	}
	return nil
}

type docsMeta struct {
	Subcategory string
	Description string
}

// parseDocsPage extracts frontmatter fields and the first fenced terraform block
// that follows an "## Example Usage" heading.
func parseDocsPage(body string) (docsMeta, string) {
	var meta docsMeta

	// Frontmatter block between leading --- lines.
	if strings.HasPrefix(body, "---\n") {
		if end := strings.Index(body[4:], "\n---\n"); end >= 0 {
			fm := body[4 : 4+end]
			meta = parseFrontmatter(fm)
		}
	}

	example := ""
	if idx := strings.Index(body, "## Example Usage"); idx >= 0 {
		// Take the first ```terraform ... ``` block after the heading.
		rest := body[idx:]
		start := strings.Index(rest, "```terraform\n")
		if start >= 0 {
			blockStart := start + len("```terraform\n")
			endOffset := strings.Index(rest[blockStart:], "\n```")
			if endOffset >= 0 {
				example = strings.TrimRight(rest[blockStart:blockStart+endOffset], "\n")
			}
		}
	}
	return meta, example
}

// parseFrontmatter handles the tiny subset of YAML that tfplugindocs emits.
// Specifically single-line key: value pairs and multi-line description using
// the literal block (|-) indicator.
func parseFrontmatter(fm string) docsMeta {
	var meta docsMeta
	lines := strings.Split(fm, "\n")
	i := 0
	for i < len(lines) {
		line := lines[i]
		if after, ok := strings.CutPrefix(line, "subcategory:"); ok {
			v := strings.TrimSpace(after)
			meta.Subcategory = strings.Trim(v, `"'`)
		}
		if after, ok := strings.CutPrefix(line, "description:"); ok {
			v := strings.TrimSpace(after)
			if v == "|-" || v == "|" {
				// Collect indented continuation lines.
				var parts []string
				i++
				for i < len(lines) && (strings.HasPrefix(lines[i], "  ") || lines[i] == "") {
					parts = append(parts, strings.TrimSpace(lines[i]))
					i++
				}
				meta.Description = strings.TrimSpace(strings.Join(parts, " "))
				continue
			}
			meta.Description = strings.Trim(v, `"'`)
		}
		i++
	}
	return meta
}
