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

func (m *tfModel) populateFromAPI(ctx context.Context, api *esclient.ConnectorResponse) diag.Diagnostics {
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
		if full.Interval.ValueString() != "" {
			s.Full = &esclient.ConnectorSchedule{
				Enabled:  full.Enabled.ValueBool(),
				Interval: full.Interval.ValueString(),
			}
		}
	}

	if !scheduling.Incremental.IsNull() && !scheduling.Incremental.IsUnknown() {
		var incremental tfSchedule
		diags.Append(scheduling.Incremental.As(ctx, &incremental, basetypes.ObjectAsOptions{})...)
		if incremental.Interval.ValueString() != "" {
			s.Incremental = &esclient.ConnectorSchedule{
				Enabled:  incremental.Enabled.ValueBool(),
				Interval: incremental.Interval.ValueString(),
			}
		}
	}

	if !scheduling.AccessControl.IsNull() && !scheduling.AccessControl.IsUnknown() {
		var accessControl tfSchedule
		diags.Append(scheduling.AccessControl.As(ctx, &accessControl, basetypes.ObjectAsOptions{})...)
		if accessControl.Interval.ValueString() != "" {
			s.AccessControl = &esclient.ConnectorSchedule{
				Enabled:  accessControl.Enabled.ValueBool(),
				Interval: accessControl.Interval.ValueString(),
			}
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
