//go:build generate

package tools

import (
	_ "github.com/hashicorp/terraform-plugin-codegen-framework/cmd/tfplugingen-framework"
	_ "github.com/hashicorp/terraform-plugin-codegen-openapi/cmd/tfplugingen-openapi"
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)

// Generate the provider code specification from the OpenAPI spec.
//go:generate go run github.com/hashicorp/terraform-plugin-codegen-openapi/cmd/tfplugingen-openapi generate --config ../codegen/generator_config.yml --output ../codegen/provider-code-spec.json ../codegen/gigahost.openapi.yml

// Generate Terraform Plugin Framework data source schemas from the spec.
//go:generate go run github.com/hashicorp/terraform-plugin-codegen-framework/cmd/tfplugingen-framework generate data-sources --input ../codegen/provider-code-spec.json --output ../internal

// Generate Terraform Plugin Framework resource schemas from the spec.
//go:generate go run github.com/hashicorp/terraform-plugin-codegen-framework/cmd/tfplugingen-framework generate resources --input ../codegen/provider-code-spec.json --output ../internal

// Format the example Terraform configurations used in the docs.
//go:generate terraform fmt -recursive ../examples/

// Generate the provider documentation from the schemas and examples.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-dir .. -provider-name gigahost
