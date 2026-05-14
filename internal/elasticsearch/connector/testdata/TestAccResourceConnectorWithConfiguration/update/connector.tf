variable "connector_name" {
  description = "The display name of the connector"
  type        = string
}

provider "elasticstack" {
  elasticsearch {}
}

resource "elasticstack_elasticsearch_search_connector" "test" {
  name         = var.connector_name
  service_type = "google_cloud_storage"
  index_name   = "test-gcs-config"

  configuration = jsonencode({
    "buckets" : {
      "depends_on" : [],
      "display" : "textarea",
      "default_value" : null,
      "label" : "List of buckets",
      "sensitive" : false,
      "type" : "list",
      "required" : true,
      "options" : [],
      "validations" : [],
      "value" : "bucket1,bucket2,bucket3",
      "order" : 1,
      "ui_restrictions" : [],
      "tooltip" : null
    },
    "retry_count" : {
      "depends_on" : [],
      "display" : "numeric",
      "default_value" : 3,
      "label" : "Maximum retries for failed requests",
      "sensitive" : false,
      "type" : "int",
      "required" : false,
      "options" : [],
      "validations" : [],
      "value" : 10,
      "order" : 3,
      "ui_restrictions" : ["advanced"]
    }
  })
}
