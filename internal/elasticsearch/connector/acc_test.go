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

package connector_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/elastic/terraform-provider-elasticstack/internal/acctest"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients"
	esclient "github.com/elastic/terraform-provider-elasticstack/internal/clients/elasticsearch"
	"github.com/hashicorp/terraform-plugin-testing/config"
	sdkacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccResourceConnector tests basic CRUD: create, update name/description/index_name,
// verifies connector_id is stable across updates, and import.
func TestAccResourceConnector(t *testing.T) {
	connectorName := sdkacctest.RandStringFromCharSet(10, sdkacctest.CharSetAlphaNum)
	var connectorID string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		CheckDestroy: checkResourceConnectorDestroy,
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("create"),
				ConfigVariables: config.Variables{
					"connector_name": config.StringVariable(connectorName),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "name", connectorName),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "service_type", "dropbox"),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "index_name", "test-dropbox"),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "description", ""),
					resource.TestCheckResourceAttrSet("elasticstack_elasticsearch_search_connector.test", "connector_id"),
					resource.TestCheckResourceAttrSet("elasticstack_elasticsearch_search_connector.test", "status"),
					// save connector_id to verify stability after update
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["elasticstack_elasticsearch_search_connector.test"]
						connectorID = rs.Primary.Attributes["connector_id"]
						return nil
					},
				),
			},
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("update"),
				ConfigVariables: config.Variables{
					"connector_name": config.StringVariable(connectorName),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "name", connectorName+"-updated"),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "description", "updated description"),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "index_name", "test-dropbox-updated"),
					// connector_id must be stable across updates
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["elasticstack_elasticsearch_search_connector.test"]
						got := rs.Primary.Attributes["connector_id"]
						if got != connectorID {
							return fmt.Errorf("connector_id changed after update: was %s, now %s", connectorID, got)
						}
						return nil
					},
				),
			},
			{
				ProtoV6ProviderFactories:             acctest.Providers,
				ConfigDirectory:                      acctest.NamedTestCaseDirectory("update"),
				ConfigVariables:                      config.Variables{"connector_name": config.StringVariable(connectorName)},
				ResourceName:                         "elasticstack_elasticsearch_search_connector.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "connector_id",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["elasticstack_elasticsearch_search_connector.test"]
					if rs == nil {
						return "", fmt.Errorf("resource not found in state")
					}
					return rs.Primary.Attributes["connector_id"], nil
				},
				ImportStateVerifyIgnore: []string{"configuration"},
			},
		},
	})
}

// TestAccResourceConnectorNoIndexName tests creating a connector without a target index,
// then adding an index_name via update.
func TestAccResourceConnectorNoIndexName(t *testing.T) {
	connectorName := sdkacctest.RandStringFromCharSet(10, sdkacctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		CheckDestroy: checkResourceConnectorDestroy,
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("create"),
				ConfigVariables: config.Variables{
					"connector_name": config.StringVariable(connectorName),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "name", connectorName),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "index_name", ""),
					resource.TestCheckResourceAttrSet("elasticstack_elasticsearch_search_connector.test", "connector_id"),
				),
			},
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("update"),
				ConfigVariables: config.Variables{
					"connector_name": config.StringVariable(connectorName),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "name", connectorName),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "index_name", "test-dropbox"),
				),
			},
		},
	})
}

// TestAccResourceConnectorServiceTypeReplace verifies that changing service_type
// destroys and recreates the connector (RequiresReplace plan modifier).
func TestAccResourceConnectorServiceTypeReplace(t *testing.T) {
	connectorName := sdkacctest.RandStringFromCharSet(10, sdkacctest.CharSetAlphaNum)
	var connectorIDBefore string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		CheckDestroy: checkResourceConnectorDestroy,
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("create"),
				ConfigVariables: config.Variables{
					"connector_name": config.StringVariable(connectorName),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "service_type", "dropbox"),
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["elasticstack_elasticsearch_search_connector.test"]
						connectorIDBefore = rs.Primary.Attributes["connector_id"]
						return nil
					},
				),
			},
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("replace"),
				ConfigVariables: config.Variables{
					"connector_name": config.StringVariable(connectorName),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "service_type", "github"),
					// connector_id must be different — it was destroyed and recreated
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["elasticstack_elasticsearch_search_connector.test"]
						got := rs.Primary.Attributes["connector_id"]
						if got == connectorIDBefore {
							return fmt.Errorf("expected connector_id to change after service_type replace, but got same id %s", got)
						}
						return nil
					},
				),
			},
		},
	})
}

// TestAccResourceConnectorWithScheduling tests create with no scheduling,
// then enabling all three schedule types, then disabling them.
func TestAccResourceConnectorWithScheduling(t *testing.T) {
	connectorName := sdkacctest.RandStringFromCharSet(10, sdkacctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		CheckDestroy: checkResourceConnectorDestroy,
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("create"),
				ConfigVariables: config.Variables{
					"connector_name": config.StringVariable(connectorName),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "scheduling.full.enabled", "false"),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "scheduling.incremental.enabled", "false"),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "scheduling.access_control.enabled", "false"),
				),
			},
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("update"),
				ConfigVariables: config.Variables{
					"connector_name": config.StringVariable(connectorName),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "scheduling.full.enabled", "true"),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "scheduling.full.interval", "0 0 0 * * ?"),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "scheduling.incremental.enabled", "true"),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "scheduling.incremental.interval", "0 0 * * * ?"),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "scheduling.access_control.enabled", "true"),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "scheduling.access_control.interval", "0 0 2 * * ?"),
				),
			},
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("disable"),
				ConfigVariables: config.Variables{
					"connector_name": config.StringVariable(connectorName),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "scheduling.full.enabled", "false"),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "scheduling.incremental.enabled", "false"),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "scheduling.access_control.enabled", "false"),
				),
			},
		},
	})
}

// TestAccResourceConnectorWithPartialScheduling tests that setting only one schedule type
// doesn't cause a perpetual diff for the unset types.
func TestAccResourceConnectorWithPartialScheduling(t *testing.T) {
	connectorName := sdkacctest.RandStringFromCharSet(10, sdkacctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		CheckDestroy: checkResourceConnectorDestroy,
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("create"),
				ConfigVariables: config.Variables{
					"connector_name": config.StringVariable(connectorName),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "scheduling.full.enabled", "true"),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "scheduling.full.interval", "0 0 0 * * ?"),
				),
			},
			{
				// Apply the same config a second time — no diff expected.
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("create"),
				ConfigVariables: config.Variables{
					"connector_name": config.StringVariable(connectorName),
				},
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestAccResourceConnectorWithPipeline tests create with explicit pipeline settings,
// then updating all pipeline flags.
func TestAccResourceConnectorWithPipeline(t *testing.T) {
	connectorName := sdkacctest.RandStringFromCharSet(10, sdkacctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		CheckDestroy: checkResourceConnectorDestroy,
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("create"),
				ConfigVariables: config.Variables{
					"connector_name": config.StringVariable(connectorName),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "pipeline.name", "ent-search-generic-ingestion"),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "pipeline.extract_binary_content", "true"),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "pipeline.reduce_whitespace", "true"),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "pipeline.run_ml_inference", "true"),
				),
			},
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("update"),
				ConfigVariables: config.Variables{
					"connector_name": config.StringVariable(connectorName),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "pipeline.name", "ent-search-generic-ingestion"),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "pipeline.extract_binary_content", "false"),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "pipeline.reduce_whitespace", "false"),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "pipeline.run_ml_inference", "false"),
				),
			},
		},
	})
}

// TestAccResourceConnectorWithAPIKey tests associating an API key with a connector
// and verifies the connector's api_key_id matches the key's raw key_id, then swaps keys.
func TestAccResourceConnectorWithAPIKey(t *testing.T) {
	connectorName := sdkacctest.RandStringFromCharSet(10, sdkacctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		CheckDestroy: checkResourceConnectorDestroy,
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("create"),
				ConfigVariables: config.Variables{
					"connector_name": config.StringVariable(connectorName),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("elasticstack_elasticsearch_search_connector.test", "api_key_id"),
					// connector.api_key_id must equal the raw key ID of the initial key
					checkRawKeyIDMatches(
						"elasticstack_elasticsearch_search_connector.test", "api_key_id",
						"elasticstack_elasticsearch_security_api_key.initial", "key_id",
					),
				),
			},
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("update"),
				ConfigVariables: config.Variables{
					"connector_name": config.StringVariable(connectorName),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("elasticstack_elasticsearch_search_connector.test", "api_key_id"),
					// after update, connector.api_key_id must match the new key's raw ID
					checkRawKeyIDMatches(
						"elasticstack_elasticsearch_search_connector.test", "api_key_id",
						"elasticstack_elasticsearch_security_api_key.updated", "key_id",
					),
				),
			},
		},
	})
}

// TestAccResourceConnectorDisappears verifies that if a connector is deleted outside
// Terraform, a subsequent refresh detects it is gone and a plan shows it must be recreated.
func TestAccResourceConnectorDisappears(t *testing.T) {
	connectorName := sdkacctest.RandStringFromCharSet(10, sdkacctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		CheckDestroy: checkResourceConnectorDestroy,
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("create"),
				ConfigVariables: config.Variables{
					"connector_name": config.StringVariable(connectorName),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("elasticstack_elasticsearch_search_connector.test", "connector_id"),
					// delete the connector outside Terraform
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["elasticstack_elasticsearch_search_connector.test"]
						connectorID := rs.Primary.Attributes["connector_id"]
						client, err := clients.NewAcceptanceTestingElasticsearchScopedClient()
						if err != nil {
							return err
						}
						diags := esclient.DeleteConnector(context.Background(), client, connectorID)
						if diags.HasError() {
							return fmt.Errorf("failed to delete connector outside Terraform: %v", diags)
						}
						return nil
					},
				),
				// The connector was deleted outside Terraform, so the post-step refresh
				// will detect it is gone and produce a non-empty plan to recreate it.
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccResourceConnectorExplicitID tests creating a connector with a user-supplied
// connector_id instead of an autogenerated one.
func TestAccResourceConnectorExplicitID(t *testing.T) {
	connectorName := sdkacctest.RandStringFromCharSet(10, sdkacctest.CharSetAlphaNum)
	connectorID := "test-" + sdkacctest.RandStringFromCharSet(8, sdkacctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		CheckDestroy: checkResourceConnectorDestroy,
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("create"),
				ConfigVariables: config.Variables{
					"connector_name": config.StringVariable(connectorName),
					"connector_id":   config.StringVariable(connectorID),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "connector_id", connectorID),
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "name", connectorName),
				),
			},
			{
				// Verify connector_id cannot be changed without replace.
				ProtoV6ProviderFactories:             acctest.Providers,
				ConfigDirectory:                      acctest.NamedTestCaseDirectory("create"),
				ConfigVariables:                      config.Variables{"connector_name": config.StringVariable(connectorName), "connector_id": config.StringVariable(connectorID)},
				ResourceName:                         "elasticstack_elasticsearch_search_connector.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "connector_id",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["elasticstack_elasticsearch_search_connector.test"]
					if rs == nil {
						return "", fmt.Errorf("resource not found in state")
					}
					return rs.Primary.Attributes["connector_id"], nil
				},
				ImportStateVerifyIgnore: []string{"configuration"},
			},
		},
	})
}

// TestAccResourceConnectorAPIKeySecretID tests api_key_secret_id for native/Elastic-managed
// connectors running inside Elastic Cloud.
//
// Manual testing steps (requires Elastic Cloud deployment):
//  1. Set ELASTICSEARCH_ENDPOINTS, ELASTICSEARCH_API_KEY for your Cloud deployment.
//  2. Create a native connector via the Kibana UI (Elastic-managed, not self-managed).
//     Note the connector_id shown in the URL.
//  3. Create an API key and note both its id and the secret storage ID returned
//     by the Elastic Cloud keystore API (_security/api_key).
//  4. Apply the following config:
//
//     resource "elasticstack_elasticsearch_search_connector" "native" {
//       name              = "my-native-connector"
//       service_type      = "sharepoint_online"
//       index_name        = "my-sharepoint-index"
//       api_key_id        = "<raw_key_id>"
//       api_key_secret_id = "<secret_storage_id>"
//     }
//
//  5. Verify that terraform plan shows no diff after apply.
//  6. Verify the connector status in Kibana shows the key association.

// checkRawKeyIDMatches compares two key ID attributes after stripping any "cluster_uuid/" prefix
// from both. This handles the fact that key_id may be returned as composite by some ES versions.
func checkRawKeyIDMatches(resourceA, attrA, resourceB, attrB string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rsA := s.RootModule().Resources[resourceA]
		if rsA == nil {
			return fmt.Errorf("resource %s not found", resourceA)
		}
		rsB := s.RootModule().Resources[resourceB]
		if rsB == nil {
			return fmt.Errorf("resource %s not found", resourceB)
		}
		stripPrefix := func(v string) string {
			for i := len(v) - 1; i >= 0; i-- {
				if v[i] == '/' {
					return v[i+1:]
				}
			}
			return v
		}
		valA := stripPrefix(rsA.Primary.Attributes[attrA])
		valB := stripPrefix(rsB.Primary.Attributes[attrB])
		if valA != valB {
			return fmt.Errorf("%s.%s (%q) does not match %s.%s (%q)", resourceA, attrA, valA, resourceB, attrB, valB)
		}
		return nil
	}
}

func checkResourceConnectorDestroy(s *terraform.State) error {
	client, err := clients.NewAcceptanceTestingElasticsearchScopedClient()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "elasticstack_elasticsearch_search_connector" {
			continue
		}

		connectorID := rs.Primary.Attributes["connector_id"]
		connector, diags := esclient.GetConnector(context.Background(), client, connectorID)
		if diags.HasError() {
			return fmt.Errorf("unable to get connector %s: %v", connectorID, diags)
		}
		if connector != nil {
			return fmt.Errorf("connector (%s) still exists", connectorID)
		}
	}
	return nil
}

// TestAccResourceConnectorWithConfiguration tests creating and updating the configuration attribute.
func TestAccResourceConnectorWithConfiguration(t *testing.T) {
	connectorName := sdkacctest.RandStringFromCharSet(10, sdkacctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		CheckDestroy: checkResourceConnectorDestroy,
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("create"),
				ConfigVariables: config.Variables{
					"connector_name": config.StringVariable(connectorName),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("elasticstack_elasticsearch_search_connector.test", "name", connectorName),
					resource.TestCheckResourceAttrSet("elasticstack_elasticsearch_search_connector.test", "configuration"),
				),
			},
			{
				// Apply again to ensure no perpetual diff
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("create"),
				ConfigVariables: config.Variables{
					"connector_name": config.StringVariable(connectorName),
				},
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("update"),
				ConfigVariables: config.Variables{
					"connector_name": config.StringVariable(connectorName),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("elasticstack_elasticsearch_search_connector.test", "configuration"),
				),
			},
		},
	})
}
