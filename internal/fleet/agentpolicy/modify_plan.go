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

package agentpolicy

import (
	"context"

	"github.com/elastic/terraform-provider-elasticstack/internal/utils/typeutils"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func (r *agentPolicyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	// Only enforce this constraint when Terraform is planning a new resource.
	// Existing resources can be updated to is_protected=true after Elastic Defend is attached.
	if !req.State.Raw.IsNull() {
		return
	}

	var planModel agentPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !typeutils.IsKnown(planModel.IsProtected) || !planModel.IsProtected.ValueBool() {
		return
	}

	resp.Diagnostics.AddAttributeError(
		path.Root("is_protected"),
		"Cannot enable tamper protection during creation",
		"Tamper protection can only be enabled when an Elastic Defend integration policy is already attached to the agent policy. "+
			"Create the policy with is_protected = false, attach Elastic Defend, then run another apply with is_protected = true.",
	)
}
