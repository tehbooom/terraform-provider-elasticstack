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

package agentbuilderagent

import (
	"context"

	"github.com/elastic/terraform-provider-elasticstack/generated/kbapi"
	"github.com/elastic/terraform-provider-elasticstack/internal/models"
	"github.com/elastic/terraform-provider-elasticstack/internal/utils/typeutils"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type agentModel struct {
	ID                        types.String `tfsdk:"id"`
	SpaceID                   types.String `tfsdk:"space_id"`
	Name                      types.String `tfsdk:"name"`
	Description               types.String `tfsdk:"description"`
	AvatarColor               types.String `tfsdk:"avatar_color"`
	AvatarSymbol              types.String `tfsdk:"avatar_symbol"`
	Labels                    types.List   `tfsdk:"labels"` // []string
	Tools                     types.List   `tfsdk:"tools"`  // []string
	Instructions              types.String `tfsdk:"instructions"`
	EnableElasticCapabilities types.Bool   `tfsdk:"enable_elastic_capabilities"`
	PluginIDs                 types.List   `tfsdk:"plugin_ids"`   // []string
	SkillIDs                  types.List   `tfsdk:"skill_ids"`    // []string
	WorkflowIDs               types.List   `tfsdk:"workflow_ids"` // []string
}

func (model *agentModel) spaceID() string {
	if typeutils.IsKnown(model.SpaceID) {
		return model.SpaceID.ValueString()
	}
	return "default"
}

func (model *agentModel) populateFromAPI(ctx context.Context, data *models.Agent) diag.Diagnostics {
	if data == nil {
		return nil
	}

	var diags diag.Diagnostics

	model.ID = types.StringValue(data.ID)
	model.Name = types.StringValue(data.Name)

	if data.Description != nil && *data.Description != "" {
		model.Description = types.StringValue(*data.Description)
	} else {
		model.Description = types.StringNull()
	}

	if data.AvatarColor != nil && *data.AvatarColor != "" {
		model.AvatarColor = types.StringValue(*data.AvatarColor)
	} else {
		model.AvatarColor = types.StringNull()
	}

	if data.AvatarSymbol != nil && *data.AvatarSymbol != "" {
		model.AvatarSymbol = types.StringValue(*data.AvatarSymbol)
	} else {
		model.AvatarSymbol = types.StringNull()
	}

	cfg := data.Configuration

	if cfg.Instructions != nil && *cfg.Instructions != "" {
		model.Instructions = types.StringValue(*cfg.Instructions)
	} else {
		model.Instructions = types.StringNull()
	}

	if cfg.EnableElasticCapabilities != nil {
		model.EnableElasticCapabilities = types.BoolValue(*cfg.EnableElasticCapabilities)
	} else {
		model.EnableElasticCapabilities = types.BoolNull()
	}

	diags.Append(populateList(ctx, cfg.PluginIDs, &model.PluginIDs)...)
	diags.Append(populateList(ctx, cfg.SkillIDs, &model.SkillIDs)...)
	diags.Append(populateList(ctx, cfg.WorkflowIDs, &model.WorkflowIDs)...)
	diags.Append(populateList(ctx, data.Labels, &model.Labels)...)

	// Extract tool IDs from nested configuration.
	var toolIDs []string
	if len(cfg.Tools) > 0 {
		toolIDs = cfg.Tools[0].ToolIDs
	}
	diags.Append(populateList(ctx, toolIDs, &model.Tools)...)

	return diags
}

func populateList(ctx context.Context, src []string, dst *types.List) diag.Diagnostics {
	if len(src) > 0 {
		v, d := types.ListValueFrom(ctx, types.StringType, src)
		*dst = v
		return d
	}
	*dst = types.ListNull(types.StringType)
	return nil
}

func (model agentModel) toAPICreateModel(ctx context.Context) (kbapi.PostAgentBuilderAgentsJSONRequestBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	body := kbapi.PostAgentBuilderAgentsJSONRequestBody{
		Id:          model.ID.ValueString(),
		Name:        model.Name.ValueString(),
		Description: model.Description.ValueString(),
	}

	if typeutils.IsKnown(model.AvatarColor) {
		body.AvatarColor = model.AvatarColor.ValueStringPointer()
	}
	if typeutils.IsKnown(model.AvatarSymbol) {
		body.AvatarSymbol = model.AvatarSymbol.ValueStringPointer()
	}

	if typeutils.IsKnown(model.Instructions) {
		body.Configuration.Instructions = model.Instructions.ValueStringPointer()
	}
	if typeutils.IsKnown(model.EnableElasticCapabilities) {
		body.Configuration.EnableElasticCapabilities = model.EnableElasticCapabilities.ValueBoolPointer()
	}

	pluginIDs, d := listToStrings(ctx, model.PluginIDs)
	diags.Append(d...)
	if len(pluginIDs) > 0 {
		body.Configuration.PluginIds = &pluginIDs
	}

	skillIDs, d := listToStrings(ctx, model.SkillIDs)
	diags.Append(d...)
	if len(skillIDs) > 0 {
		body.Configuration.SkillIds = &skillIDs
	}

	workflowIDs, d := listToStrings(ctx, model.WorkflowIDs)
	diags.Append(d...)
	if len(workflowIDs) > 0 {
		body.Configuration.WorkflowIds = &workflowIDs
	}

	toolIDs, d := listToStrings(ctx, model.Tools)
	diags.Append(d...)
	body.Configuration.Tools = []struct {
		ToolIds []string `json:"tool_ids"` //nolint:revive
	}{{ToolIds: toolIDs}}

	labels, d := listToStrings(ctx, model.Labels)
	diags.Append(d...)
	if len(labels) > 0 {
		body.Labels = &labels
	}

	return body, diags
}

func (model agentModel) toAPIUpdateModel(ctx context.Context) (kbapi.PutAgentBuilderAgentsIdJSONRequestBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	name := model.Name.ValueString()
	body := kbapi.PutAgentBuilderAgentsIdJSONRequestBody{
		Name: &name,
	}

	if typeutils.IsKnown(model.Description) {
		body.Description = model.Description.ValueStringPointer()
	}
	if typeutils.IsKnown(model.AvatarColor) {
		body.AvatarColor = model.AvatarColor.ValueStringPointer()
	}
	if typeutils.IsKnown(model.AvatarSymbol) {
		body.AvatarSymbol = model.AvatarSymbol.ValueStringPointer()
	}

	cfg := &struct {
		EnableElasticCapabilities *bool     `json:"enable_elastic_capabilities,omitempty"`
		Instructions              *string   `json:"instructions,omitempty"`
		PluginIds                 *[]string `json:"plugin_ids,omitempty"` //nolint:revive
		SkillIds                  *[]string `json:"skill_ids,omitempty"`  //nolint:revive
		Tools                     *[]struct {
			ToolIds []string `json:"tool_ids"` //nolint:revive
		} `json:"tools,omitempty"`
		WorkflowIds *[]string `json:"workflow_ids,omitempty"` //nolint:revive
	}{}

	if typeutils.IsKnown(model.Instructions) {
		cfg.Instructions = model.Instructions.ValueStringPointer()
	}
	if typeutils.IsKnown(model.EnableElasticCapabilities) {
		cfg.EnableElasticCapabilities = model.EnableElasticCapabilities.ValueBoolPointer()
	}

	pluginIDs, d := listToStrings(ctx, model.PluginIDs)
	diags.Append(d...)
	if len(pluginIDs) > 0 {
		cfg.PluginIds = &pluginIDs
	}

	skillIDs, d := listToStrings(ctx, model.SkillIDs)
	diags.Append(d...)
	if len(skillIDs) > 0 {
		cfg.SkillIds = &skillIDs
	}

	workflowIDs, d := listToStrings(ctx, model.WorkflowIDs)
	diags.Append(d...)
	if len(workflowIDs) > 0 {
		cfg.WorkflowIds = &workflowIDs
	}

	toolIDs, d := listToStrings(ctx, model.Tools)
	diags.Append(d...)
	tools := []struct {
		ToolIds []string `json:"tool_ids"` //nolint:revive
	}{{ToolIds: toolIDs}}
	cfg.Tools = &tools

	body.Configuration = cfg

	labels, d := listToStrings(ctx, model.Labels)
	diags.Append(d...)
	if len(labels) > 0 {
		body.Labels = &labels
	}

	return body, diags
}

func listToStrings(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return []string{}, nil
	}
	var out []string
	d := list.ElementsAs(ctx, &out, false)
	return out, d
}
