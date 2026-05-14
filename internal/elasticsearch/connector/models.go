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

package connector

import (
	"context"
	"encoding/json"

	esclient "github.com/elastic/terraform-provider-elasticstack/internal/clients/elasticsearch"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type tfModel struct {
	ElasticsearchConnection types.List           `tfsdk:"elasticsearch_connection"`
	ConnectorID             types.String         `tfsdk:"connector_id"`
	Name                    types.String         `tfsdk:"name"`
	Description             types.String         `tfsdk:"description"`
	IndexName               types.String         `tfsdk:"index_name"`
	ServiceType             types.String         `tfsdk:"service_type"`
	Configuration           jsontypes.Normalized `tfsdk:"configuration"`
	Scheduling              types.Object         `tfsdk:"scheduling"`
	Pipeline                types.Object         `tfsdk:"pipeline"`
	APIKeyID                types.String         `tfsdk:"api_key_id"`
	APIKeySecretID          types.String         `tfsdk:"api_key_secret_id"`
	Status                  types.String         `tfsdk:"status"`
}

// tfScheduling is used only for As() conversions, not stored directly in tfModel.
type tfScheduling struct {
	Full          types.Object `tfsdk:"full"`
	Incremental   types.Object `tfsdk:"incremental"`
	AccessControl types.Object `tfsdk:"access_control"`
}

type tfSchedule struct {
	Enabled  types.Bool   `tfsdk:"enabled"`
	Interval types.String `tfsdk:"interval"`
}

type tfPipeline struct {
	Name                 types.String `tfsdk:"name"`
	ExtractBinaryContent types.Bool   `tfsdk:"extract_binary_content"`
	ReduceWhitespace     types.Bool   `tfsdk:"reduce_whitespace"`
	RunMlInference       types.Bool   `tfsdk:"run_ml_inference"`
}

var scheduleAttrTypes = map[string]attr.Type{
	"enabled":  types.BoolType,
	"interval": types.StringType,
}

var schedulingAttrTypes = map[string]attr.Type{
	"full":           types.ObjectType{AttrTypes: scheduleAttrTypes},
	"incremental":    types.ObjectType{AttrTypes: scheduleAttrTypes},
	"access_control": types.ObjectType{AttrTypes: scheduleAttrTypes},
}

var pipelineAttrTypes = map[string]attr.Type{
	"name":                   types.StringType,
	"extract_binary_content": types.BoolType,
	"reduce_whitespace":      types.BoolType,
	"run_ml_inference":       types.BoolType,
}

// stripCompositeIDPrefix removes the "cluster_uuid/" prefix that the connector API
// returns for api_key_id and api_key_secret_id, leaving just the raw key ID.
func stripCompositeIDPrefix(id string) string {
	for i := len(id) - 1; i >= 0; i-- {
		if id[i] == '/' {
			return id[i+1:]
		}
	}
	return id
}

// applyConnectorConfigurationFromPlan restores the configuration JSON from the plan/state after
// read: GET may reorder, normalize, or redact sensitive `value` fields so the body will not
// match the exact string Terraform validated at apply time.
func applyConnectorConfigurationFromPlan(model *tfModel, configurationFromPlan jsontypes.Normalized) {
	if !configurationFromPlan.IsNull() && !configurationFromPlan.IsUnknown() {
		model.Configuration = configurationFromPlan
	}
}

// isSchedulingDisabled checks if all schedules in a scheduling object are disabled.
func isSchedulingDisabled(ctx context.Context, scheduling types.Object) bool {
	var sched tfScheduling
	if err := scheduling.As(ctx, &sched, basetypes.ObjectAsOptions{}); err != nil {
		return false
	}

	checkDisabled := func(obj types.Object) bool {
		if obj.IsNull() || obj.IsUnknown() {
			return true
		}
		var s tfSchedule
		if err := obj.As(ctx, &s, basetypes.ObjectAsOptions{}); err != nil {
			return false
		}
		return !s.Enabled.ValueBool()
	}

	return checkDisabled(sched.Full) && checkDisabled(sched.Incremental) && checkDisabled(sched.AccessControl)
}

// mergeConnectorConfiguration takes the prior configuration from state and the new configuration
// from the API. It restores any redacted sensitive `value` fields from the prior configuration
// into the new configuration so that Terraform can accurately detect drift on non-sensitive fields
// without producing perpetual diffs on redacted sensitive fields.
func mergeConnectorConfiguration(prior, remote jsontypes.Normalized) jsontypes.Normalized {
	if prior.IsNull() || prior.IsUnknown() || prior.ValueString() == "" {
		return remote
	}
	if remote.IsNull() || remote.IsUnknown() || remote.ValueString() == "" {
		return prior
	}

	var priorMap map[string]any
	if err := json.Unmarshal([]byte(prior.ValueString()), &priorMap); err != nil {
		return remote // fallback to remote if prior is malformed
	}

	var remoteMap map[string]any
	if err := json.Unmarshal([]byte(remote.ValueString()), &remoteMap); err != nil {
		return remote
	}

	for key, remoteFieldVal := range remoteMap {
		remoteField, ok := remoteFieldVal.(map[string]any)
		if !ok {
			continue
		}

		isSensitive := false
		if s, ok := remoteField["sensitive"]; ok {
			if b, ok := s.(bool); ok {
				isSensitive = b
			}
		}

		if isSensitive {
			if priorFieldVal, ok := priorMap[key]; ok {
				if priorField, ok := priorFieldVal.(map[string]any); ok {
					if priorValue, hasPriorValue := priorField["value"]; hasPriorValue {
						remoteField["value"] = priorValue
					}
				}
			}
		}
	}

	mergedBytes, err := json.Marshal(remoteMap)
	if err != nil {
		return remote
	}
	return jsontypes.NewNormalizedValue(string(mergedBytes))
}

func scheduleToObject(s *esclient.ConnectorSchedule) types.Object {
	if s == nil {
		return types.ObjectNull(scheduleAttrTypes)
	}
	obj, _ := types.ObjectValue(scheduleAttrTypes, map[string]attr.Value{
		"enabled":  types.BoolValue(s.Enabled),
		"interval": types.StringValue(s.Interval),
	})
	return obj
}

func (m *tfModel) populateFromAPI(_ context.Context, api *esclient.ConnectorResponse) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ConnectorID = types.StringValue(api.ID)
	m.Name = types.StringValue(api.Name)
	m.Description = types.StringValue(api.Description)
	m.IndexName = types.StringValue(api.IndexName)
	m.ServiceType = types.StringValue(api.ServiceType)
	m.Status = types.StringValue(api.Status)
	m.APIKeyID = types.StringValue(stripCompositeIDPrefix(api.APIKeyID))
	m.APIKeySecretID = types.StringValue(stripCompositeIDPrefix(api.APIKeySecretID))

	if api.Configuration != nil {
		configBytes, err := json.Marshal(api.Configuration)
		if err != nil {
			diags.AddError("Failed to marshal configuration", err.Error())
			return diags
		}
		m.Configuration = jsontypes.NewNormalizedValue(string(configBytes))
	}

	if api.Scheduling != nil {
		schedulingObj, d := types.ObjectValue(schedulingAttrTypes, map[string]attr.Value{
			"full":           scheduleToObject(api.Scheduling.Full),
			"incremental":    scheduleToObject(api.Scheduling.Incremental),
			"access_control": scheduleToObject(api.Scheduling.AccessControl),
		})
		diags.Append(d...)
		m.Scheduling = schedulingObj
	} else {
		m.Scheduling = types.ObjectNull(schedulingAttrTypes)
	}

	if api.Pipeline != nil {
		pipelineObj, d := types.ObjectValue(pipelineAttrTypes, map[string]attr.Value{
			"name":                   types.StringValue(api.Pipeline.Name),
			"extract_binary_content": types.BoolValue(api.Pipeline.ExtractBinaryContent),
			"reduce_whitespace":      types.BoolValue(api.Pipeline.ReduceWhitespace),
			"run_ml_inference":       types.BoolValue(api.Pipeline.RunMlInference),
		})
		diags.Append(d...)
		m.Pipeline = pipelineObj
	} else {
		m.Pipeline = types.ObjectNull(pipelineAttrTypes)
	}

	return diags
}

func (m *tfModel) toSchedulingAPI(ctx context.Context) (*esclient.ConnectorScheduling, diag.Diagnostics) {
	var diags diag.Diagnostics

	if m.Scheduling.IsNull() || m.Scheduling.IsUnknown() {
		return nil, diags
	}

	var scheduling tfScheduling
	diags.Append(m.Scheduling.As(ctx, &scheduling, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	s := &esclient.ConnectorScheduling{}

	if !scheduling.Full.IsNull() && !scheduling.Full.IsUnknown() {
		var full tfSchedule
		diags.Append(scheduling.Full.As(ctx, &full, basetypes.ObjectAsOptions{})...)
		s.Full = &esclient.ConnectorSchedule{
			Enabled:  full.Enabled.ValueBool(),
			Interval: full.Interval.ValueString(),
		}
	}

	if !scheduling.Incremental.IsNull() && !scheduling.Incremental.IsUnknown() {
		var incremental tfSchedule
		diags.Append(scheduling.Incremental.As(ctx, &incremental, basetypes.ObjectAsOptions{})...)
		s.Incremental = &esclient.ConnectorSchedule{
			Enabled:  incremental.Enabled.ValueBool(),
			Interval: incremental.Interval.ValueString(),
		}
	}

	if !scheduling.AccessControl.IsNull() && !scheduling.AccessControl.IsUnknown() {
		var accessControl tfSchedule
		diags.Append(scheduling.AccessControl.As(ctx, &accessControl, basetypes.ObjectAsOptions{})...)
		s.AccessControl = &esclient.ConnectorSchedule{
			Enabled:  accessControl.Enabled.ValueBool(),
			Interval: accessControl.Interval.ValueString(),
		}
	}

	return s, diags
}

func (m *tfModel) toPipelineAPI(ctx context.Context) (*esclient.ConnectorPipeline, diag.Diagnostics) {
	var diags diag.Diagnostics

	if m.Pipeline.IsNull() || m.Pipeline.IsUnknown() {
		return nil, diags
	}

	var pipeline tfPipeline
	diags.Append(m.Pipeline.As(ctx, &pipeline, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	return &esclient.ConnectorPipeline{
		Name:                 pipeline.Name.ValueString(),
		ExtractBinaryContent: pipeline.ExtractBinaryContent.ValueBool(),
		ReduceWhitespace:     pipeline.ReduceWhitespace.ValueBool(),
		RunMlInference:       pipeline.RunMlInference.ValueBool(),
	}, diags
}
