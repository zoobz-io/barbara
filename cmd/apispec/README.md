# cmd/apispec

Dumps the public API's OpenAPI spec to disk. No server, no database — this is a
CLI that feeds the [`@barbara/api-sdk`](../../web/packages/api-sdk/) client
generator. The heavy lifting lives in
[`internal/openapispec`](../../internal/openapispec/); this main just picks the
surface (`api/handlers`) and the output path.

## Output path

Defaults to `web/packages/api-sdk/data/openapi.json`; override with the first
argument. The Makefile passes the SDK's data snapshot:

```bash
make openapi-api   # go run ./cmd/apispec web/packages/api-sdk/data/openapi.json
```

Change a public endpoint's shape and re-run this, then `pnpm generate` in the
SDK package (see the [web README](../../web/README.md)).
