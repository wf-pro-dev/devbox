<script lang="ts">
  import type { HealthResponse, Peer, Location } from "../../types";
  import { onMount } from "svelte";
  import { getLocations } from "../../api";

  export let health: HealthResponse | null = null;
  export let allTags: Array<{ name: string; count: number; color: string }> = [];
  export let peers: Peer[] = [];
  export let activeTag = "";
  export let onSelectTag: (tag: string) => void = () => {};
  export let onSelectRoot: () => void = () => {};
  export let onSelectLocation: (hostname: string) => void = () => {};
  let locations: Location[] = [];
  let locationsLoading = true;

  onMount(async () => {
    try {
      locations = await getLocations();
      locationsLoading = false;
    } catch (e: unknown) {
      console.error(e);
    }
  });
</script>

<aside class="finder-sidebar">
  <section>
    <h3>Locations</h3>
    <button class="row loc" on:click={onSelectRoot}>
      <span class="dot home"></span>
      <span>Devbox</span>
    </button>
   {#if locationsLoading}
      <p class="muted">Loading locations…</p>
    {:else}
      {#each locations as location}
        <button class="row loc" on:click={() => onSelectLocation(location.Hostname)}>
          <span class="dot home"></span>
          <span>{location.Hostname}</span>
        </button>
      {/each}
    {/if}
  </section>

  <section>
    <h3>Tags</h3>
    {#if allTags.length === 0}
      <p class="muted">No tags loaded.</p>
    {:else}
      {#each allTags as tag}
        <button class="row tag-row" class:active={activeTag === tag.name} on:click={() => onSelectTag(tag.name)}>
          <span class="dot" style="background:{tag.color}"></span>
          <span class="label">{tag.name}</span>
          <span class="count">{tag.count}</span>
        </button>
      {/each}
    {/if}
  </section>

  <section>
    <h3>Machines</h3>
    {#if health?.caller_host}
      <div class="row machine self">
        <span class="dot online"></span>
        <span class="label">{health.caller_host}</span>
        <span class="you">you</span>
      </div>
    {/if}
    {#each peers as peer}
      <div class="row machine">
        <span class="dot" class:online={peer.status?.Online}></span>
        <span class="label">{peer.status?.HostName}</span>
      </div>
    {/each}
  </section>
</aside>

<style>
  .finder-sidebar {
    width: 152px;
    min-width: 152px;
    border-right: 0.5px solid var(--f-border);
    background: var(--f-bg1);
    padding: 10px 8px;
    display: flex;
    flex-direction: column;
    gap: 18px;
    overflow-y: auto;
  }
  section {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  h3 {
    font-size: 9.5px;
    font-weight: 600;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--f-text3);
    padding: 0 6px;
  }
  .row {
    display: flex;
    align-items: center;
    gap: 7px;
    min-height: 26px;
    padding: 5px 7px;
    border-radius: 6px;
    border: none;
    background: transparent;
    color: var(--f-text);
    font-size: 11.5px;
  }
  .loc:hover,
  .tag-row:hover {
    background: var(--f-bg2);
  }
  .tag-row.active {
    background: var(--f-selection);
    color: var(--f-accent);
  }
  .dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--f-border2);
    flex-shrink: 0;
  }
  .dot.home {
    background: var(--f-accent);
  }
  .dot.recent {
    background: var(--f-folder);
  }
  .dot.online {
    background: var(--f-ok);
  }
  .label {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .count,
  .you {
    font-family: var(--mono);
    font-size: 10px;
    color: var(--f-text3);
  }
  .you {
    color: var(--f-accent);
    border: 0.5px solid var(--f-accent-border);
    background: var(--f-accent-bg);
    padding: 1px 5px;
    border-radius: 4px;
  }
  .muted {
    padding: 4px 7px;
    font-size: 11px;
    color: var(--f-text3);
  }
</style>
