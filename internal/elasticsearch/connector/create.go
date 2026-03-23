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
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tfModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, diags := clients.MaybeNewAPIClientFromFrameworkResource(ctx, plan.ElasticsearchConnection, r.client)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	connectorID, diags := esclient.PutConnector(
		ctx, client,
		plan.ConnectorID.ValueString(),
		plan.Name.ValueString(),
		plan.IndexName.ValueString(),
		plan.ServiceType.ValueString(),
		plan.Description.ValueString(),
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ConnectorID = types.StringValue(connectorID)

	// Apply configuration if provided
	if !plan.Configuration.IsNull() && !plan.Configuration.IsUnknown() {
		resp.Diagnostics.Append(esclient.UpdateConnectorConfiguration(ctx, client, connectorID, plan.Configuration.ValueString())...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Apply scheduling if provided
	if plan.Scheduling != nil {
		resp.Diagnostics.Append(esclient.UpdateConnectorScheduling(ctx, client, connectorID, plan.toSchedulingAPI())...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Apply pipeline if provided
	if plan.Pipeline != nil {
		resp.Diagnostics.Append(esclient.UpdateConnectorPipeline(ctx, client, connectorID, plan.toPipelineAPI())...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Associate API key if provided
	if !plan.APIKeyID.IsNull() && !plan.APIKeyID.IsUnknown() && plan.APIKeyID.ValueString() != "" {
		resp.Diagnostics.Append(esclient.UpdateConnectorAPIKeyID(ctx, client, connectorID,
			plan.APIKeyID.ValueString(), plan.APIKeySecretID.ValueString())...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Read back to populate computed fields
	exists, diags := r.readFromAPI(ctx, client, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !exists {
		resp.Diagnostics.AddError("Connector not found after creation", "The connector was created but could not be read back")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}
