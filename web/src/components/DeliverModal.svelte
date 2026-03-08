<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import { api } from '../api.js';

  export let file;
  const dispatch = createEventDispatcher();

  let peers = [];
  let loading = true;
  let delivering = false;
  let error = '';
  let results = null;
  let selected = new Set();
  let destDir = '';

  onMount(async () => {
    try {
      peers = await api.listPeers();
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  });

  function togglePeer(hostname) {
    const s = new Set(selected);
    s.has(hostname) ? s.delete(hostname) : s.add(hostname);
    selected = s;
  }

  function selectAll() {
    selected = new Set(peers.filter(p => p.online).map(p => p.hostname));
  }

  async function deliver() {
    delivering = true; error = ''; results = null;
    try {
      const res = await api.deliverFile(file.id, [...selected], false, destDir);
      results = res.results;
    } catch (e) {
      error = e.message;
    } finally {
      delivering = false;
    }
  }

  function onKey(e) { if (e.key === 'Escape') dispatch('close'); }
</script>

<svelte:window on:keydown={onKey} />

<div class="backdrop" on:click={() => dispatch('close')} role="button" tabindex="-1" on:keydown={onKey}>
  <div class="modal" on:click|stopPropagation role="dialog">
    <div class="mhdr">
      <div>
        <h2>Deliver file</h2>
        <p class="subtitle">Push <span class="fname">{file.name}</span> to machines on your tailnet</p>
      </div>
      <button class="close" on:click={() => dispatch('close')}>×</button>
    </div>

    {#if loading}
      <div class="state">Loading peers…</div>
    {:else if error && !results}
      <div class="state err">{error}</div>
    {:else if results}
      <!-- Results view -->
      <div class="results">
        {#each results as r}
          <div class="result-row" class:ok={r.success} class:fail={!r.success}>
            <span class="result-icon">{r.success ? '✓' : '✗'}</span>
            <span class="result-host">{r.target}</span>
            {#if r.error}<span class="result-err">{r.error}</span>{/if}
          </div>
        {/each}
      </div>
      <div class="mftr">
        <button class="btn-cancel" on:click={() => dispatch('close')}>Done</button>
      </div>
    {:else}
      <!-- Peer picker -->
      <div class="peers">
        {#if peers.length === 0}
          <p class="empty">No peers found on your tailnet.</p>
        {:else}
          <div class="peers-hdr">
            <span class="peers-label">{selected.size} selected</span>
            <button class="select-all" on:click={selectAll}>Select all online</button>
          </div>
          {#each peers as peer}
            <button
              class="peer-row"
              class:checked={selected.has(peer.hostname)}
              class:offline={!peer.online}
              on:click={() => peer.online && togglePeer(peer.hostname)}
              disabled={!peer.online}
            >
              <span class="peer-check">{selected.has(peer.hostname) ? '☑' : '☐'}</span>
              <span class="peer-dot" class:online={peer.online}></span>
              <span class="peer-name">{peer.hostname}</span>
              <span class="peer-ip">{peer.ip}</span>
              {#if !peer.online}<span class="peer-status">offline</span>{/if}
            </button>
          {/each}
        {/if}
      </div>

      <div class="dest-row">
        <label class="dest-label">
          Destination directory
          <input
            type="text"
            placeholder="~/devbox-received"
            bind:value={destDir}
          />
        </label>
      </div>

      {#if error}<div class="err-msg">{error}</div>{/if}

      <div class="mftr">
        <button class="btn-cancel" on:click={() => dispatch('close')}>Cancel</button>
        <button class="btn-submit" on:click={deliver}
          disabled={selected.size === 0 || delivering}>
          {delivering ? 'Delivering…' : `Deliver to ${selected.size} machine${selected.size !== 1 ? 's' : ''}`}
        </button>
      </div>
    {/if}
  </div>
</div>

<style>
  .backdrop {
    position: fixed; inset: 0; background: rgba(0,0,0,0.3);
    display: flex; align-items: center; justify-content: center;
    z-index: 200; backdrop-filter: blur(2px);
  }
  .modal {
    background: white; border: 1px solid var(--border); border-radius: var(--radius-lg);
    width: 480px; max-width: 94vw; max-height: 80vh;
    box-shadow: 0 8px 32px rgba(0,0,0,0.12);
    display: flex; flex-direction: column; overflow: hidden;
  }
  .mhdr {
    display: flex; align-items: flex-start; justify-content: space-between;
    padding: 16px 20px; border-bottom: 1px solid var(--border); flex-shrink: 0;
  }
  .mhdr h2 { font-family: var(--serif); font-size: 18px; font-weight: normal; }
  .subtitle { font-size: 12px; color: var(--text-3); margin-top: 2px; }
  .fname { font-family: var(--mono); color: var(--text-2); }
  .close { background: none; border: none; font-size: 20px; color: var(--text-3); cursor: pointer; padding: 2px 6px; }
  .close:hover { color: var(--text); }

  .state { padding: 40px; text-align: center; color: var(--text-3); font-size: 13px; }
  .state.err { color: #dc2626; }

  .peers { flex: 1; overflow-y: auto; padding: 8px 0; }
  .peers-hdr {
    display: flex; align-items: center; justify-content: space-between;
    padding: 6px 20px 10px; 
  }
  .peers-label { font-size: 11px; color: var(--text-3); font-family: var(--mono); }
  .select-all { background: none; border: none; font-size: 11px; color: #2563eb; cursor: pointer; text-decoration: underline; }

  .peer-row {
    display: flex; align-items: center; gap: 10px;
    padding: 8px 20px; width: 100%; text-align: left;
    background: none; border: none; cursor: pointer; transition: background 0.1s;
  }
  .peer-row:hover:not(:disabled) { background: var(--bg-2); }
  .peer-row.checked { background: #eff6ff; }
  .peer-row.offline { opacity: 0.45; cursor: not-allowed; }

  .peer-check { font-size: 15px; color: #2563eb; width: 16px; flex-shrink: 0; }
  .peer-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--border-2); flex-shrink: 0; }
  .peer-dot.online { background: #16a34a; }
  .peer-name { font-family: var(--mono); font-size: 13px; color: var(--text); flex: 1; }
  .peer-ip { font-family: var(--mono); font-size: 11px; color: var(--text-3); }
  .peer-status { font-size: 10px; color: var(--text-3); background: var(--bg-3); padding: 1px 6px; border-radius: 3px; }

  .dest-row { padding: 12px 20px; border-top: 1px solid var(--border); flex-shrink: 0; }
  .dest-label { display: flex; flex-direction: column; gap: 5px; font-size: 12px; color: var(--text-2); }
  .dest-label input {
    height: 32px; padding: 0 10px; border: 1px solid var(--border);
    border-radius: var(--radius); font-size: 13px; font-family: var(--mono);
    background: var(--bg); outline: none;
  }
  .dest-label input:focus { border-color: var(--border-2); background: white; }

  .err-msg { margin: 0 20px 8px; padding: 8px 12px; background: #fef2f2; border: 1px solid #fecaca; border-radius: var(--radius); font-size: 12px; color: #dc2626; }

  .results { flex: 1; overflow-y: auto; padding: 12px 20px; display: flex; flex-direction: column; gap: 6px; }
  .result-row {
    display: flex; align-items: center; gap: 10px;
    padding: 8px 12px; border-radius: var(--radius);
    font-size: 13px;
  }
  .result-row.ok { background: #f0fdf4; }
  .result-row.fail { background: #fef2f2; }
  .result-icon { font-size: 14px; flex-shrink: 0; }
  .result-row.ok .result-icon { color: #16a34a; }
  .result-row.fail .result-icon { color: #dc2626; }
  .result-host { font-family: var(--mono); font-size: 13px; flex: 1; }
  .result-err { font-size: 11px; color: #dc2626; font-family: var(--mono); }

  .mftr { display: flex; gap: 8px; justify-content: flex-end; padding: 14px 20px; border-top: 1px solid var(--border); background: var(--bg); flex-shrink: 0; }
  .btn-cancel { height: 34px; padding: 0 16px; border: 1px solid var(--border); border-radius: var(--radius); background: white; font-size: 13px; color: var(--text-2); cursor: pointer; }
  .btn-cancel:hover { background: var(--bg-2); }
  .btn-submit { height: 34px; padding: 0 18px; background: var(--text); color: white; border: none; border-radius: var(--radius); font-size: 13px; font-weight: 500; cursor: pointer; transition: background 0.15s; }
  .btn-submit:hover:not(:disabled) { background: #3d3c38; }
  .btn-submit:disabled { opacity: 0.4; pointer-events: none; }
</style>