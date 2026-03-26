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
	if !plan.Scheduling.Equal(state.Scheduling) {
		if !plan.Scheduling.IsNull() && !plan.Scheduling.IsUnknown() {
			scheduling, diags := plan.toSchedulingAPI(ctx)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.Diagnostics.Append(esclient.UpdateConnectorScheduling(ctx, client, connectorID, scheduling)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	// Update pipeline if changed
	if !plan.Pipeline.Equal(state.Pipeline) {
		if !plan.Pipeline.IsNull() && !plan.Pipeline.IsUnknown() {
			pipeline, diags := plan.toPipelineAPI(ctx)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.Diagnostics.Append(esclient.UpdateConnectorPipeline(ctx, client, connectorID, pipeline)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	// Update API key association if changed
	if !plan.APIKeyID.Equal(state.APIKeyID) || !plan.APIKeySecretID.Equal(state.APIKeySecretID) {
		if !plan.APIKeyID.IsNull() && !plan.APIKeyID.IsUnknown() && plan.APIKeyID.ValueString() != "" {
			resp.Diagnostics.Append(esclient.UpdateConnectorAPIKeyID(ctx, client, connectorID,
				plan.APIKeyID.ValueString(), plan.APIKeySecretID.ValueString())...)
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

