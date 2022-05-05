---
subcategory: "Ingest"
layout: ""
page_title: "Elasticstack: elasticstack_elasticsearch_ingest_processor_rename Data Source"
description: |-
  Helper data source to create a processor which renames an existing field.
---

# Data Source: elasticstack_elasticsearch_ingest_processor_rename

Renames an existing field. If the field doesn’t exist or the new name is already used, an exception will be thrown.

See: https://www.elastic.co/guide/en/elasticsearch/reference/current/rename-processor.html


## Example Usage

```terraform
provider "elasticstack" {
  elasticsearch {}
}

data "elasticstack_elasticsearch_ingest_processor_rename" "rename" {
  field        = "provider"
  target_field = "cloud.provider"
}

resource "elasticstack_elasticsearch_ingest_pipeline" "my_ingest_pipeline" {
  name = "rename-ingest"

  processors = [
    data.elasticstack_elasticsearch_ingest_processor_rename.rename.json
  ]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- **field** (String) The field to be renamed.
- **target_field** (String) The new name of the field.

### Optional

- **description** (String) Description of the processor.
- **if** (String) Conditionally execute the processor
- **ignore_failure** (Boolean) Ignore failures for the processor.
- **ignore_missing** (Boolean) If `true` and `field` does not exist or is `null`, the processor quietly exits without modifying the document.
- **on_failure** (List of String) Handle failures for the processor.
- **tag** (String) Identifier for the processor.

### Read-Only

- **id** (String) Internal identifier of the resource.
- **json** (String) JSON representation of this data source.