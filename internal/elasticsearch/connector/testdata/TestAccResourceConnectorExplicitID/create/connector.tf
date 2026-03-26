variable "connector_name" {
  description = "The display name of the connector"
  type        = string
}

variable "connector_id" {
  description = "The explicit connector ID to use"
  type        = string
}

provider "elasticstack" {
  elasticsearch {}
}

resource "elasticstack_elasticsearch_search_connector" "test" {
  connector_id = var.connector_id
  name         = var.connector_name
  service_type = "dropbox"
  index_name   = "test-dropbox"
}
