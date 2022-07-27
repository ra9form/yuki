module github.com/utrack/clay/integration

require (
	github.com/go-chi/chi v3.3.4+incompatible
	github.com/go-openapi/spec v0.0.0-20180415031709-bcff419492ee
	github.com/gogo/protobuf v1.3.2
	github.com/google/go-cmp v0.5.6
	github.com/googleapis/googleapis v0.0.0-20220316214218-db9d2a3c5e2f // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.5.0
	github.com/jmoiron/jsonq v0.0.0-20150511023944-e874b168d07e
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.5.1
	github.com/ra9form/yuki v3.0.0
	google.golang.org/genproto v0.0.0-20210617175327-b9e0b3197ced
	google.golang.org/grpc v1.38.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.1.0 // indirect
	google.golang.org/protobuf v1.27.1
	gopkg.in/yaml.v2 v2.2.3 // indirect
)

go 1.13

replace github.com/ra9form/yuki => ../
