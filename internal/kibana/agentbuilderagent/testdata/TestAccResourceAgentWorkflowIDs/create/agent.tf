variable "agent_id" {
  description = "The agent ID"
  type        = string
}

provider "elasticstack" {
  kibana {}
}

resource "elasticstack_kibana_agentbuilder_workflow" "pre" {
  configuration_yaml = <<-EOT
name: Pre-execution Workflow
description: Runs before agent execution
enabled: true
triggers:
  - type: manual
inputs: []
steps:
  - name: noop
    type: console
    with:
      message: "pre-execution"
EOT
}

resource "elasticstack_kibana_agentbuilder_agent" "test" {
  id           = var.agent_id
  name         = "Agent With Workflow IDs"
  instructions = "You are an agent with pre-execution workflows."
  workflow_ids = [elasticstack_kibana_agentbuilder_workflow.pre.workflow_id]
}
