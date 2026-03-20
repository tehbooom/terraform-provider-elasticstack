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

	"github.com/elastic/terraform-provider-elasticstack/internal/clients"
	esclient "github.com/elastic/terraform-provider-elasticstack/internal/clients/elasticsearch"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan tfModel
	var state tfModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := clients.MaybeNewAPIClientFromFrameworkResource(ctx, plan.ElasticsearchConnection, r.client)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	connectorID := state.ConnectorID.ValueString()
	plan.ConnectorID = state.ConnectorID

	// Update name/description if changed
	if !plan.Name.Equal(state.Name) || !plan.Description.Equal(state.Description) {
		resp.Diagnostics.Append(esclient.UpdateConnectorName(ctx, client, connectorID,
			plan.Name.ValueString(), plan.Description.ValueString())...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Update index name if changed
	if !plan.IndexName.Equal(state.IndexName) {
		resp.Diagnostics.Append(esclient.UpdateConnectorIndexName(ctx, client, connectorID, plan.IndexName.ValueString())...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Update configuration if changed
	if !plan.Configuration.Equal(state.Configuration) {
		if !plan.Configuration.IsNull() && !plan.Configuration.IsUnknown() {
			resp.Diagnostics.Append(esclient.UpdateConnectorConfiguration(ctx, client, connectorID, plan.Configuration.ValueString())...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	// Update scheduling if changed
	if schedulingChanged(plan.Scheduling, state.Scheduling) {
		if plan.Scheduling != nil {
			resp.Diagnostics.Append(esclient.UpdateConnectorScheduling(ctx, client, connectorID, plan.toSchedulingAPI())...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	// Update pipeline if changed
	if pipelineChanged(plan.Pipeline, state.Pipeline) {
		if plan.Pipeline != nil {
			resp.Diagnostics.Append(esclient.UpdateConnectorPipeline(ctx, client, connectorID, plan.toPipelineAPI())...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	// Read back to populate computed fields
	exists, diags := r.readFromAPI(ctx, client, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !exists {
		resp.Diagnostics.AddError("Connector not found after update", "The connector was updated but could not be read back")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func schedulingChanged(plan, state *tfScheduling) bool {
	if plan == nil && state == nil {
		return false
	}
	if plan == nil || state == nil {
		return true
	}
	return scheduleChanged(plan.Full, state.Full) ||
		scheduleChanged(plan.Incremental, state.Incremental) ||
		scheduleChanged(plan.AccessControl, state.AccessControl)
}

func scheduleChanged(plan, state *tfSchedule) bool {
	if plan == nil && state == nil {
		return false
	}
	if plan == nil || state == nil {
		return true
	}
	return !plan.Enabled.Equal(state.Enabled) || !plan.Interval.Equal(state.Interval)
}

func pipelineChanged(plan, state *tfPipeline) bool {
	if plan == nil && state == nil {
		return false
	}
	if plan == nil || state == nil {
		return true
	}
	return !plan.Name.Equal(state.Name) ||
		!plan.ExtractBinaryContent.Equal(state.ExtractBinaryContent) ||
		!plan.ReduceWhitespace.Equal(state.ReduceWhitespace) ||
		!plan.RunMlInference.Equal(state.RunMlInference)
}
