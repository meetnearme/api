module github.com/meetnearme/api

go 1.24.0

toolchain go1.24.3

require (
	github.com/JohannesKaufmann/html-to-markdown v1.5.0
	github.com/PuerkitoBio/goquery v1.10.1
	github.com/a-h/templ v0.2.793
	github.com/aws/aws-cdk-go/awscdk/v2 v2.133.0
	github.com/aws/aws-lambda-go v1.46.0
	github.com/aws/aws-sdk-go-v2 v1.36.3
	github.com/aws/aws-sdk-go-v2/config v1.29.4
	github.com/aws/aws-sdk-go-v2/credentials v1.17.61
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.15.8
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression v1.7.43
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.35.3
	github.com/aws/aws-sdk-go-v2/service/rdsdata v1.23.4
	github.com/aws/constructs-go/constructs/v10 v10.3.0
	github.com/aws/jsii-runtime-go v1.96.0
	github.com/awslabs/aws-lambda-go-api-proxy v0.16.2
	github.com/bcampbell/fuzzytime v0.0.0-20191010161914-05ea0010feac
	github.com/cloudflare/cloudflare-go/v3 v3.1.0
	github.com/go-openapi/strfmt v0.23.0
	github.com/go-playground/validator v9.31.0+incompatible
	github.com/golang-jwt/jwt/v4 v4.5.2
	github.com/golang/geo v0.0.0-20250404181303-07d601f131f3
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/imroc/req v0.3.2
	github.com/itlightning/dateparse v0.2.1
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9
	github.com/lucasb-eyer/go-colorful v1.2.0
	github.com/nats-io/nats.go v1.43.0
	github.com/playwright-community/playwright-go v0.5101.0
	github.com/ringsaturn/tzf v1.0.0
	github.com/stripe/stripe-go/v83 v83.0.0
	github.com/weaviate/weaviate v1.28.8
	github.com/weaviate/weaviate-go-client/v4 v4.16.1
	github.com/zitadel/oidc/v3 v3.33.1
	github.com/zitadel/zitadel-go/v3 v3.0.1
	gorm.io/driver/postgres v1.5.11
	gorm.io/gorm v1.25.10
)

require (
	github.com/Masterminds/semver/v3 v3.2.1 // indirect
	github.com/andybalholm/cascadia v1.3.3 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodbstreams v1.23.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.9.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.25.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.29.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.16 // indirect
	github.com/aws/smithy-go v1.22.2 // indirect
	github.com/cdklabs/awscdk-asset-awscli-go/awscliv1/v2 v2.2.202 // indirect
	github.com/cdklabs/awscdk-asset-kubectl-go/kubectlv20/v2 v2.1.2 // indirect
	github.com/cdklabs/awscdk-asset-node-proxy-agent-go/nodeproxyagentv6/v2 v2.0.1 // indirect
	github.com/deckarep/golang-set/v2 v2.8.0 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-jose/go-jose/v3 v3.0.4 // indirect
	github.com/go-jose/go-jose/v4 v4.1.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/analysis v0.23.0 // indirect
	github.com/go-openapi/errors v0.22.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/loads v0.22.0 // indirect
	github.com/go-openapi/spec v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-openapi/validate v0.24.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/gorilla/securecookie v1.1.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.5 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/muhlemmer/gu v0.3.1 // indirect
	github.com/nats-io/nkeys v0.4.11 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/onsi/gomega v1.36.1 // indirect
	github.com/paulmach/orb v0.11.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/ringsaturn/tzf-rel-lite v0.0.2025-b // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/tidwall/geoindex v1.7.0 // indirect
	github.com/tidwall/geojson v1.4.5 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/rtree v1.10.0 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/twpayne/go-polyline v1.1.1 // indirect
	github.com/yuin/goldmark v1.7.0 // indirect
	github.com/zitadel/logging v0.6.1 // indirect
	github.com/zitadel/schema v1.3.0 // indirect
	go.mongodb.org/mongo-driver v1.17.4 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	golang.org/x/crypto v0.41.0 // indirect
	golang.org/x/exp v0.0.0-20241217172543-b2144cdd0a67 // indirect
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/mod v0.26.0 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	golang.org/x/tools v0.35.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250825161204-c5933d9347a5 // indirect
	google.golang.org/grpc v1.75.0 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
