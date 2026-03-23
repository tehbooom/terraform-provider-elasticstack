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

package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/elastic/terraform-provider-elasticstack/internal/clients"
	"github.com/elastic/terraform-provider-elasticstack/internal/diagutil"
	fwdiags "github.com/hashicorp/terraform-plugin-framework/diag"
)

// ConnectorResponse is the API response for GET /_connector/{id}
type ConnectorResponse struct {
	ID             string                     `json:"id"`
	Name           string                     `json:"name"`
	Description    string                     `json:"description"`
	IndexName      string                     `json:"index_name"`
	ServiceType    string                     `json:"service_type"`
	Status         string                     `json:"status"`
	IsNative       bool                       `json:"is_native"`
	APIKeyID       string                     `json:"api_key_id,omitempty"`
	APIKeySecretID string                     `json:"api_key_secret_id,omitempty"`
	Scheduling     *ConnectorScheduling       `json:"scheduling,omitempty"`
	Pipeline       *ConnectorPipeline         `json:"pipeline,omitempty"`
	Configuration  map[string]json.RawMessage `json:"configuration,omitempty"`
}

// ConnectorSchedule represents the schedule for a single sync type.
type ConnectorSchedule struct {
	Enabled  bool   `json:"enabled"`
	Interval string `json:"interval"`
}

// ConnectorScheduling holds full/incremental/access_control schedules.
type ConnectorScheduling struct {
	AccessControl *ConnectorSchedule `json:"access_control,omitempty"`
	Full          *ConnectorSchedule `json:"full,omitempty"`
	Incremental   *ConnectorSchedule `json:"incremental,omitempty"`
}

// ConnectorPipeline represents the ingest pipeline config.
type ConnectorPipeline struct {
	ExtractBinaryContent bool   `json:"extract_binary_content"`
	Name                 string `json:"name"`
	ReduceWhitespace     bool   `json:"reduce_whitespace"`
	RunMlInference       bool   `json:"run_ml_inference"`
}

// PutConnector creates or updates a connector. Returns the connector ID.
// indexName may be empty string to omit it from the request (connector without a target index).
func PutConnector(ctx context.Context, apiClient *clients.APIClient, connectorID, name, indexName, serviceType, description string) (string, fwdiags.Diagnostics) {
	var diags fwdiags.Diagnostics

	esClient, err := apiClient.GetESClient()
	if err != nil {
		diags.AddError("Failed to get Elasticsearch client", err.Error())
		return "", diags
	}

	body := map[string]any{
		"name":         name,
		"service_type": serviceType,
	}
	if indexName != "" {
		body["index_name"] = indexName
	}
	if description != "" {
		body["description"] = description
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		diags.AddError("Failed to marshal connector body", err.Error())
		return "", diags
	}

	res, err := esClient.ConnectorPut(
		esClient.ConnectorPut.WithContext(ctx),
		esClient.ConnectorPut.WithBody(bytes.NewReader(bodyBytes)),
		esClient.ConnectorPut.WithConnectorID(connectorID),
	)
	if err != nil {
		diags.AddError("Failed to create/update connector", err.Error())
		return "", diags
	}
	defer res.Body.Close()

	if diag := diagutil.CheckErrorFromFW(res, "Failed to create/update connector"); diag.HasError() {
		return "", diag
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		diags.AddError("Failed to decode connector response", err.Error())
		return "", diags
	}

	return result.ID, diags
}

// GetConnector fetches a connector by ID. Returns nil if not found.
func GetConnector(ctx context.Context, apiClient *clients.APIClient, connectorID string) (*ConnectorResponse, fwdiags.Diagnostics) {
	var diags fwdiags.Diagnostics

	esClient, err := apiClient.GetESClient()
	if err != nil {
		diags.AddError("Failed to get Elasticsearch client", err.Error())
		return nil, diags
	}

	res, err := esClient.ConnectorGet(connectorID,
		esClient.ConnectorGet.WithContext(ctx),
	)
	if err != nil {
		diags.AddError("Failed to get connector", err.Error())
		return nil, diags
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if diag := diagutil.CheckErrorFromFW(res, "Failed to get connector"); diag.HasError() {
		return nil, diag
	}

	var connector ConnectorResponse
	if err := json.NewDecoder(res.Body).Decode(&connector); err != nil {
		diags.AddError("Failed to decode connector", err.Error())
		return nil, diags
	}

	return &connector, diags
}

// DeleteConnector deletes a connector by ID.
func DeleteConnector(ctx context.Context, apiClient *clients.APIClient, connectorID string) fwdiags.Diagnostics {
	var diags fwdiags.Diagnostics

	esClient, err := apiClient.GetESClient()
	if err != nil {
		diags.AddError("Failed to get Elasticsearch client", err.Error())
		return diags
	}

	res, err := esClient.ConnectorDelete(connectorID,
		esClient.ConnectorDelete.WithContext(ctx),
	)
	if err != nil {
		diags.AddError("Failed to delete connector", err.Error())
		return diags
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return diags
	}

	return diagutil.CheckErrorFromFW(res, "Failed to delete connector")
}

// UpdateConnectorConfiguration updates the connector configuration via raw JSON.
func UpdateConnectorConfiguration(ctx context.Context, apiClient *clients.APIClient, connectorID string, configJSON string) fwdiags.Diagnostics {
	var diags fwdiags.Diagnostics

	esClient, err := apiClient.GetESClient()
	if err != nil {
		diags.AddError("Failed to get Elasticsearch client", err.Error())
		return diags
	}

	res, err := esClient.ConnectorUpdateConfiguration(bytes.NewReader([]byte(configJSON)), connectorID,
		esClient.ConnectorUpdateConfiguration.WithContext(ctx),
	)
	if err != nil {
		diags.AddError("Failed to update connector configuration", err.Error())
		return diags
	}
	defer res.Body.Close()

	return diagutil.CheckErrorFromFW(res, "Failed to update connector configuration")
}

// UpdateConnectorScheduling updates the connector scheduling.
func UpdateConnectorScheduling(ctx context.Context, apiClient *clients.APIClient, connectorID string, scheduling *ConnectorScheduling) fwdiags.Diagnostics {
	var diags fwdiags.Diagnostics

	esClient, err := apiClient.GetESClient()
	if err != nil {
		diags.AddError("Failed to get Elasticsearch client", err.Error())
		return diags
	}

	body := map[string]any{
		"scheduling": scheduling,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		diags.AddError("Failed to marshal scheduling body", err.Error())
		return diags
	}

	res, err := esClient.ConnectorUpdateScheduling(bytes.NewReader(bodyBytes), connectorID,
		esClient.ConnectorUpdateScheduling.WithContext(ctx),
	)
	if err != nil {
		diags.AddError("Failed to update connector scheduling", err.Error())
		return diags
	}
	defer res.Body.Close()

	return diagutil.CheckErrorFromFW(res, "Failed to update connector scheduling")
}

// UpdateConnectorPipeline updates the connector pipeline.
func UpdateConnectorPipeline(ctx context.Context, apiClient *clients.APIClient, connectorID string, pipeline *ConnectorPipeline) fwdiags.Diagnostics {
	var diags fwdiags.Diagnostics

	esClient, err := apiClient.GetESClient()
	if err != nil {
		diags.AddError("Failed to get Elasticsearch client", err.Error())
		return diags
	}

	bodyBytes, err := json.Marshal(pipeline)
	if err != nil {
		diags.AddError("Failed to marshal pipeline body", err.Error())
		return diags
	}

	res, err := esClient.ConnectorUpdatePipeline(bytes.NewReader(bodyBytes), connectorID,
		esClient.ConnectorUpdatePipeline.WithContext(ctx),
	)
	if err != nil {
		diags.AddError("Failed to update connector pipeline", err.Error())
		return diags
	}
	defer res.Body.Close()

	return diagutil.CheckErrorFromFW(res, "Failed to update connector pipeline")
}

// UpdateConnectorName updates the connector name and description.
func UpdateConnectorName(ctx context.Context, apiClient *clients.APIClient, connectorID, name, description string) fwdiags.Diagnostics {
	var diags fwdiags.Diagnostics

	esClient, err := apiClient.GetESClient()
	if err != nil {
		diags.AddError("Failed to get Elasticsearch client", err.Error())
		return diags
	}

	body := map[string]any{
		"name":        name,
		"description": description,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		diags.AddError("Failed to marshal name body", err.Error())
		return diags
	}

	res, err := esClient.ConnectorUpdateName(bytes.NewReader(bodyBytes), connectorID,
		esClient.ConnectorUpdateName.WithContext(ctx),
	)
	if err != nil {
		diags.AddError("Failed to update connector name", err.Error())
		return diags
	}
	defer res.Body.Close()

	return diagutil.CheckErrorFromFW(res, "Failed to update connector name")
}

// UpdateConnectorAPIKeyID associates an API key with the connector.
// apiKeySecretID is optional (only needed for native/Elastic-managed connectors).
func UpdateConnectorAPIKeyID(ctx context.Context, apiClient *clients.APIClient, connectorID, apiKeyID, apiKeySecretID string) fwdiags.Diagnostics {
	var diags fwdiags.Diagnostics

	esClient, err := apiClient.GetESClient()
	if err != nil {
		diags.AddError("Failed to get Elasticsearch client", err.Error())
		return diags
	}

	body := map[string]any{
		"api_key_id": apiKeyID,
	}
	if apiKeySecretID != "" {
		body["api_key_secret_id"] = apiKeySecretID
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		diags.AddError("Failed to marshal api_key_id body", err.Error())
		return diags
	}

	res, err := esClient.ConnectorUpdateAPIKeyDocumentID(bytes.NewReader(bodyBytes), connectorID,
		esClient.ConnectorUpdateAPIKeyDocumentID.WithContext(ctx),
	)
	if err != nil {
		diags.AddError("Failed to update connector API key ID", err.Error())
		return diags
	}
	defer res.Body.Close()

	return diagutil.CheckErrorFromFW(res, "Failed to update connector API key ID")
}

// UpdateConnectorIndexName updates the connector index name.
func UpdateConnectorIndexName(ctx context.Context, apiClient *clients.APIClient, connectorID, indexName string) fwdiags.Diagnostics {
	var diags fwdiags.Diagnostics

	esClient, err := apiClient.GetESClient()
	if err != nil {
		diags.AddError("Failed to get Elasticsearch client", err.Error())
		return diags
	}

	body := map[string]any{
		"index_name": indexName,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		diags.AddError("Failed to marshal index_name body", err.Error())
		return diags
	}

	res, err := esClient.ConnectorUpdateIndexName(bytes.NewReader(bodyBytes), connectorID,
		esClient.ConnectorUpdateIndexName.WithContext(ctx),
	)
	if err != nil {
		diags.AddError("Failed to update connector index name", err.Error())
		return diags
	}
	defer res.Body.Close()

	return diagutil.CheckErrorFromFW(res, "Failed to update connector index name")
}
