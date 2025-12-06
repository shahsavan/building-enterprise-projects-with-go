package avroschemas

import _ "embed"

// Assignment holds the embedded Avro schema for assignments.
//
//go:embed assignment.avsc
var Assignment []byte
