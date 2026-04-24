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
	"strings"
	"testing"
)

func TestExtractEntityNameFromH1(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"# `elasticstack_elasticsearch_index` — Schema and Functional Requirements", "elasticstack_elasticsearch_index", true},
		{"# `elasticstack_kibana_alerting_rule` — X", "elasticstack_kibana_alerting_rule", true},
		{"# Not a provider entity", "", false},
		{"## `elasticstack_foo` — subheading", "", false},
		{"# `something_else` — nope", "", false},
	}
	for _, tc := range cases {
		got, ok := extractEntityNameFromH1(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Errorf("extractEntityNameFromH1(%q) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestExtractSection(t *testing.T) {
	body := strings.Join([]string{
		"# Title",
		"",
		"## Purpose",
		"",
		"This is the purpose.",
		"",
		"## Schema",
		"",
		"schema body",
		"",
		"### Subsection",
		"",
		"still schema",
		"",
		"## Requirements",
		"",
		"req body",
	}, "\n")

	if got := extractSection(body, "## Purpose"); got != "This is the purpose." {
		t.Errorf("Purpose = %q", got)
	}
	schema := extractSection(body, "## Schema")
	if !strings.Contains(schema, "schema body") || !strings.Contains(schema, "still schema") {
		t.Errorf("Schema missing expected content: %q", schema)
	}
	if strings.Contains(schema, "req body") {
		t.Errorf("Schema leaked into Requirements: %q", schema)
	}
}

func TestParseSpecHeaders(t *testing.T) {
	body := `# ` + "`elasticstack_elasticsearch_security_role`" + ` — Schema and Functional Requirements

Resource implementation: ` + "`internal/elasticsearch/security/role`" + `
Data source implementation: ` + "`internal/elasticsearch/security/role_data_source.go`" + `

## Purpose

Some purpose.

## Schema

` + "```hcl" + `
resource "x" "y" {}
` + "```" + `
`
	ent := parseSpec("elasticsearch-security-role", "/tmp/spec.md", body)
	if ent == nil {
		t.Fatal("entity nil")
	}
	if ent.Name != "elasticstack_elasticsearch_security_role" {
		t.Errorf("Name = %q", ent.Name)
	}
	if !ent.Kinds.has(kindResource) || !ent.Kinds.has(kindDataSource) {
		t.Errorf("Kinds = %d, expected both", ent.Kinds)
	}
	if !strings.Contains(ent.ResourceImpl, "internal/elasticsearch/security/role") {
		t.Errorf("ResourceImpl = %q", ent.ResourceImpl)
	}
	if !strings.Contains(ent.Schema, "resource \"x\"") {
		t.Errorf("Schema = %q", ent.Schema)
	}
}

func TestParseDocsFrontmatter(t *testing.T) {
	body := `---
subcategory: "Security"
page_title: "something"
description: |-
  Adds and updates roles in the native realm. See the role API documentation https://example for more details.
---

# Something

## Example Usage

` + "```terraform" + `
provider "elasticstack" {}
resource "x" "y" {}
` + "```" + `
`
	meta, example := parseDocsPage(body)
	if meta.Subcategory != "Security" {
		t.Errorf("Subcategory = %q", meta.Subcategory)
	}
	if !strings.HasPrefix(meta.Description, "Adds and updates roles") {
		t.Errorf("Description = %q", meta.Description)
	}
	if !strings.Contains(example, `resource "x"`) {
		t.Errorf("Example = %q", example)
	}
}

func TestOneLineStripsSeeURL(t *testing.T) {
	in := "Creates Elasticsearch indices. See: https://example/docs"
	want := "Creates Elasticsearch indices."
	if got := oneLine(in); got != want {
		t.Errorf("oneLine = %q, want %q", got, want)
	}
}
