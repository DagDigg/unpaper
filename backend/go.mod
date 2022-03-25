module github.com/DagDigg/unpaper/backend

go 1.16

require (
	github.com/DagDigg/unpaper/core v0.0.0-00010101000000-000000000000
	github.com/Masterminds/squirrel v1.5.0
	github.com/cenkalti/backoff/v3 v3.2.2 // indirect
	github.com/containerd/containerd v1.4.4 // indirect
	github.com/containerd/continuity v0.0.0-20210315143101-93e15499afd5 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/dhui/dktest v0.3.4 // indirect
	github.com/docker/docker v20.10.5+incompatible // indirect
	github.com/go-redis/redis/v8 v8.10.0
	github.com/golang-migrate/migrate/v4 v4.14.1
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.2.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.3.0
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jackc/pgmock v0.0.0-20201204152224-4fe30f7445fd // indirect
	github.com/jackc/pgproto3/v2 v2.0.7 // indirect
	github.com/jackc/pgx/v4 v4.10.1
	github.com/lib/pq v1.10.2
	github.com/moby/term v0.0.0-20201216013528-df9cb8a40635 // indirect
	github.com/opencontainers/runc v1.0.0-rc93 // indirect
	github.com/ory/dockertest/v3 v3.6.3
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/stripe/stripe-go/v72 v72.37.0
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20210314154223-e6e6c4f2bb5b
	golang.org/x/lint v0.0.0-20201208152925-83fdc39ff7b5 // indirect
	golang.org/x/mod v0.4.2 // indirect
	golang.org/x/oauth2 v0.0.0-20210313182246-cd4f82c27b84 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20210315173758-2651cd453018
	google.golang.org/grpc v1.36.0
	google.golang.org/protobuf v1.26.0
	gotest.tools/v3 v3.0.3 // indirect
	honnef.co/go/tools v0.1.3 // indirect
	k8s.io/api v0.22.0
	k8s.io/apimachinery v0.22.0
	k8s.io/client-go v0.22.0
)

replace github.com/DagDigg/unpaper/core => ../core
