module platform-mcp

go 1.24.0

require (
	github.com/gin-gonic/gin v1.11.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/prometheus/client_golang v1.20.5
	github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth v0.0.0-00010101000000-000000000000
	go.uber.org/zap v1.26.0
	google.golang.org/grpc v1.68.0
	google.golang.org/protobuf v1.36.9
)

replace github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth => ../../libs/platform-auth
