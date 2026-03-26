variable "connector_name" {
  description = "The display name of the connector"
  type        = string
}

provider "elasticstack" {
  elasticsearch {}
}

resource "elasticstack_elasticsearch_security_api_key" "initial" {
  name = "${var.connector_name}-key"

  role_descriptors = jsonencode({
    connector-role = {
      cluster = ["monitor", "manage_connector"]
      indices = [{
        names      = ["test-dropbox", ".search-acl-filter-test-dropbox", ".elastic-connectors*"]
        privileges = ["all"]
      }]
    }
  })
}

resource "elasticstack_elasticsearch_security_api_key" "updated" {
  name = "${var.connector_name}-key-updated"

  role_descriptors = jsonencode({
    connector-role = {
      cluster = ["monitor", "manage_connector"]
      indices = [{
        names      = ["test-dropbox", ".search-acl-filter-test-dropbox", ".elastic-connectors*"]
        privileges = ["all"]
      }]
    }
  })
}

locals {
  initial_raw_key_id = element(split("/", elasticstack_elasticsearch_security_api_key.initial.key_id), length(split("/", elasticstack_elasticsearch_security_api_key.initial.key_id)) - 1)
}

resource "elasticstack_elasticsearch_search_connector" "test" {
  name         = var.connector_name
  service_type = "dropbox"
  index_name   = "test-dropbox"
  api_key_id   = local.initial_raw_key_id
}
