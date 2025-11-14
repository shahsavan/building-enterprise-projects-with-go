//go:build generate

// This file is not built into the app. It is only used with `go generate ./api`.

package main

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=oapi-types.cfg.yaml openapi.yaml
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=oapi-server.cfg.yaml openapi.yaml

import (
	_ "github.com/oapi-codegen/oapi-codegen/v2/pkg/codegen"
	_ "gopkg.in/yaml.v2"
)
