module github.com/tozny/utils-go

require (
	github.com/Shopify/sarama v1.27.2
	github.com/aws/aws-sdk-go v1.34.0
	github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2 v2.10.0
	github.com/cloudevents/sdk-go/v2 v2.10.0
	github.com/go-pg/pg v8.0.3+incompatible
	github.com/go-redis/redis v6.15.2+incompatible
	github.com/google/uuid v1.1.1
	github.com/olivere/elastic v6.2.17+incompatible
	github.com/pascaldekloe/jwt v1.10.0
	github.com/robinjoseph08/go-pg-migrations v0.1.2
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20220314234659-1baeb1ce4c0b
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/eapache/go-resiliency v1.2.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20180814174437-776d5712da21 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/jcmturner/gofork v1.0.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jmespath/go-jmespath v0.3.0 // indirect
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/klauspost/compress v1.11.0 // indirect
	github.com/mailru/easyjson v0.0.0-20190626092158-b2ccc519800e // indirect
	github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
	github.com/modern-go/reflect2 v0.0.0-20180701023420-4b7aa43c6742 // indirect
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/pierrec/lz4 v2.5.2+incompatible // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b // indirect
	golang.org/x/sys v0.5.0 // indirect
	gopkg.in/jcmturner/aescts.v1 v1.0.1 // indirect
	gopkg.in/jcmturner/dnsutils.v1 v1.0.1 // indirect
	gopkg.in/jcmturner/gokrb5.v7 v7.5.0 // indirect
	gopkg.in/jcmturner/rpc.v1 v1.1.0 // indirect
	mellium.im/sasl v0.2.1 // indirect
)

replace golang.org/x/net => golang.org/x/net v0.7.0

replace golang.org/x/sys => golang.org/x/sys v0.0.0-20220412211240-33da011f77ad

replace mellium.im/sasl => mellium.im/sasl v0.3.1

go 1.18
