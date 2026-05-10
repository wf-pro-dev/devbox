<script lang="ts">
  import { onMount } from "svelte";
  import { Toaster } from "svelte-sonner";
  import { api } from "./api";
  import FinderView from "./components/finder/FinderView.svelte";
  import type { HealthResponse } from "./types";

  let health: HealthResponse | null = null;

  async function loadHealth() {
    try {
      health = await api.health();
    } catch {
      health = null;
    }
  }

  onMount(() => {
    loadHealth();
  });
</script>

<div class="app-shell">
  <Toaster position="top-center" />
  <FinderView {health} />
</div>

<style>
  @import url("https://fonts.googleapis.com/css2?family=Instrument+Serif:ital@0;1&family=DM+Mono:wght@300;400;500&family=DM+Sans:wght@300;400;500;600&display=swap");
  @import url("https://cdn.jsdelivr.net/npm/@tabler/icons-webfont@latest/tabler-icons.min.css");

  :global(*, *::before, *::after) {
    box-sizing: border-box;
    margin: 0;
    padding: 0;
  }

  :global(:root) {
    --bg: #fafaf9;
    --bg-2: #f5f4f0;
    --bg-3: #eeede8;
    --border: #e2e0d8;
    --border-2: #d0cec4;
    --text: #1c1b18;
    --text-2: #6b6860;
    --text-3: #9b9890;
    --radius: 6px;
    --radius-lg: 10px;
    --mono: "DM Mono", "Fira Mono", monospace;
    --serif: "Instrument Serif", Georgia, serif;
    --sans: "DM Sans", system-ui, sans-serif;

    --f-bg0: #f7f6f3;
    --f-bg1: #efede8;
    --f-bg2: #e5e3dc;
    --f-surface: #fafaf8;
    --f-surface2: #f3f1ec;
    --f-border: rgba(0, 0, 0, 0.09);
    --f-border2: rgba(0, 0, 0, 0.15);
    --f-text: #1a1916;
    --f-text2: #5c5a54;
    --f-text3: #9a9790;
    --f-accent: #2b5ce6;
    --f-accent-bg: rgba(43, 92, 230, 0.09);
    --f-accent-border: rgba(43, 92, 230, 0.22);
    --f-selection: rgba(43, 92, 230, 0.1);
    --f-folder: #e8922a;
    --f-search-bg: #fffefa;
    --f-search-border: rgba(0, 0, 0, 0.14);
    --f-ok: #1a7a44;
    --f-ok-bg: #e8f5ee;
    --f-ok-border: #9fd3b3;
    --f-warn: #935b10;
    --f-warn-bg: #fff5e0;
    --f-warn-border: #f4c97a;
    --f-danger: #b91c1c;
    --f-danger-bg: #fef2f2;
    --f-danger-border: #fecaca;
  }

  :global(html, body) {
    height: 100%;
    background: var(--f-bg0);
    color: var(--text);
    font-family: var(--sans);
    font-size: 14px;
    line-height: 1.5;
    -webkit-font-smoothing: antialiased;
  }

  :global(body) {
    overflow: hidden;
  }

  :global(a) {
    color: inherit;
    text-decoration: none;
  }

  :global(button) {
    cursor: pointer;
    font-family: inherit;
  }

  :global(input, select, textarea) {
    font-family: inherit;
    color: var(--text);
  }

  :global(::-webkit-scrollbar) {
    width: 5px;
    height: 5px;
  }

  :global(::-webkit-scrollbar-track) {
    background: transparent;
  }

  :global(::-webkit-scrollbar-thumb) {
    background: var(--border-2);
    border-radius: 3px;
  }

  .app-shell {
    height: 100vh;
    padding: 10px;
    overflow: hidden;
    background: var(--f-bg0);
  }
</style>
