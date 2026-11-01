(function () {
  "use strict";

  function shellArgument(value) {
    const clean = String(value).trim();
    if (/^[A-Za-z0-9_./:@%+=,-]+$/.test(clean)) return clean;
    return `'${clean.replaceAll("'", "'\"'\"'")}'`;
  }

  // apiOnly reports a frontend that always ships the JSON API and never ships
  // blog or docs: Headless and SvelteKit.
  function apiOnly(framework) {
    return framework === "headless" || framework === "svelte";
  }

  function buildCommand(state) {
    const args = [
      "goilerplate",
      "new",
      "--name",
      shellArgument(state.name),
      "--module",
      shellArgument(state.module),
    ];
    const withoutHTML = apiOnly(state.framework);
    if (state.framework === "headless") {
      args.push("--headless");
    } else {
      args.push("--framework", state.framework);
      if (state.api && !withoutHTML) args.push("--api");
    }
    if (state.mcp && (withoutHTML || state.api)) args.push("--mcp");
    args.push("--database", state.database);
    args.push("--payment", state.payment);
    args.push("--mail", state.mail);
    if (state.workspaces) args.push("--workspaces");
    if (state.oauth.length) args.push("--oauth", state.oauth.join(","));
    if (state.storage) args.push("--storage");
    if (state.content.length && !withoutHTML) args.push("--content", state.content.join(","));
    args.push(shellArgument(state.destination));
    return args.join(" ");
  }

  const contract = { buildCommand };
  if (typeof module !== "undefined" && module.exports) module.exports = contract;
  if (typeof window !== "undefined") window.GoilerplateConfigurator = contract;
  if (typeof document === "undefined") return;

  const root = document.getElementById("configurator");
  if (!root) return;

  const form = document.getElementById("configurator-form");
  const command = document.getElementById("configurator-command");
  const copy = document.getElementById("configurator-copy");
  const copyStatus = document.getElementById("configurator-copy-status");

  function field(name) {
    return form.elements.namedItem(name);
  }

  function checked(name) {
    const element = field(name);
    return Boolean(element && element.checked);
  }

  function selectedValues(prefix, orderedValues) {
    return orderedValues.filter((value) => checked(`${prefix}-${value}`));
  }

  function state() {
    const framework = field("framework").value;
    return {
      name: field("name").value,
      module: field("module").value,
      destination: field("destination").value,
      framework,
      api: apiOnly(framework) || checked("api"),
      mcp: (apiOnly(framework) || checked("api")) && checked("mcp"),
      database: field("database").value,
      payment: field("payment").value,
      mail: field("mail").value,
      workspaces: checked("workspaces"),
      oauth: selectedValues("oauth", ["google", "github"]),
      storage: checked("storage"),
      content: selectedValues("content", ["blog", "docs"]),
    };
  }

  const preview = document.querySelector("[data-stack-preview]");
  const agentsPreview = document.querySelector("[data-agents-preview]");
  const labels = {
    htmx: "htmx 4.0",
    datastar: "Datastar 1.0",
    svelte: "SvelteKit",
    headless: "Headless",
    sqlite: "SQLite",
    postgres: "PostgreSQL",
    smtp: "SMTP",
    resend: "Resend",
    stripe: "Stripe",
    polar: "Polar",
    google: "Google OAuth",
    github: "GitHub OAuth",
    blog: "Blog",
    docs: "Docs",
  };

  function stackItems(currentState) {
    const items = [
      labels[currentState.framework],
      ...(currentState.api ? ["JSON API"] : []),
      ...(currentState.mcp ? ["MCP"] : []),
      labels[currentState.database],
      labels[currentState.mail],
      labels[currentState.payment],
      "Admin panel",
    ];
    if (currentState.workspaces) items.push("Workspaces");
    currentState.oauth.forEach((provider) => items.push(labels[provider]));
    if (currentState.storage) items.push("Storage");
    if (!apiOnly(currentState.framework)) currentState.content.forEach((module) => items.push(labels[module]));
    return items;
  }

  const agentsStack = {
    headless: "- Headless: no native frontend. The project serves the JSON API under `/api/v1` to your own frontend or apps.",
    svelte:
      "- SvelteKit: a static single-page app in `web/` that the Go binary embeds and serves on the same origin as the JSON API under `/api/v1`.",
  };

  function renderPreview(currentState) {
    if (preview) {
      preview.replaceChildren(
        ...stackItems(currentState).map((label, index) => {
          const tag = document.createElement("span");
          tag.className = "stack-tag rounded-full bg-secondary px-3 py-1 font-mono text-[12px]";
          tag.style.setProperty("--i", index);
          tag.textContent = label;
          return tag;
        }),
      );
    }
    if (agentsPreview) {
      agentsPreview.textContent =
        agentsStack[currentState.framework] || `- templ, ${labels[currentState.framework]}, vanilla JavaScript, and Tailwind CSS.`;
    }
  }

  const headlessNote = form.querySelector("[data-headless-note]");
  const svelteNote = form.querySelector("[data-svelte-note]");

  // setLocked disables one checkbox control and optionally forces its value.
  // Headless and SvelteKit always ship the JSON API and never ship blog or
  // docs. The MCP server needs the JSON API.
  function setLocked(name, locked, forced) {
    const input = field(name);
    if (!input) return;
    const root = input.previousElementSibling;
    if (locked && input.checked !== forced) {
      if (window.templ && window.templ.checkbox && root) {
        window.templ.checkbox.setChecked(root, forced);
      } else {
        input.checked = forced;
      }
    }
    input.disabled = locked;
    if (root) {
      root.toggleAttribute("data-disabled", locked);
      root.setAttribute("aria-disabled", String(locked));
      root.tabIndex = locked ? -1 : 0;
    }
    const label = input.closest("label");
    if (label) label.classList.toggle("opacity-50", locked);
  }

  function render() {
    const framework = field("framework").value;
    const locked = apiOnly(framework);
    setLocked("api", locked, true);
    setLocked("mcp", !locked && !checked("api"), false);
    setLocked("content-blog", locked, false);
    setLocked("content-docs", locked, false);
    if (headlessNote) headlessNote.hidden = framework !== "headless";
    if (svelteNote) svelteNote.hidden = framework !== "svelte";
    const currentState = state();
    command.textContent = buildCommand(currentState);
    renderPreview(currentState);
  }

  form.addEventListener("input", render);
  form.addEventListener("change", render);
  function attachCopy(copy, command, copyStatus, label, validate) {
    const copyIcon = copy.querySelector("[data-copy-icon]");
    const copyCheck = copy.querySelector("[data-copy-check]");
    let resetTimer;
    copy.addEventListener("click", async function () {
      if (validate && !form.reportValidity()) {
        copyStatus.textContent = "Fill in the three project fields first";
        return;
      }
      window.clearTimeout(resetTimer);
      try {
        if (navigator.clipboard && window.isSecureContext) {
          await navigator.clipboard.writeText(command.textContent);
        } else {
          const textarea = document.createElement("textarea");
          textarea.value = command.textContent;
          textarea.setAttribute("readonly", "");
          textarea.style.position = "fixed";
          textarea.style.opacity = "0";
          document.body.appendChild(textarea);
          textarea.select();
          const copied = document.execCommand("copy");
          textarea.remove();
          if (!copied) throw new Error("copy command failed");
        }
        copyStatus.textContent = "Command copied";
        copy.setAttribute("aria-label", "Command copied");
        copyIcon.classList.add("hidden");
        copyCheck.classList.remove("hidden");
        resetTimer = window.setTimeout(() => {
          copyStatus.textContent = label;
          copy.setAttribute("aria-label", label);
          copyIcon.classList.remove("hidden");
          copyCheck.classList.add("hidden");
        }, 2200);
      } catch (_) {
        copyStatus.textContent = "Copy failed. Select the command and copy it manually";
      }
    });
  }

  attachCopy(copy, command, copyStatus, "Copy command", true);
  attachCopy(
    document.getElementById("configurator-install-copy"),
    document.getElementById("configurator-install-command"),
    document.getElementById("configurator-install-copy-status"),
    "Copy installation commands",
    false,
  );

  render();
})();
