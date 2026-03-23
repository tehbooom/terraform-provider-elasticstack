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
	"encoding/json"

	esclient "github.com/elastic/terraform-provider-elasticstack/internal/clients/elasticsearch"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type tfModel struct {
	ElasticsearchConnection types.List           `tfsdk:"elasticsearch_connection"`
	ConnectorID             types.String         `tfsdk:"connector_id"`
	Name                    types.String         `tfsdk:"name"`
	Description             types.String         `tfsdk:"description"`
	IndexName               types.String         `tfsdk:"index_name"`
	ServiceType             types.String         `tfsdk:"service_type"`
	Configuration           jsontypes.Normalized `tfsdk:"configuration"`
	Scheduling              *tfScheduling        `tfsdk:"scheduling"`
	Pipeline                *tfPipeline          `tfsdk:"pipeline"`
	APIKeyID                types.String         `tfsdk:"api_key_id"`
	APIKeySecretID          types.String         `tfsdk:"api_key_secret_id"`
	Status                  types.String         `tfsdk:"status"`
}

type tfScheduling struct {
	Full          *tfSchedule `tfsdk:"full"`
	Incremental   *tfSchedule `tfsdk:"incremental"`
	AccessControl *tfSchedule `tfsdk:"access_control"`
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

func (m *tfModel) populateFromAPI(api *esclient.ConnectorResponse) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ConnectorID = types.StringValue(api.ID)
	m.Name = types.StringValue(api.Name)
	m.Description = types.StringValue(api.Description)
	m.IndexName = types.StringValue(api.IndexName)
	m.ServiceType = types.StringValue(api.ServiceType)
	m.Status = types.StringValue(api.Status)
	m.APIKeyID = types.StringValue(api.APIKeyID)
	m.APIKeySecretID = types.StringValue(api.APIKeySecretID)

	if api.Configuration != nil {
		configBytes, err := json.Marshal(api.Configuration)
		if err != nil {
			diags.AddError("Failed to marshal configuration", err.Error())
			return diags
		}
		m.Configuration = jsontypes.NewNormalizedValue(string(configBytes))
	}

	if api.Scheduling != nil {
		m.Scheduling = &tfScheduling{}
		if api.Scheduling.Full != nil {
			m.Scheduling.Full = &tfSchedule{
				Enabled:  types.BoolValue(api.Scheduling.Full.Enabled),
				Interval: types.StringValue(api.Scheduling.Full.Interval),
			}
		}
		if api.Scheduling.Incremental != nil {
			m.Scheduling.Incremental = &tfSchedule{
				Enabled:  types.BoolValue(api.Scheduling.Incremental.Enabled),
				Interval: types.StringValue(api.Scheduling.Incremental.Interval),
			}
		}
		if api.Scheduling.AccessControl != nil {
			m.Scheduling.AccessControl = &tfSchedule{
				Enabled:  types.BoolValue(api.Scheduling.AccessControl.Enabled),
				Interval: types.StringValue(api.Scheduling.AccessControl.Interval),
			}
		}
	}

	if api.Pipeline != nil {
		m.Pipeline = &tfPipeline{
			Name:                 types.StringValue(api.Pipeline.Name),
			ExtractBinaryContent: types.BoolValue(api.Pipeline.ExtractBinaryContent),
			ReduceWhitespace:     types.BoolValue(api.Pipeline.ReduceWhitespace),
			RunMlInference:       types.BoolValue(api.Pipeline.RunMlInference),
		}
	}

	return diags
}

func (m *tfModel) toSchedulingAPI() *esclient.ConnectorScheduling {
	if m.Scheduling == nil {
		return nil
	}
	s := &esclient.ConnectorScheduling{}
	if m.Scheduling.Full != nil {
		s.Full = &esclient.ConnectorSchedule{
			Enabled:  m.Scheduling.Full.Enabled.ValueBool(),
			Interval: m.Scheduling.Full.Interval.ValueString(),
		}
	}
	if m.Scheduling.Incremental != nil {
		s.Incremental = &esclient.ConnectorSchedule{
			Enabled:  m.Scheduling.Incremental.Enabled.ValueBool(),
			Interval: m.Scheduling.Incremental.Interval.ValueString(),
		}
	}
	if m.Scheduling.AccessControl != nil {
		s.AccessControl = &esclient.ConnectorSchedule{
			Enabled:  m.Scheduling.AccessControl.Enabled.ValueBool(),
			Interval: m.Scheduling.AccessControl.Interval.ValueString(),
		}
	}
	return s
}

func (m *tfModel) toPipelineAPI() *esclient.ConnectorPipeline {
	if m.Pipeline == nil {
		return nil
	}
	return &esclient.ConnectorPipeline{
		Name:                 m.Pipeline.Name.ValueString(),
		ExtractBinaryContent: m.Pipeline.ExtractBinaryContent.ValueBool(),
		ReduceWhitespace:     m.Pipeline.ReduceWhitespace.ValueBool(),
		RunMlInference:       m.Pipeline.RunMlInference.ValueBool(),
	}
}
