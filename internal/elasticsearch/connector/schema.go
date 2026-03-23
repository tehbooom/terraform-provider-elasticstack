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

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"

	providerschema "github.com/elastic/terraform-provider-elasticstack/internal/schema"
)

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Elasticsearch search connector. See https://www.elastic.co/docs/api/doc/elasticsearch/group/endpoint-connector",
		Blocks: map[string]schema.Block{
			"elasticsearch_connection": providerschema.GetEsFWConnectionBlock(false),
		},
		Attributes: map[string]schema.Attribute{
			"connector_id": schema.StringAttribute{
				Description: "The unique identifier for the connector. If not set, Elasticsearch will generate one.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The display name of the connector.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A human-readable description of the connector.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"index_name": schema.StringAttribute{
				Description: "The name of the Elasticsearch index to sync data into. Can be omitted to create a connector without a target index.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service_type": schema.StringAttribute{
				Description: "The service type of the connector (e.g. 'google_drive', 'sharepoint_online').",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"configuration": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Description: "Connector configuration as a JSON object. The schema varies by service type. The full configuration object including metadata fields (display, label, type, etc.) should be provided.",
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
			},
			"scheduling": schema.SingleNestedAttribute{
				Description: "Sync scheduling configuration.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"full": schema.SingleNestedAttribute{
						Description: "Full sync schedule.",
						Optional:    true,
						Computed:    true,
						Attributes:  scheduleAttributes(),
					},
					"incremental": schema.SingleNestedAttribute{
						Description: "Incremental sync schedule.",
						Optional:    true,
						Computed:    true,
						Attributes:  scheduleAttributes(),
					},
					"access_control": schema.SingleNestedAttribute{
						Description: "Access control sync schedule.",
						Optional:    true,
						Computed:    true,
						Attributes:  scheduleAttributes(),
					},
				},
			},
			"pipeline": schema.SingleNestedAttribute{
				Description: "Ingest pipeline configuration.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Description: "The name of the ingest pipeline.",
						Optional:    true,
						Computed:    true,
					},
					"extract_binary_content": schema.BoolAttribute{
						Description: "Whether to extract binary content.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(true),
					},
					"reduce_whitespace": schema.BoolAttribute{
						Description: "Whether to reduce whitespace.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(true),
					},
					"run_ml_inference": schema.BoolAttribute{
						Description: "Whether to run ML inference.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(true),
					},
				},
			},
			"api_key_id": schema.StringAttribute{
				Description: "The ID of the API key used by the connector for authentication. For self-managed connectors this registers which key is associated with the connector. For native connectors (Elastic-managed) this is also used internally.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"api_key_secret_id": schema.StringAttribute{
				Description: "The secret storage ID of the API key. Required for native/Elastic-managed connectors running inside Elastic Cloud.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The current status of the connector.",
				Computed:    true,
			},
		},
	}
}

func scheduleAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"enabled": schema.BoolAttribute{
			Description: "Whether the schedule is enabled.",
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
		},
		"interval": schema.StringAttribute{
			Description: "The cron expression for the schedule interval.",
			Optional:    true,
			Computed:    true,
		},
	}
}
