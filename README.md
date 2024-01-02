# Wishlist - An API Server in Go, Fully Containerized and Kubernetes-ready
Wishlist is an API Server to provide users a wishlist functionality over JSON API, users can define wishlists and add products to them from different sources and online-shops.

Right now wishlist has email verification, JWT auth and the user management system. Business features will be added but the core of implementation decisions are already made and visible in the source code.

## Features
- Custom web framework that utilizes the standatd http package, defining type for handler and middlewares
- Multi-stage dockerfile to build differnet images for admin, live-reload and production binaries
- Kubernetes dev environment that is ready to start by Makefile commands
- Custom test utilities to bring up real containers in tests like database, using docker compose format for multi-container support
- Polymorphic components as storage which are interchangable and testable

## Structure
![UML Diagram](https://i.ibb.co/km4pQwW/gleek-q36nzkohg8-Ip7-C8drd3ss-Q.png)

```bash
├── Makefile
├── business # hold the business-specific logic and packages
│     ├── auth  # Authentication and JWT
│     │   ├── auth.go
│     │   ├── auth_test.go
│     │   └── claims.go # Different claims are defined here
│     ├── database
│     │     ├── db
│     │     │     └── db.go # A helper package to connect and query postgres db
│     │     └── migration # Migration package to migrate and seed database
│     │         ├── migration.go
│     │         ├── seed.sql
│     │         └── sql # MIgration scripts in SQL
│     ├── email # The mail client is defined here and initially implemented by an external service called Courier
│     │     └── email.go
│     ├── entities # All application core entities are kept here, packages in this layer do not depend on any other package
│     │   └── user # The user entity: Defines a Storage interface that can store user data, provides a BookKepper object that uses Storage to persist data
│     │       ├── model.go
│     │       └── user.go
│     ├── keystore # Keystore is an in-memory keystore to rotate keys used by auth to sign and validate JWT token
│     │     ├── keystore.go
│     │     └── keystore_test.go
│     ├── otp # otp provides one-time-password
│     │     ├── otp.go
│     │     └── otp_test.go
│     ├── storage # Storage is a layer that holds packages that store data, can be a cache, a persistant keyvalue store or relational database manipulation
│     │     ├── keyvalue # keyvalue defines the interface of a keyvalue store
│     │     │     ├── keyvalue.go
│     │     │     └── kvstores # kvstores are holds different implementations of keyvalue store
│     │     │         └── freecache.go # freecache is an implementation of keyvalue store using freecache package
│     │     └── postgres # postgres holds the implementations of entities' storage using postgres via db package
│     │         ├── userdb
│     │         │     ├── model.go
│     │         │     └── userdb.go
│     ├── validate # validate uses go-playground/validator to provide a validator that is used to validate http requests
│     │     ├── custom.go
│     │     ├── custom_test.go
│     │     ├── errors.go
│     │     ├── validate.go
│     │     └── validate_test.go
│     └── web # business.web has business web manipulations like middlewares
│         └── middlewares # middlewares are registered to requests for purposes like: auth, logging and error handling
│             ├── auth.go
│             ├── errors.go
│             └── log.go
├── cmd # entrypoint of binary builds
│     ├── admin # admin is the tool for administration stuff like migrating database before app start
│     │     └── main.go
│     ├── wishapi # the API entrypoint
│     │     ├── main.go
│     │     └── v1 # v1 has the handlers of api v1
│     │         └── handlers
│     │             ├── probes # kubernetes liveness and readiness probes
│     │             │     └── probes.go
│     │             ├── usergrp # usergrp is the handler group for user authentication
│     │             │     ├── model.go
│     │             │     ├── usergrp.go
│     │             │     └── usergrp_test.go
│     └── zapformat # zapformat is used for generating a human readable log stream from app which uses zap for structured logging
│         └── main.go
├── foundation # foundation has the packages used by business packages, they dont depend on any package themselves
│     ├── apitest # used for testing a handler group, provides a http server and test database
│     │     ├── apitest.go
│     │     ├── database.go
│     │     └── server.go
│     ├── compose # compose is  a testing-utility to bring up containers from a docker compose file and provides the url and cleanup functions
│     │     ├── compose.go
│     │     ├── compose_test.go
│     │     └── container.go
│     └── web # web is a custom web framework that defines it's own handler and middleware types and use them to bring up a web app server
│         ├── context.go
│         ├── error.go
│         ├── web.go
│         └── web_test.go
├── infra
│     ├── docker
│     │     └── Dockerfile # different binaries build stages and production container image stages
│     └── k8s # k8s has the kubernetes yaml files
│         ├── dev # dev env files
│         │     ├── kind-config.yaml # kind dev env config
│         │     ├── postgres # postgres kubernetes deployment for dev
│         │     ├── wishapi # application dev-customized version
│         │     ├── wishapi-live # application dev with live reload
│         │     └── zipkin # zipkin kubernetes deployment for dev
│         └── wishapi # application k8s base files - can be customized to be used for production or dev
```