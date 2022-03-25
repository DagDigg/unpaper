.PHONY: protoc
protoc:
	./protoc-gen.sh

.PHONY: start
start:
	./scripts/start.sh

.PHONY: pg
pg:
	./scripts/start-postgres-shell.sh
	
.PHONY: generate
generate:
	./scripts/gen-sql.sh

.PHONY: gen-certs
gen-certs:
	./certificate/gen.sh

.PHONY: build-extauth
build-extauth:
	./scripts/build-extauth.sh

.PHONY: build-backend
build-backend:
	./scripts/build-backend.sh