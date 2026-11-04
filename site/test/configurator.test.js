"use strict";

const assert = require("node:assert/strict");
const test = require("node:test");
const { buildCommand } = require("../assets/js/configurator.js");

function fullState() {
  return {
    name: "My App",
    module: "example.com/my-app",
    destination: "./my-app",
    framework: "datastar",
    api: false,
    mcp: false,
    database: "postgres",
    payment: "stripe",
    mail: "resend",
    workspaces: true,
    oauth: ["google", "github"],
    storage: true,
    content: ["blog", "docs"],
  };
}

function minimalState() {
  return {
    ...fullState(),
    framework: "htmx",
    database: "sqlite",
    mail: "smtp",
    workspaces: false,
    oauth: [],
    storage: false,
    content: [],
  };
}

test("builds the minimal command with providers and no modules", () => {
  assert.equal(
    buildCommand(minimalState()),
    "goilerplate new --name 'My App' --module example.com/my-app --framework htmx --database sqlite --payment stripe --mail smtp ./my-app",
  );
});

test("builds every flag in CLI parser order", () => {
  assert.equal(
    buildCommand(fullState()),
    "goilerplate new --name 'My App' --module example.com/my-app --framework datastar --database postgres --payment stripe --mail resend --workspaces --oauth google,github --storage --content blog,docs ./my-app",
  );
});

test("never names an edition", () => {
  assert.doesNotMatch(buildCommand(fullState()), /--edition/);
  assert.doesNotMatch(buildCommand(minimalState()), /--edition/);
});

test("quotes shell metacharacters as one argument", () => {
  const state = fullState();
  state.name = "Rocket $(touch nope) 'Lab'";
  assert.match(buildCommand(state), /--name 'Rocket \$\(touch nope\) '\"'\"'Lab'\"'\"'' --module/);
});

test("adds the JSON API to an htmx or Datastar command", () => {
  const minimal = minimalState();
  minimal.api = true;
  assert.equal(
    buildCommand(minimal),
    "goilerplate new --name 'My App' --module example.com/my-app --framework htmx --api --database sqlite --payment stripe --mail smtp ./my-app",
  );
  const full = fullState();
  full.api = true;
  assert.equal(
    buildCommand(full),
    "goilerplate new --name 'My App' --module example.com/my-app --framework datastar --api --database postgres --payment stripe --mail resend --workspaces --oauth google,github --storage --content blog,docs ./my-app",
  );
});

test("builds Headless commands without a framework or content", () => {
  const minimal = minimalState();
  minimal.framework = "headless";
  minimal.api = true;
  assert.equal(
    buildCommand(minimal),
    "goilerplate new --name 'My App' --module example.com/my-app --headless --database sqlite --payment stripe --mail smtp ./my-app",
  );
  const full = fullState();
  full.framework = "headless";
  assert.equal(
    buildCommand(full),
    "goilerplate new --name 'My App' --module example.com/my-app --headless --database postgres --payment stripe --mail resend --workspaces --oauth google,github --storage ./my-app",
  );
});

test("builds SvelteKit commands with the framework, without --api or content", () => {
  const minimal = minimalState();
  minimal.framework = "svelte";
  minimal.api = true;
  assert.equal(
    buildCommand(minimal),
    "goilerplate new --name 'My App' --module example.com/my-app --framework svelte --database sqlite --payment stripe --mail smtp ./my-app",
  );
  minimal.mcp = true;
  assert.equal(
    buildCommand(minimal),
    "goilerplate new --name 'My App' --module example.com/my-app --framework svelte --mcp --database sqlite --payment stripe --mail smtp ./my-app",
  );
  const full = fullState();
  full.framework = "svelte";
  full.mcp = true;
  assert.equal(
    buildCommand(full),
    "goilerplate new --name 'My App' --module example.com/my-app --framework svelte --mcp --database postgres --payment stripe --mail resend --workspaces --oauth google,github --storage ./my-app",
  );
});

test("adds the MCP server after the JSON API or Headless", () => {
  const minimal = minimalState();
  minimal.api = true;
  minimal.mcp = true;
  assert.equal(
    buildCommand(minimal),
    "goilerplate new --name 'My App' --module example.com/my-app --framework htmx --api --mcp --database sqlite --payment stripe --mail smtp ./my-app",
  );
  const full = fullState();
  full.framework = "headless";
  full.mcp = true;
  assert.equal(
    buildCommand(full),
    "goilerplate new --name 'My App' --module example.com/my-app --headless --mcp --database postgres --payment stripe --mail resend --workspaces --oauth google,github --storage ./my-app",
  );
});

test("drops the MCP server without the JSON API", () => {
  const state = fullState();
  state.mcp = true;
  assert.doesNotMatch(buildCommand(state), /--mcp/);
});
