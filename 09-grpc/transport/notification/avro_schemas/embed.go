package avroschemas

import _ "embed"

// Notification holds the embedded Avro schema for notification.
//
//go:embed notification.avsc
var Notification []byte
