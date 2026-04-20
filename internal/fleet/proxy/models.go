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

package proxy

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/terraform-provider-elasticstack/internal/clients"
	fleet "github.com/elastic/terraform-provider-elasticstack/internal/clients/fleet"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type proxyModel struct {
	ID                     types.String `tfsdk:"id"`
	KibanaConnection       types.List   `tfsdk:"kibana_connection"`
	ProxyID                types.String `tfsdk:"proxy_id"`
	SpaceID                types.String `tfsdk:"space_id"`
	Name                   types.String `tfsdk:"name"`
	URL                    types.String `tfsdk:"url"`
	Certificate            types.String `tfsdk:"certificate"`
	CertificateAuthorities types.String `tfsdk:"certificate_authorities"`
	CertificateKey         types.String `tfsdk:"certificate_key"`
	ProxyHeaders           types.Map    `tfsdk:"proxy_headers"`
	IsPreconfigured        types.Bool   `tfsdk:"is_preconfigured"`
}

func (model *proxyModel) populateFromAPI(spaceID string, item fleet.ProxyItem) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue((&clients.CompositeID{ClusterID: spaceID, ResourceID: item.ID}).String())
	model.ProxyID = types.StringValue(item.ID)
	model.SpaceID = types.StringValue(spaceID)
	model.Name = types.StringValue(item.Name)
	model.URL = types.StringValue(item.URL)

	if item.Certificate != nil && *item.Certificate != "" {
		model.Certificate = types.StringValue(*item.Certificate)
	} else {
		model.Certificate = types.StringNull()
	}

	if item.CertificateKey != nil && *item.CertificateKey != "" {
		model.CertificateKey = types.StringValue(*item.CertificateKey)
	} else {
		model.CertificateKey = types.StringNull()
	}

	if item.CertificateAuthorities != nil && *item.CertificateAuthorities != "" {
		model.CertificateAuthorities = types.StringValue(*item.CertificateAuthorities)
	} else {
		model.CertificateAuthorities = types.StringNull()
	}

	if item.IsPreconfigured != nil {
		model.IsPreconfigured = types.BoolValue(*item.IsPreconfigured)
	} else {
		model.IsPreconfigured = types.BoolValue(false)
	}

	if len(item.ProxyHeaders) > 0 {
		elems := make(map[string]attr.Value, len(item.ProxyHeaders))
		for k, raw := range item.ProxyHeaders {
			var s string
			if err := json.Unmarshal(raw, &s); err != nil {
				diags.AddWarning(
					"Non-string proxy header value",
					fmt.Sprintf("Proxy header %q has a non-string value %s and cannot be represented in state. "+
						"Only string header values are supported.", k, string(raw)),
				)
				continue
			}
			elems[k] = types.StringValue(s)
		}
		if !diags.HasError() {
			headersMap, mapDiags := types.MapValue(types.StringType, elems)
			diags.Append(mapDiags...)
			if !diags.HasError() {
				model.ProxyHeaders = headersMap
			}
		}
	} else {
		model.ProxyHeaders = types.MapNull(types.StringType)
	}

	return diags
}

// proxyHeadersFromModel converts the map[string]string Terraform model into
// map[string]json.RawMessage, encoding each string value as a JSON string.
// This is used as the wire representation for proxy_headers since the generated
// union wrapper types have an unexported field that cannot be populated via
// json.Unmarshal.
func proxyHeadersFromModel(m types.Map) (map[string]json.RawMessage, diag.Diagnostics) {
	var diags diag.Diagnostics

	if m.IsNull() || m.IsUnknown() || len(m.Elements()) == 0 {
		return nil, diags
	}

	result := make(map[string]json.RawMessage, len(m.Elements()))
	for k, v := range m.Elements() {
		s := v.(types.String).ValueString()
		b, err := json.Marshal(s)
		if err != nil {
			diags.AddError("Failed to encode proxy header", fmt.Sprintf("Could not encode proxy header %q: %s", k, err))
			return nil, diags
		}
		result[k] = b
	}

	return result, diags
}

// proxyCreateBody is a parallel struct for PostFleetProxies that uses
// map[string]json.RawMessage for proxy_headers, bypassing the generated union
// wrapper types whose unexported fields cannot be set via JSON round-tripping.
type proxyCreateBody struct {
	Certificate            *string                    `json:"certificate,omitempty"`
	CertificateAuthorities *string                    `json:"certificate_authorities,omitempty"`
	CertificateKey         *string                    `json:"certificate_key,omitempty"`
	ID                     *string                    `json:"id,omitempty"`
	IsPreconfigured        *bool                      `json:"is_preconfigured,omitempty"`
	Name                   string                     `json:"name"`
	ProxyHeaders           map[string]json.RawMessage `json:"proxy_headers,omitempty"`
	URL                    string                     `json:"url"`
}

// proxyUpdateBody is a parallel struct for PutFleetProxiesItemid.
// ProxyHeaders has no omitempty so that an empty map is sent as {} to clear headers.
type proxyUpdateBody struct {
	Certificate            *string                    `json:"certificate,omitempty"`
	CertificateAuthorities *string                    `json:"certificate_authorities,omitempty"`
	CertificateKey         *string                    `json:"certificate_key,omitempty"`
	Name                   *string                    `json:"name,omitempty"`
	ProxyHeaders           map[string]json.RawMessage `json:"proxy_headers"`
	URL                    *string                    `json:"url,omitempty"`
}

func (model proxyModel) toAPICreateModel() (proxyCreateBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	body := proxyCreateBody{
		Name: model.Name.ValueString(),
		URL:  model.URL.ValueString(),
	}

	if !model.ProxyID.IsNull() && !model.ProxyID.IsUnknown() {
		body.ID = model.ProxyID.ValueStringPointer()
	}

	if !model.Certificate.IsNull() && !model.Certificate.IsUnknown() {
		body.Certificate = model.Certificate.ValueStringPointer()
	}

	if !model.CertificateAuthorities.IsNull() && !model.CertificateAuthorities.IsUnknown() {
		body.CertificateAuthorities = model.CertificateAuthorities.ValueStringPointer()
	}

	if !model.CertificateKey.IsNull() && !model.CertificateKey.IsUnknown() {
		body.CertificateKey = model.CertificateKey.ValueStringPointer()
	}

	if !model.ProxyHeaders.IsNull() && !model.ProxyHeaders.IsUnknown() {
		headers, headerDiags := proxyHeadersFromModel(model.ProxyHeaders)
		diags.Append(headerDiags...)
		if diags.HasError() {
			return body, diags
		}
		body.ProxyHeaders = headers
	}

	return body, diags
}

func (model proxyModel) toAPIUpdateModel() (proxyUpdateBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	body := proxyUpdateBody{
		Name:         model.Name.ValueStringPointer(),
		URL:          model.URL.ValueStringPointer(),
		ProxyHeaders: map[string]json.RawMessage{},
	}

	if !model.Certificate.IsNull() && !model.Certificate.IsUnknown() {
		body.Certificate = model.Certificate.ValueStringPointer()
	}

	if !model.CertificateAuthorities.IsNull() && !model.CertificateAuthorities.IsUnknown() {
		body.CertificateAuthorities = model.CertificateAuthorities.ValueStringPointer()
	}

	if !model.CertificateKey.IsNull() && !model.CertificateKey.IsUnknown() {
		body.CertificateKey = model.CertificateKey.ValueStringPointer()
	}

	if !model.ProxyHeaders.IsNull() && !model.ProxyHeaders.IsUnknown() && len(model.ProxyHeaders.Elements()) > 0 {
		headers, headerDiags := proxyHeadersFromModel(model.ProxyHeaders)
		diags.Append(headerDiags...)
		if diags.HasError() {
			return body, diags
		}
		body.ProxyHeaders = headers
	}

	return body, diags
}
