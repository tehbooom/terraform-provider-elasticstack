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
  index_name   = "test-dropbox"

  pipeline = {
    name                   = "ent-search-generic-ingestion"
    extract_binary_content = false
    reduce_whitespace      = false
    run_ml_inference       = false
  }
}
