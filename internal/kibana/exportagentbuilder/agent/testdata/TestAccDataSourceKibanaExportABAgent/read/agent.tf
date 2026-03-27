variable "agent_id" {
  description = "The agent ID"
  type        = string
}

provider "elasticstack" {
  kibana {}
}

resource "elasticstack_kibana_agentbuilder_agent" "test" {
  agent_id     = var.agent_id
  name         = "Test Agent"
  description  = "A test agent for export"
  instructions = "You are a helpful assistant."
}

data "elasticstack_kibana_agentbuilder_export_agent" "test" {
  id                   = elasticstack_kibana_agentbuilder_agent.test.id
  include_dependencies = false
}
