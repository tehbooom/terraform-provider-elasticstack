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

package agent_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/elastic/terraform-provider-elasticstack/internal/acctest"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients"
	"github.com/elastic/terraform-provider-elasticstack/internal/versionutils"
	"github.com/google/uuid"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var (
	minKibanaAgentBuilderAPIVersion = version.Must(version.NewVersion("9.3.0"))
)

func preCheckWithWorkflowsEnabled(t *testing.T) {
	acctest.PreCheck(t)

	client, err := clients.NewAcceptanceTestingClient()
	if err != nil {
		t.Fatalf("Failed to create API client: %v", err)
	}

	serverVersion, diags := client.ServerVersion(context.Background())
	if diags.HasError() {
		t.Fatalf("Failed to get server version: %v", diags)
	}
	if serverVersion.LessThan(minKibanaAgentBuilderAPIVersion) {
		t.Skipf("Skipping test: server version %s is below minimum %s", serverVersion, minKibanaAgentBuilderAPIVersion)
	}

	kibanaClient, err := client.GetKibanaOapiClient()
	if err != nil {
		t.Fatalf("Failed to get Kibana client: %v", err)
	}

	settingsURL := fmt.Sprintf("%s/internal/kibana/settings/workflows:ui:enabled", kibanaClient.URL)
	body := map[string]any{
		"value": true,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal body: %v", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", settingsURL, bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("Failed to create POST request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("kbn-xsrf", "true")
	req.Header.Set("x-elastic-internal-origin", "Kibana")

	resp, err := kibanaClient.HTTP.Do(req)
	if err != nil {
		t.Fatalf("Failed to enable workflows: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to enable workflows (status %d): %s. Make sure workflows are enabled in kibana.yml with 'xpack.aiAssistant.workflows.enabled: true'", resp.StatusCode, string(respBody))
	}
}

// TestAccDataSourceKibanaExportABAgent tests exporting an agent without dependencies.
func TestAccDataSourceKibanaExportABAgent(t *testing.T) {
	agentID := "test-agent-" + uuid.New().String()[:8]

	resource.Test(t, resource.TestCase{
		PreCheck: func() { preCheckWithWorkflowsEnabled(t) },
		Steps: []resource.TestStep{
			{
				SkipFunc:                 versionutils.CheckIfVersionIsUnsupported(minKibanaAgentBuilderAPIVersion),
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("read"),
				ConfigVariables: config.Variables{
					"agent_id": config.StringVariable(agentID),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.elasticstack_kibana_agentbuilder_export_agent.test", "id"),
					resource.TestCheckResourceAttrSet("data.elasticstack_kibana_agentbuilder_export_agent.test", "agent"),
					resource.TestCheckResourceAttr("data.elasticstack_kibana_agentbuilder_export_agent.test", "include_dependencies", "false"),
					resource.TestCheckResourceAttr("data.elasticstack_kibana_agentbuilder_export_agent.test", "tools.#", "0"),
				),
			},
		},
	})
}

// TestAccDataSourceKibanaExportABAgentWithDependencies tests exporting an agent with its tools and workflows.
func TestAccDataSourceKibanaExportABAgentWithDependencies(t *testing.T) {
	agentID := "test-agent-deps-" + uuid.New().String()[:8]
	esqlToolID := "test-esql-tool-" + uuid.New().String()[:8]
	workflowToolID := "test-wf-tool-" + uuid.New().String()[:8]

	resource.Test(t, resource.TestCase{
		PreCheck: func() { preCheckWithWorkflowsEnabled(t) },
		Steps: []resource.TestStep{
			{
				SkipFunc:                 versionutils.CheckIfVersionIsUnsupported(minKibanaAgentBuilderAPIVersion),
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("read_with_deps"),
				ConfigVariables: config.Variables{
					"agent_id":         config.StringVariable(agentID),
					"esql_tool_id":     config.StringVariable(esqlToolID),
					"workflow_tool_id": config.StringVariable(workflowToolID),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.elasticstack_kibana_agentbuilder_export_agent.test", "id"),
					resource.TestCheckResourceAttrSet("data.elasticstack_kibana_agentbuilder_export_agent.test", "agent"),
					resource.TestCheckResourceAttr("data.elasticstack_kibana_agentbuilder_export_agent.test", "include_dependencies", "true"),
					// Both tools should be exported
					resource.TestCheckResourceAttr("data.elasticstack_kibana_agentbuilder_export_agent.test", "tools.#", "2"),
					// The workflow tool should have workflow_id and workflow_configuration_yaml set
					resource.TestCheckResourceAttrSet("data.elasticstack_kibana_agentbuilder_export_agent.test", "tools.0.workflow_id"),
					resource.TestCheckResourceAttrSet("data.elasticstack_kibana_agentbuilder_export_agent.test", "tools.0.workflow_configuration_yaml"),
				),
			},
		},
	})
}
