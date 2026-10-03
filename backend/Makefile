.PHONY: openapi-docs

openapi-docs:
	go generate ./...
	npx --yes @redocly/cli@1.34.5 build-docs docs/swagger.yaml --output docs/redoc.html
