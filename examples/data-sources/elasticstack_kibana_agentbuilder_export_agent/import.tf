# Import an agent and all its dependencies into another cluster.
#
# Prerequisites:
#   Run `terraform apply` in the export directory first. This config
#   reads the single "agent" output from that state to recreate the
#   full agent with its tools and workflows.
#
# Resource ordering:
#   1. Workflows are created first (no dependencies).
#      This covers both tool-embedded workflows and standalone workflows
#      referenced by the agent's workflow_ids.
#   2. Tools are created next. Workflow-type tools reference the new
#      workflow ID via interpolation, so Terraform resolves the
#      dependency automatically.
#   3. The agent is created last with an explicit depends_on for tools,
#      because the tool IDs come from the exported state (not a resource
#      reference Terraform can track).

provider "elasticstack" {
  kibana {}
}

variable "export_state_path" {
  description = "Path to the terraform.tfstate file produced by the export configuration."
  type        = string
  default     = "../export/terraform.tfstate"
}

data "terraform_remote_state" "export" {
  backend = "local"

  config = {
    path = var.export_state_path
  }
}

locals {
  exported = jsondecode(data.terraform_remote_state.export.outputs.agent)

  agent = jsondecode(local.exported.agent)

  # Readonly tools are built-in and already exist on the target cluster,
  # so we only create the writable ones.
  tools = [for t in local.exported.tools : t if !t.readonly]

  # Workflows come from two sources:
  #   - tool-embedded: extracted from workflow-type tools (yaml is on the tool)
  #   - standalone: exported separately via the agent's workflow_ids field
  # Merge both, deduplicating by ID.
  tool_workflows = [
    for t in local.tools : { id = t.workflow_id, yaml = t.workflow_configuration_yaml }
    if t.type == "workflow" && t.workflow_id != null
  ]
  standalone_workflows = local.exported.workflows

  all_workflows = {
    for w in concat(local.tool_workflows, local.standalone_workflows) : w.id => w...
  }
  workflows = [for id, versions in local.all_workflows : versions[0]]

  # Map each old workflow ID to its index so we can look up the new resource.
  old_workflow_id_to_index = {
    for i, w in local.workflows : w.id => i
  }

  # For each tool, pre-compute the configuration to use on the new cluster.
  # Workflow-type tools need their workflow_id swapped to the newly created one;
  # all other tool types keep their original configuration as-is.
  tool_configurations = [
    for t in local.tools : (
      t.type == "workflow"
      ? jsonencode({
        workflow_id = elasticstack_kibana_agentbuilder_workflow.workflows[
          local.old_workflow_id_to_index[t.workflow_id]
        ].workflow_id
      })
      : t.configuration
    )
  ]

  # Remap old standalone workflow IDs to new ones for the agent's workflow_ids.
  new_workflow_ids = try([
    for old_id in local.agent.configuration.workflow_ids :
    elasticstack_kibana_agentbuilder_workflow.workflows[
      local.old_workflow_id_to_index[old_id]
    ].workflow_id
  ], null)
}

# 1. Create workflows

resource "elasticstack_kibana_agentbuilder_workflow" "workflows" {
  count              = length(local.workflows)
  configuration_yaml = local.workflows[count.index].yaml
}

# 2. Create tools

resource "elasticstack_kibana_agentbuilder_tool" "tools" {
  count       = length(local.tools)
  tool_id     = local.tools[count.index].id
  type        = local.tools[count.index].type
  description = local.tools[count.index].description
  tags        = local.tools[count.index].tags

  configuration = local.tool_configurations[count.index]
}

# 3. Create agent

resource "elasticstack_kibana_agentbuilder_agent" "agent" {
  id            = local.agent.id
  name          = local.agent.name
  description   = try(local.agent.description, null)
  avatar_color  = try(local.agent.avatar_color, null)
  avatar_symbol = try(local.agent.avatar_symbol, null)
  labels        = try(local.agent.labels, null)
  instructions  = try(local.agent.configuration.instructions, null)

  enable_elastic_capabilities = try(local.agent.configuration.enable_elastic_capabilities, null)
  plugin_ids                  = try(local.agent.configuration.plugin_ids, null)
  skill_ids                   = try(local.agent.configuration.skill_ids, null)
  workflow_ids                = local.new_workflow_ids

  tools = try(
    flatten([for t in local.agent.configuration.tools : t.tool_ids]),
    null
  )

  depends_on = [elasticstack_kibana_agentbuilder_tool.tools]
}
