provider "elasticstack" {
  elasticsearch {}
  kibana {}
}

resource "elasticstack_kibana_space" "tamper_space" {
  space_id    = var.space_id
  name        = "Tamper Protection ${var.space_id}"
  description = "Isolated space for tamper protection acceptance test"
}

resource "elasticstack_fleet_agent_policy" "test_policy" {
  name            = var.policy_name
  namespace       = "default"
  description     = "Test Agent Policy with tamper protection"
  monitor_logs    = true
  monitor_metrics = false
  skip_destroy    = false
  is_protected    = false
  space_ids       = [var.space_id]

  depends_on = [elasticstack_kibana_space.tamper_space]
}
