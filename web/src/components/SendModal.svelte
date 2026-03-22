<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { listPeers, api, sendDirectory } from '../api';
  import type { File, Directory, Peer, SendResult } from '../types';

  /** Pass either a file OR a dir */
  export let file: File | null = null;
  export let dir: Directory | null = null;

  const dispatch = createEventDispatcher<{ close: void }>();

  let peers: Peer[] = [];
  let loading = true;
  let delivering = false;
  let error = '';
  let results: SendResult[] | null = null;
  let selected = new Set<string>();
  let destDir = '';

  $: label = file ? file.file_name : (dir ? dir.name : '');

  onMount(async () => {
    try {
      peers = await listPeers();
    } catch (e: unknown) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  });

  function togglePeer(hostname: string) {
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
      const targets = [...selected];
      let res;
      if (file) {
        res = await api.sendFile(file.id, targets, false, destDir);
      } else if (dir) {
        res = await sendDirectory(dir.id, targets, false, destDir);
      }
      results = res ?? [];
      console.log(results)
    } catch (e: unknown) {
      error = (e as Error).message;
    } finally {
      delivering = false;
    }
  }

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Escape') dispatch('close');
  }
</script>

<svelte:window on:keydown={onKey} />

<!-- svelte-ignore a11y-no-static-element-interactions -->
<div
  class="backdrop"
  on:click={() => dispatch('close')}
  on:keydown={(e) => e.key === 'Escape' && dispatch('close')}
>
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div class="modal" on:click|stopPropagation role="dialog" aria-modal="true">
    <div class="mhdr">
      <div>
        <h2>Send {dir ? 'directory' : 'file'}</h2>
        <p class="subtitle">
          Push <span class="fname">{label}</span> to machines on your tailnet
        </p>
      </div>
      <button class="close" on:click={() => dispatch('close')}>
        <svg viewBox="0 0 16 16" fill="none" width="14" height="14">
          <path d="M3 3l10 10M13 3L3 13" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
      </button>
    </div>

    {#if loading}
      <div class="state">Loading peers…</div>
    {:else if error && !results}
      <div class="state err">{error}</div>
    {:else if results}
      <div class="results">
        {#each results as r}
          <div class="result-row" class:ok={r.success} class:fail={!r.success}>
            <span class="result-icon">
              {#if r.success}
                <svg viewBox="0 0 16 16" fill="none" width="14" height="14">
                  <path d="M3 8l3.5 3.5L13 4" stroke="#16a34a" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/>
                </svg>
              {:else}
                <svg viewBox="0 0 16 16" fill="none" width="14" height="14">
                  <path d="M4 4l8 8M12 4l-8 8" stroke="#dc2626" stroke-width="1.6" stroke-linecap="round"/>
                </svg>
              {/if}
            </span>
            <span class="result-host">{r.dest_machine}</span>
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
              <span class="peer-check">
                {#if selected.has(peer.hostname)}
                  <svg viewBox="0 0 14 14" fill="none" width="13" height="13">
                    <rect x="0.7" y="0.7" width="12.6" height="12.6" rx="3" fill="#2563eb" stroke="#2563eb" stroke-width="1"/>
                    <path d="M3.5 7l2.5 2.5L10.5 4.5" stroke="white" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
                  </svg>
                {:else}
                  <svg viewBox="0 0 14 14" fill="none" width="13" height="13">
                    <rect x="0.7" y="0.7" width="12.6" height="12.6" rx="3" stroke="var(--border-2)" stroke-width="1"/>
                  </svg>
                {/if}
              </span>
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
            placeholder="/var/lib/tailkitd/recv/devbox"
            bind:value={destDir}
          />
        </label>
      </div>

      {#if error}<div class="err-msg">{error}</div>{/if}

      <div class="mftr">
        <button class="btn-cancel" on:click={() => dispatch('close')}>Cancel</button>
        <button
          class="btn-submit"
          on:click={deliver}
          disabled={selected.size === 0 || delivering}
        >
          {delivering
            ? 'Sending…'
            : `Send to ${selected.size} machine${selected.size !== 1 ? 's' : ''}`}
        </button>
      </div>
    {/if}
  </div>
</div>

<style>
  .backdrop {
    position: fixed; inset: 0; background: rgba(0,0,0,0.35);
    display: flex; align-items: center; justify-content: center;
    z-index: 200; backdrop-filter: blur(3px);
  }
  .modal {
    background: white; border: 1px solid var(--border); border-radius: var(--radius-lg);
    width: 480px; max-width: 94vw; max-height: 80vh;
    box-shadow: 0 12px 40px rgba(0,0,0,0.14);
    display: flex; flex-direction: column; overflow: hidden;
  }
  .mhdr {
    display: flex; align-items: flex-start; justify-content: space-between;
    padding: 16px 20px; border-bottom: 1px solid var(--border); flex-shrink: 0;
  }
  .mhdr h2 { font-family: var(--serif); font-size: 18px; font-weight: normal; }
  .subtitle { font-size: 12px; color: var(--text-3); margin-top: 3px; }
  .fname { font-family: var(--mono); color: var(--text-2); }
  .close {
    display: flex; align-items: center; justify-content: center;
    width: 28px; height: 28px;
    background: none; border: none; color: var(--text-3);
    cursor: pointer; border-radius: var(--radius); transition: all 0.1s;
  }
  .close:hover { background: var(--bg-2); color: var(--text); }

  .state { padding: 40px; text-align: center; color: var(--text-3); font-size: 13px; }
  .state.err { color: #dc2626; }

  .peers { flex: 1; overflow-y: auto; padding: 8px 0; }
  .peers-hdr {
    display: flex; align-items: center; justify-content: space-between;
    padding: 6px 20px 10px;
  }
  .peers-label { font-size: 11px; color: var(--text-3); font-family: var(--mono); }
  .select-all {
    background: none; border: none; font-size: 11px; color: #2563eb;
    cursor: pointer; text-decoration: underline;
  }
  .peer-row {
    display: flex; align-items: center; gap: 10px;
    padding: 8px 20px; width: 100%; text-align: left;
    background: none; border: none; cursor: pointer; transition: background 0.1s;
  }
  .peer-row:hover:not(:disabled) { background: var(--bg-2); }
  .peer-row.checked { background: #eff6ff; }
  .peer-row.offline { opacity: 0.45; cursor: not-allowed; }
  .peer-check { display: flex; align-items: center; flex-shrink: 0; }
  .peer-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--border-2); flex-shrink: 0; }
  .peer-dot.online { background: #16a34a; }
  .peer-name { font-family: var(--mono); font-size: 13px; color: var(--text); flex: 1; }
  .peer-ip { font-family: var(--mono); font-size: 11px; color: var(--text-3); }
  .peer-status {
    font-size: 10px; color: var(--text-3);
    background: var(--bg-3); padding: 1px 6px; border-radius: 3px;
  }

  .dest-row { padding: 12px 20px; border-top: 1px solid var(--border); flex-shrink: 0; }
  .dest-label { display: flex; flex-direction: column; gap: 5px; font-size: 12px; color: var(--text-2); }
  .dest-label input {
    height: 32px; padding: 0 10px; border: 1px solid var(--border);
    border-radius: var(--radius); font-size: 13px; font-family: var(--mono);
    background: var(--bg); outline: none;
  }
  .dest-label input:focus { border-color: var(--border-2); background: white; }

  .err-msg {
    margin: 0 20px 8px; padding: 8px 12px;
    background: #fef2f2; border: 1px solid #fecaca;
    border-radius: var(--radius); font-size: 12px; color: #dc2626;
  }

  .results { flex: 1; overflow-y: auto; padding: 12px 20px; display: flex; flex-direction: column; gap: 6px; }
  .result-row {
    display: flex; align-items: center; gap: 10px;
    padding: 8px 12px; border-radius: var(--radius); font-size: 13px;
  }
  .result-row.ok { background: #f0fdf4; }
  .result-row.fail { background: #fef2f2; }
  .result-icon { flex-shrink: 0; display: flex; }
  .result-host { font-family: var(--mono); font-size: 13px; flex: 1; }
  .result-err { font-size: 11px; color: #dc2626; font-family: var(--mono); }

  .empty { padding: 40px 20px; text-align: center; font-size: 13px; color: var(--text-3); }

  .mftr {
    display: flex; gap: 8px; justify-content: flex-end;
    padding: 14px 20px; border-top: 1px solid var(--border);
    background: var(--bg); flex-shrink: 0;
  }
  .btn-cancel {
    height: 34px; padding: 0 16px; border: 1px solid var(--border);
    border-radius: var(--radius); background: white; font-size: 13px;
    color: var(--text-2); cursor: pointer;
  }
  .btn-cancel:hover { background: var(--bg-2); }
  .btn-submit {
    height: 34px; padding: 0 18px; background: var(--text);
    color: white; border: none; border-radius: var(--radius);
    font-size: 13px; font-weight: 500; cursor: pointer; transition: background 0.15s;
  }
  .btn-submit:hover:not(:disabled) { background: #3d3c38; }
  .btn-submit:disabled { opacity: 0.4; pointer-events: none; }
</style>