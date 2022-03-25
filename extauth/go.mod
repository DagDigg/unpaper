module github.com/DagDigg/unpaper/extauth

go 1.16

require (
	github.com/DagDigg/unpaper/core v0.0.0-00010101000000-000000000000
	github.com/cncf/udpa/go v0.0.0-20210210032658-bff43e8824d0 // indirect
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/envoyproxy/go-control-plane v0.9.9-0.20201210154907-fd9021fe5dad
	github.com/envoyproxy/protoc-gen-validate v0.5.0 // indirect
	github.com/go-redis/redis/v8 v8.10.0
	github.com/gogo/googleapis v1.4.0
	github.com/golang-migrate/migrate/v4 v4.14.1
	github.com/jackc/pgx/v4 v4.10.1
	github.com/ory/dockertest/v3 v3.6.3
	github.com/pquerna/cachecontrol v0.0.0-20201205024021-ac21108117ac // indirect
	golang.org/x/crypto v0.0.0-20210314154223-e6e6c4f2bb5b // indirect
	golang.org/x/oauth2 v0.0.0-20210313182246-cd4f82c27b84
	google.golang.org/genproto v0.0.0-20210315173758-2651cd453018
	google.golang.org/grpc v1.36.0
	gopkg.in/square/go-jose.v2 v2.5.1 // indirect
)

replace github.com/DagDigg/unpaper/core => ../core
