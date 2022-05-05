---
subcategory: "Ingest"
layout: ""
page_title: "Elasticstack: elasticstack_elasticsearch_ingest_processor_date_index_name Data Source"
description: |-
  Helper data source to create a processor which helps to point documents to the right time based index based on a date or timestamp field in a document by using the date math index name support.
---

# Data Source: elasticstack_elasticsearch_ingest_processor_date_index_name

The purpose of this processor is to point documents to the right time based index based on a date or timestamp field in a document by using the date math index name support.

The processor sets the _index metadata field with a date math index name expression based on the provided index name prefix, a date or timestamp field in the documents being processed and the provided date rounding.

First, this processor fetches the date or timestamp from a field in the document being processed. Optionally, date formatting can be configured on how the field’s value should be parsed into a date. Then this date, the provided index name prefix and the provided date rounding get formatted into a date math index name expression. Also here optionally date formatting can be specified on how the date should be formatted into a date math index name expression.

See: https://www.elastic.co/guide/en/elasticsearch/reference/current/date-index-name-processor.html

## Example Usage

```terraform
provider "elasticstack" {
  elasticsearch {}
}

data "elasticstack_elasticsearch_ingest_processor_date_index_name" "date_index_name" {
  description       = "monthly date-time index naming"
  field             = "date1"
  index_name_prefix = "my-index-"
  date_rounding     = "M"
}

resource "elasticstack_elasticsearch_ingest_pipeline" "my_ingest_pipeline" {
  name = "date-index-name-ingest"

  processors = [
    data.elasticstack_elasticsearch_ingest_processor_date_index_name.date_index_name.json
  ]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- **date_rounding** (String) How to round the date when formatting the date into the index name.
- **field** (String) The field to get the date or timestamp from.

### Optional

- **date_formats** (List of String) An array of the expected date formats for parsing dates / timestamps in the document being preprocessed.
- **description** (String) Description of the processor.
- **if** (String) Conditionally execute the processor
- **ignore_failure** (Boolean) Ignore failures for the processor.
- **index_name_format** (String) The format to be used when printing the parsed date into the index name.
- **index_name_prefix** (String) A prefix of the index name to be prepended before the printed date.
- **locale** (String) The locale to use when parsing the date from the document being preprocessed, relevant when parsing month names or week days.
- **on_failure** (List of String) Handle failures for the processor.
- **tag** (String) Identifier for the processor.
- **timezone** (String) The timezone to use when parsing the date and when date math index supports resolves expressions into concrete index names.

### Read-Only

- **id** (String) Internal identifier of the resource
- **json** (String) JSON representation of this data source.