variable "agent_id" {
  description = "The agent ID"
  type        = string
}

provider "elasticstack" {
  kibana {}
}

resource "elasticstack_kibana_agentbuilder_agent" "test" {
  id                          = var.agent_id
  name                        = "Agent With New Fields"
  instructions                = "You are a capable agent."
  enable_elastic_capabilities = true
  plugin_ids                  = ["plugin-a"]
  skill_ids                   = ["skill-x", "skill-y"]
}
