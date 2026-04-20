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

package proxy_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/elastic/terraform-provider-elasticstack/internal/acctest"
	"github.com/elastic/terraform-provider-elasticstack/internal/clients"
	fleetclient "github.com/elastic/terraform-provider-elasticstack/internal/clients/fleet"
	"github.com/elastic/terraform-provider-elasticstack/internal/diagutil"
	"github.com/hashicorp/terraform-plugin-testing/config"
	sdkacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccResourceFleetProxy(t *testing.T) {
	proxyName := sdkacctest.RandString(22)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		CheckDestroy: checkResourceFleetProxyDestroy,
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("create"),
				ConfigVariables: config.Variables{
					"name": config.StringVariable(fmt.Sprintf("Proxy %s", proxyName)),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("elasticstack_fleet_proxy.test_proxy", "name", fmt.Sprintf("Proxy %s", proxyName)),
					resource.TestCheckResourceAttr("elasticstack_fleet_proxy.test_proxy", "url", "https://proxy.example.com:3128"),
					resource.TestCheckResourceAttrSet("elasticstack_fleet_proxy.test_proxy", "proxy_id"),
					resource.TestCheckResourceAttrSet("elasticstack_fleet_proxy.test_proxy", "id"),
					resource.TestCheckResourceAttr("elasticstack_fleet_proxy.test_proxy", "is_preconfigured", "false"),
					resource.TestCheckNoResourceAttr("elasticstack_fleet_proxy.test_proxy", "proxy_headers.%"),
				),
			},
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("update"),
				ConfigVariables: config.Variables{
					"name": config.StringVariable(fmt.Sprintf("Proxy Updated %s", proxyName)),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("elasticstack_fleet_proxy.test_proxy", "name", fmt.Sprintf("Proxy Updated %s", proxyName)),
					resource.TestCheckResourceAttr("elasticstack_fleet_proxy.test_proxy", "url", "https://proxy-updated.example.com:3128"),
					resource.TestCheckResourceAttr("elasticstack_fleet_proxy.test_proxy", "proxy_headers.X-Custom-Header", "my-value"),
					resource.TestCheckResourceAttr("elasticstack_fleet_proxy.test_proxy", "proxy_headers.X-Another", "another-value"),
				),
			},
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("clear_headers"),
				ConfigVariables: config.Variables{
					"name": config.StringVariable(fmt.Sprintf("Proxy Updated %s", proxyName)),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("elasticstack_fleet_proxy.test_proxy", "url", "https://proxy-updated.example.com:3128"),
					resource.TestCheckNoResourceAttr("elasticstack_fleet_proxy.test_proxy", "proxy_headers.%"),
				),
			},
			{
				ProtoV6ProviderFactories: acctest.Providers,
				ConfigDirectory:          acctest.NamedTestCaseDirectory("clear_headers"),
				ConfigVariables: config.Variables{
					"name": config.StringVariable(fmt.Sprintf("Proxy Updated %s", proxyName)),
				},
				ResourceName:      "elasticstack_fleet_proxy.test_proxy",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"kibana_connection",
				},
			},
		},
	})
}

func checkResourceFleetProxyDestroy(s *terraform.State) error {
	client, err := clients.NewAcceptanceTestingKibanaScopedClient()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "elasticstack_fleet_proxy" {
			continue
		}

		fc, err := client.GetFleetClient()
		if err != nil {
			return err
		}

		proxyID := rs.Primary.Attributes["proxy_id"]
		spaceID := rs.Primary.Attributes["space_id"]

		proxy, diags := fleetclient.GetProxy(context.Background(), fc, spaceID, proxyID)
		if diags.HasError() {
			return diagutil.FwDiagsAsError(diags)
		}
		if proxy != nil {
			return fmt.Errorf("fleet proxy id=%v still exists, but it should have been removed", proxyID)
		}
	}
	return nil
}
