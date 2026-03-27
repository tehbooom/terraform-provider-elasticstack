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

package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/terraform-provider-elasticstack/internal/clients"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients/kibanaoapi"
	"github.com/elastic/terraform-provider-elasticstack/internal/diagutil"
	"github.com/elastic/terraform-provider-elasticstack/internal/models"
	"github.com/elastic/terraform-provider-elasticstack/internal/utils/customtypes"
	"github.com/elastic/terraform-provider-elasticstack/internal/utils/typeutils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Read refreshes the Terraform state with the latest data.
func (d *DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config dataSourceModel

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	supported, sdkDiags := d.client.EnforceMinVersion(ctx, minKibanaAgentBuilderAPIVersion)
	resp.Diagnostics.Append(diagutil.FrameworkDiagsFromSDK(sdkDiags)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !supported {
		resp.Diagnostics.AddError("Unsupported server version",
			fmt.Sprintf("Agent Builder agents require Elastic Stack v%s or later.", minKibanaAgentBuilderAPIVersion))
		return
	}

	oapiClient, err := d.client.GetKibanaOapiClient()
	if err != nil {
		resp.Diagnostics.AddError("unable to get Kibana client", err.Error())
		return
	}

	spaceID := "default"
	if typeutils.IsKnown(config.SpaceID) {
		spaceID = config.SpaceID.ValueString()
	}

	agentID := config.ID.ValueString()

	agent, agentDiags := kibanaoapi.GetAgent(ctx, oapiClient, spaceID, agentID)
	resp.Diagnostics.Append(agentDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if agent == nil {
		resp.Diagnostics.AddError("Agent not found", fmt.Sprintf("Unable to fetch agent with ID %s", agentID))
		return
	}

	agentJSON, err := json.Marshal(agent)
	if err != nil {
		resp.Diagnostics.AddError("JSON marshaling failed", fmt.Sprintf("Unable to marshal agent to JSON: %v", err))
		return
	}

	compositeID := &clients.CompositeID{ClusterID: spaceID, ResourceID: agentID}

	var state dataSourceModel
	state.ID = types.StringValue(compositeID.String())
	state.SpaceID = types.StringValue(spaceID)
	state.Agent = types.StringValue(string(agentJSON))
	state.IncludeDependencies = config.IncludeDependencies
	state.Tools = []toolModel{}

	includeDeps := typeutils.IsKnown(config.IncludeDependencies) && config.IncludeDependencies.ValueBool()

	if includeDeps {
		// Collect all tool IDs from the agent configuration.
		toolIDSet := make(map[string]struct{})
		for _, toolsConfig := range agent.Configuration.Tools {
			for _, id := range toolsConfig.ToolIDs {
				toolIDSet[id] = struct{}{}
			}
		}

		// Fetch each tool and track workflow IDs for workflow-type tools.
		workflowIDSet := make(map[string]struct{})
		toolsByID := make(map[string]*models.Tool)

		for toolID := range toolIDSet {
			tool, toolDiags := kibanaoapi.GetTool(ctx, oapiClient, spaceID, toolID)
			resp.Diagnostics.Append(toolDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			if tool == nil {
				continue
			}
			toolsByID[toolID] = tool

			if tool.Type == "workflow" {
				if workflowID, ok := tool.Configuration["workflow_id"].(string); ok && workflowID != "" {
					workflowIDSet[workflowID] = struct{}{}
				}
			}
		}

		// Fetch each referenced workflow.
		workflowsByID := make(map[string]*models.Workflow)
		for workflowID := range workflowIDSet {
			workflow, wDiags := kibanaoapi.GetWorkflow(ctx, oapiClient, spaceID, workflowID)
			resp.Diagnostics.Append(wDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			if workflow != nil {
				workflowsByID[workflowID] = workflow
			}
		}

		// Convert to state models.
		for _, tool := range toolsByID {
			tm, tmDiags := toolModelFromAPI(ctx, tool, workflowsByID)
			resp.Diagnostics.Append(tmDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			state.Tools = append(state.Tools, tm)
		}
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// toolModelFromAPI converts a models.Tool (and optionally its workflow) into a toolModel.
func toolModelFromAPI(ctx context.Context, tool *models.Tool, workflowsByID map[string]*models.Workflow) (toolModel, diag.Diagnostics) {
	var tm toolModel
	var diags diag.Diagnostics

	tm.ID = types.StringValue(tool.ID)
	tm.Type = types.StringValue(tool.Type)

	if tool.Description != nil {
		tm.Description = types.StringValue(*tool.Description)
	} else {
		tm.Description = types.StringNull()
	}

	if len(tool.Tags) > 0 {
		tags, tagDiags := types.ListValueFrom(ctx, types.StringType, tool.Tags)
		diags.Append(tagDiags...)
		tm.Tags = tags
	} else {
		tm.Tags = types.ListNull(types.StringType)
	}

	tm.ReadOnly = types.BoolValue(tool.ReadOnly)

	if tool.Configuration != nil {
		configJSON, err := json.Marshal(tool.Configuration)
		if err != nil {
			diags.AddError("Configuration Error", "Failed to marshal configuration to JSON: "+err.Error())
			return tm, diags
		}
		tm.Configuration = types.StringValue(string(configJSON))
	} else {
		tm.Configuration = types.StringNull()
	}

	if tool.Type == "workflow" {
		if workflowID, ok := tool.Configuration["workflow_id"].(string); ok && workflowID != "" {
			tm.WorkflowID = types.StringValue(workflowID)
			if workflow, found := workflowsByID[workflowID]; found {
				tm.WorkflowConfigurationYaml = customtypes.NewNormalizedYamlValue(workflow.Yaml)
			} else {
				tm.WorkflowConfigurationYaml = customtypes.NewNormalizedYamlNull()
			}
		} else {
			tm.WorkflowID = types.StringNull()
			tm.WorkflowConfigurationYaml = customtypes.NewNormalizedYamlNull()
		}
	} else {
		tm.WorkflowID = types.StringNull()
		tm.WorkflowConfigurationYaml = customtypes.NewNormalizedYamlNull()
	}

	return tm, diags
}
