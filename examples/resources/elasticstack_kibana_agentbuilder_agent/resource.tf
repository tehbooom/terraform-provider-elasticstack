provider "elasticstack" {
  kibana {}
}

# Basic agent
resource "elasticstack_kibana_agentbuilder_agent" "my_agent" {
  id            = "my-agent"
  name          = "My Agent"
  description   = "An example agent built with Agent Builder."
  avatar_color  = "#BFDBFF"
  avatar_symbol = "MA"
  labels        = ["example", "demo"]
  instructions  = "You are a helpful assistant."

  tools = [
    elasticstack_kibana_agentbuilder_tool.my_tool.tool_id,
  ]
}

# Agent with all configuration options
resource "elasticstack_kibana_agentbuilder_workflow" "pre_exec" {
  configuration_yaml = <<-EOT
name: Pre-execution Workflow
description: Runs before every agent execution
enabled: true
triggers:
  - type: manual
inputs: []
steps:
  - name: setup
    type: console
    with:
      message: "Preparing agent context"
EOT
}

resource "elasticstack_kibana_agentbuilder_agent" "full_agent" {
  id           = "full-agent"
  name         = "Full-Featured Agent"
  description  = "Agent demonstrating all available configuration options."
  instructions = "You are a capable assistant with access to Elastic capabilities."

  enable_elastic_capabilities = true
  plugin_ids                  = ["plugin-a", "plugin-b"]
  skill_ids                   = ["skill-x"]
  workflow_ids                = [elasticstack_kibana_agentbuilder_workflow.pre_exec.workflow_id]
  labels                      = ["production"]
}

# Agent in a non-default space
resource "elasticstack_kibana_space" "my_space" {
  space_id = "my-space"
  name     = "My Space"
}

resource "elasticstack_kibana_agentbuilder_agent" "space_agent" {
  id       = "space-agent"
  space_id = elasticstack_kibana_space.my_space.space_id
  name     = "Space-Scoped Agent"
}
