variable "connector_name" {
  description = "The display name of the connector"
  type        = string
}

provider "elasticstack" {
  elasticsearch {}
}

resource "elasticstack_elasticsearch_search_connector" "test" {
  name         = "${var.connector_name}-updated"
  description  = "updated description"
  service_type = "dropbox"
  index_name   = "test-dropbox-updated"
}
