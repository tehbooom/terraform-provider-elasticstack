package synthetics

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func (r *Resource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {

	kibanaClient := GetKibanaClient(r, response.Diagnostics)
	if kibanaClient == nil {
		return
	}

	plan := new(tfModelV0)
	diags := request.Plan.Get(ctx, plan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	input, diags := plan.toKibanaAPIRequest()
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	namespace := plan.SpaceID.ValueString()
	result, err := kibanaClient.KibanaSynthetics.Monitor.Add(ctx, input.config, input.fields, namespace)
	if err != nil {
		response.Diagnostics.AddError(fmt.Sprintf("Failed to create Kibana monitor `%s`, namespace %s", input.config.Name, namespace), err.Error())
		return
	}

	plan, err = plan.toModelV0(result)
	if err != nil {
		response.Diagnostics.AddError("Failed to convert Kibana monitor API to TF state", err.Error())
		return
	}

	diags = response.State.Set(ctx, plan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}