<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { formatDateShort, listPeers } from '../api';
  import type { File, Peer, HealthResponse } from '../types';

  export let health: HealthResponse | null = null;
  export let recentFiles: File[] = [];
  export let activeTag = '';
  export let allTags: string[] = [];

  let peers: Peer[] = [];
  let peersLoading = true;

  const dispatch = createEventDispatcher<{
    selectTag: string;
    selectFile: File;
  }>();

  onMount(async () => {
    try {
      peers = await listPeers();
      peers = peers.sort((a, b) => a.hostname.localeCompare(b.hostname));
      console.log(peers);
    } catch (e: unknown) {
      console.error(e);
      peers = [];
    } finally {
      peersLoading = false;
    }
  });
</script>

<aside>
  <!-- Machines -->
  <section>
    <h3>Machines</h3>
    {#if health?.caller_host}
      <div class="machine self">
        <span class="dot online"></span>
        <span class="machine-name">{health.caller_host}</span>
        <span class="badge you">you</span>
      </div>
    {/if}
    {#if peersLoading}
      <p class="muted">Loading peers…</p>
    {:else}
      {#each peers as peer}
        <div class="machine">
          <span class="dot" class:online={peer.online}></span>
          <span class="machine-name">{peer.hostname}</span>
          <span class="badge" class:offline={!peer.online}>{peer.online ? 'online' : 'offline'}</span>
        </div>
      {:else}
        {#if !health?.caller_host}
          <p class="muted">No machines found.</p>
        {/if}
      {/each}
    {/if}
  </section>

  <!-- Recent files -->
  <section>
    <h3>Recent</h3>
    {#if recentFiles.length === 0}
      <p class="muted">No files yet.</p>
    {:else}
      {#each recentFiles as file}
        <button class="recent-row" on:click={() => dispatch('selectFile', file)}>
          <span class="recent-name">{file.file_name}</span>
          <span class="recent-date">{formatDateShort(file.created_at)}</span>
        </button>
      {/each}
    {/if}
  </section>

  <!-- Tags -->
  {#if allTags.length > 0}
    <section>
      <h3>Tags</h3>
      <div class="tag-list">
        {#each allTags as tag}
          <button
            class="tag"
            class:active={activeTag === tag}
            on:click={() => dispatch('selectTag', tag)}
          >#{tag}</button>
        {/each}
      </div>
    </section>
  {/if}
</aside>

<style>
  aside {
    width: 210px; min-width: 210px;
    border-right: 1px solid var(--border);
    padding: 14px 10px;
    overflow-y: auto;
    display: flex; flex-direction: column; gap: 22px;
    background: var(--bg);
  }

  section { display: flex; flex-direction: column; gap: 5px; }

  h3 {
    font-size: 10px; font-weight: 600;
    text-transform: uppercase; letter-spacing: 0.08em;
    color: var(--text-3); padding: 0 6px; margin-bottom: 3px;
  }

  .machine {
    display: flex; align-items: center; gap: 7px;
    padding: 6px 8px; border-radius: var(--radius);
    background: var(--bg-2);
  }
  .machine.self { background: var(--bg-3); }

  .dot {
    width: 6px; height: 6px; border-radius: 50%;
    background: var(--border-2); flex-shrink: 0;
  }
  .dot.online { background: #16a34a; }

  .machine-name {
    font-family: var(--mono); font-size: 12px; flex: 1;
    overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
    color: var(--text);
  }

  .badge {
    font-size: 10px; color: var(--text-3);
    background: var(--bg); border: 1px solid var(--border);
    padding: 1px 5px; border-radius: 3px; flex-shrink: 0;
  }
  .badge.you { background: #eff6ff; border-color: #bfdbfe; color: #2563eb; }
  .badge.offline { opacity: 0.6; }

  .recent-row {
    display: flex; align-items: center; justify-content: space-between; gap: 6px;
    padding: 5px 8px; border-radius: var(--radius);
    background: none; border: none; width: 100%;
    text-align: left; cursor: pointer;
    transition: background 0.1s;
  }
  .recent-row:hover { background: var(--bg-2); }

  .recent-name {
    font-family: var(--mono); font-size: 11.5px; color: var(--text);
    overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  }
  .recent-date { font-size: 10px; color: var(--text-3); flex-shrink: 0; }

  .tag-list { display: flex; flex-wrap: wrap; gap: 4px; padding: 0 4px; }

  .tag {
    background: none; border: 1px solid var(--border); border-radius: 4px;
    padding: 2px 8px; font-size: 11px; font-family: var(--mono);
    color: var(--text-3); cursor: pointer; transition: all 0.1s;
  }
  .tag:hover, .tag.active { border-color: #3b82f6; color: #3b82f6; background: #eff6ff; }

  .muted { font-size: 11px; color: var(--text-3); padding: 2px 8px; font-style: italic; }
</style>