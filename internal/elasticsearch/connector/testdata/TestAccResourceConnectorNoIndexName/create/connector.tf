variable "connector_name" {
  description = "The display name of the connector"
  type        = string
}

provider "elasticstack" {
  elasticsearch {}
}

resource "elasticstack_elasticsearch_search_connector" "test" {
  name         = var.connector_name
  service_type = "dropbox"
}
