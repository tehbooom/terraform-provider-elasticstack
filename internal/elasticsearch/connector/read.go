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
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *Resource) readFromAPI(ctx context.Context, client *clients.ElasticsearchScopedClient, model *tfModel) (bool, diag.Diagnostics) {
	connector, diags := esclient.GetConnector(ctx, client, model.ConnectorID.ValueString())
	if diags.HasError() {
		return false, diags
	}
	if connector == nil {
		return false, diags
	}

	diags.Append(model.populateFromAPI(ctx, connector)...)
	return true, diags
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state tfModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := r.Client().GetElasticsearchClient(ctx, state.ElasticsearchConnection)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfgPrior := state.Configuration
	schedPrior := state.Scheduling

	exists, diags := r.readFromAPI(ctx, client, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	// Keep stored configuration as last known Terraform state: GET may reorder, normalize, or
	// redact sensitive `value` fields so the body will not match the applied config string.
	// We merge the prior state with the remote state to restore redacted sensitive values
	// so that Terraform can detect drift on non-sensitive fields without perpetual diffs.
	state.Configuration = mergeConnectorConfiguration(cfgPrior, state.Configuration)

	// If prior scheduling was null, and the API returns all schedules as disabled, keep it null
	// to prevent a perpetual diff on omitted scheduling blocks.
	if schedPrior.IsNull() && !state.Scheduling.IsNull() {
		if isSchedulingDisabled(ctx, state.Scheduling) {
			state.Scheduling = types.ObjectNull(schedulingAttrTypes)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
