resource "elasticstack_elasticsearch_search_connector" "google_drive" {
  connector_id = "my_google_drive_connector"
  name         = "My Google Drive Connector"
  index_name   = "my_google_drive_index"
  service_type = "google_drive"
  description  = "Syncs content from Google Drive"

  # Full connector configuration as JSON (schema varies by service_type)
  configuration = jsonencode({
    service_account_credentials = {
      default_value   = null
      depends_on      = []
      display         = "textarea"
      label           = "Google Drive service account JSON"
      options         = []
      order           = 1
      required        = true
      sensitive       = true
      tooltip         = "This connector authenticates as a service account to synchronize content from Google Drive."
      type            = "str"
      ui_restrictions = []
      validations     = []
      value           = var.service_account_key
    }
    use_document_level_security = {
      default_value   = null
      depends_on      = []
      display         = "toggle"
      label           = "Enable document level security"
      options         = []
      order           = 5
      required        = true
      sensitive       = false
      tooltip         = "Enable document level security for Google Drive"
      type            = "bool"
      ui_restrictions = []
      validations     = []
      value           = false
    }
  })

  scheduling = {
    full = {
      enabled  = true
      interval = "0 0 * * * ?"
    }
    incremental = {
      enabled  = false
      interval = ""
    }
    access_control = {
      enabled  = false
      interval = ""
    }
  }

  pipeline = {
    name                   = "ent-search-generic-ingestion"
    extract_binary_content = true
    reduce_whitespace      = true
    run_ml_inference       = false
  }
}
