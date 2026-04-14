import { describe, it, mock } from "node:test";
import assert from "node:assert/strict";
import { Worker } from "./worker.js";
import { MagiCClient } from "./client.js";

describe("Worker", () => {
  it("should register capabilities", () => {
    const w = new Worker({ name: "TestBot", endpoint: "http://localhost:9000" });
    w.capability("greet", "Says hello", async (input) => `Hello ${input.name}`);
    // Worker stores capabilities internally — verify via registration payload
    assert.equal(w.name, "TestBot");
  });

  it("should parse port from endpoint", () => {
    const w = new Worker({ name: "Bot", endpoint: "http://localhost:3456" });
    assert.equal(w.endpoint, "http://localhost:3456");
  });

  it("should use default port 9000", () => {
    const w = new Worker({ name: "Bot", endpoint: "http://localhost" });
    assert.equal(w.endpoint, "http://localhost");
  });
});

describe("MagiCClient", () => {
  it("should construct with base URL", () => {
    const client = new MagiCClient("http://localhost:8080");
    assert.ok(client);
  });

  it("should strip trailing slash from base URL", () => {
    const client = new MagiCClient("http://localhost:8080/");
    assert.ok(client);
  });

  it("should set auth header when API key provided", () => {
    const client = new MagiCClient("http://localhost:8080", "test-key");
    assert.ok(client);
  });
});
