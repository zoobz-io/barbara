import { describe, expect, it } from "vitest";

import { createApiClient } from "../src/index.js";

/**
 * Smoke test that the namespace tree assembles into a live client. This mostly
 * exercises the build/spec wiring — a green run here means `op`/`client` bound
 * against the generated `paths` and the namespaces resolved end-to-end.
 */
describe("createApiClient", () => {
  const client = createApiClient();

  it("exposes the top-level resource namespaces", () => {
    expect(client.published).toBeDefined();
    expect(client.apps).toBeDefined();
    expect(client.collections).toBeDefined();
    expect(client.documents).toBeDefined();
    expect(client.versions).toBeDefined();
    expect(client.releases).toBeDefined();
    expect(client.assets).toBeDefined();
  });

  it("exposes namespace operations as callable methods", () => {
    expect(typeof client.apps.list).toBe("function");
    expect(typeof client.apps.get).toBe("function");
    expect(typeof client.published.search).toBe("function");
  });

  it("exposes the publishing lifecycle operations", () => {
    expect(typeof client.documents.publish).toBe("function");
    expect(typeof client.documents.unpublish).toBe("function");
    expect(typeof client.documents.rollback).toBe("function");
    expect(typeof client.releases.cut).toBe("function");
    expect(typeof client.releases.rollback).toBe("function");
  });

  it("exposes nested sub-namespaces", () => {
    expect(typeof client.documents.tags.add).toBe("function");
    expect(typeof client.documents.tags.remove).toBe("function");
  });
});
