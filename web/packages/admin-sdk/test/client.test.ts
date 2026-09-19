import { describe, expect, it } from "vitest";

import { createAdminClient } from "../src/index.js";

/**
 * Smoke test that the namespace tree assembles into a live client. This mostly
 * exercises the build/spec wiring — a green run here means `op`/`client` bound
 * against the generated `paths` and the namespaces resolved end-to-end.
 */
describe("createAdminClient", () => {
  const client = createAdminClient();

  it("exposes the search namespace", () => {
    expect(client.search).toBeDefined();
    expect(typeof client.search.all).toBe("function");
  });
});
