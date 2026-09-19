# cmd/adminspec

Dumps the admin API's OpenAPI spec to disk. No server, no database — this is a
CLI that feeds the [`@barbara/admin-sdk`](../../web/packages/admin-sdk/) client
generator. The heavy lifting lives in
[`internal/openapispec`](../../internal/openapispec/); this main just picks the
surface (`admin/handlers`) and the output path.

## Output path

Defaults to `web/packages/admin-sdk/data/openapi.json`; override with the first
argument. The Makefile passes the SDK's data snapshot:

```bash
make openapi-admin   # go run ./cmd/adminspec web/packages/admin-sdk/data/openapi.json
```

Change an admin endpoint's shape and re-run this, then `pnpm generate` in the
SDK package (see the [web README](../../web/README.md)).
