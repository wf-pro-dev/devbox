<script lang="ts">
  import { api } from '../api';
  import type { File, NodeDriftResult } from '../types';
  import { createEventDispatcher } from 'svelte';

  export let file: File;

  const dispatch = createEventDispatcher<{ diffNode: string }>();

  let results: NodeDriftResult[] = [];
  let loading = false;
  let error = '';
  let loaded = false;

  // ── Status classification ─────────────────────────────────────────────────
  // The server returns raw strings like "MATCH (latest)", "NOT FOUND",
  // "NO TAILKITD FOUND", "DRIFTED". We map these to display info.

  type StatusKind = 'match' | 'drifted' | 'not_found' | 'unreachable' | 'unknown';

  function classifyStatus(s: string): StatusKind {
    const u = s.toUpperCase();
    if (u.startsWith('MATCH'))            return 'match';
    if (u.includes('DRIFTED'))            return 'drifted';
    if (u === 'NOT FOUND')                return 'not_found';
    if (u.includes('NO TAILKITD'))        return 'unreachable';
    return 'unknown';
  }

  function badgeClass(s: string): string {
    const k = classifyStatus(s);
    if (k === 'match')       return 'badge-ok';
    if (k === 'drifted')     return 'badge-warn';
    if (k === 'not_found')   return 'badge-missing';
    return 'badge-err';
  }

  function dotClass(s: string): string {
    const k = classifyStatus(s);
    if (k === 'match')       return 'dot-ok';
    if (k === 'drifted')     return 'dot-warn';
    if (k === 'not_found')   return 'dot-missing';
    return 'dot-err';
  }

  function isDrifted(s: string): boolean {
    return classifyStatus(s) === 'drifted';
  }

  function isRowHighlighted(s: string): boolean {
    return classifyStatus(s) === 'drifted';
  }

  // ── Summary counts ────────────────────────────────────────────────────────
  $: matchCount       = results.filter(r => classifyStatus(r.status) === 'match').length;
  $: driftedCount     = results.filter(r => classifyStatus(r.status) === 'drifted').length;
  $: notFoundCount    = results.filter(r => classifyStatus(r.status) === 'not_found').length;
  $: unreachableCount = results.filter(r => ['unreachable','unknown'].includes(classifyStatus(r.status))).length;

  async function load() {
    loading = true;
    error = '';
    try {
      results = await api.getFileStatus(file.id);
      loaded = true;
    } catch (e: unknown) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }
</script>

<div class="status-pane">

  <!-- ── Summary bar (shown after first load) ──────────────────────────── -->
  {#if loaded}
    <div class="summary-bar">
      <span class="sum-chip ok">
        <span class="dot"></span>{matchCount} in sync
      </span>
      {#if driftedCount > 0}
        <span class="sum-chip warn">
          <span class="dot"></span>{driftedCount} drifted
        </span>
      {/if}
      {#if notFoundCount > 0}
        <span class="sum-chip missing">
          <span class="dot"></span>{notFoundCount} not found
        </span>
      {/if}
      {#if unreachableCount > 0}
        <span class="sum-chip err">
          <span class="dot"></span>{unreachableCount} unreachable
        </span>
      {/if}
      <div class="sum-spacer"></div>
      <button class="refresh-btn" on:click={load} disabled={loading}>
        <svg viewBox="0 0 14 14" fill="none" width="11" height="11" class:spinning={loading}>
          <path d="M12 7A5 5 0 1 1 7 2" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/>
          <path d="M7 1v3h3" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        Refresh
      </button>
    </div>
  {/if}

  <!-- ── Loading ───────────────────────────────────────────────────────── -->
  {#if loading}
    <div class="state-msg">
      <div class="spinner"></div>
      Checking fleet…
    </div>

  <!-- ── Error ─────────────────────────────────────────────────────────── -->
  {:else if error}
    <div class="state-msg err-msg">
      <svg viewBox="0 0 16 16" fill="none" width="16" height="16">
        <circle cx="8" cy="8" r="6.5" stroke="#ef4444" stroke-width="1.3"/>
        <path d="M8 5v3.5M8 10.5v.5" stroke="#ef4444" stroke-width="1.4" stroke-linecap="round"/>
      </svg>
      {error}
      <button class="retry-btn" on:click={load}>Retry</button>
    </div>

  <!-- ── Empty state (not yet run) ─────────────────────────────────────── -->
  {:else if !loaded}
    <div class="empty-state">
      <div class="empty-icon">
        <svg viewBox="0 0 48 48" fill="none" width="44" height="44">
          <circle cx="24" cy="24" r="20" stroke="var(--border-2)" stroke-width="1.5"/>
          <circle cx="24" cy="24" r="8"  stroke="var(--border-2)" stroke-width="1.5"/>
          <path d="M24 4v6M24 38v6M4 24h6M38 24h6" stroke="var(--border-2)" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
      </div>
      <p class="empty-title">Check fleet status</p>
      <p class="empty-sub">Compare this file against every online tailkit node.</p>
      <button class="check-btn" on:click={load}>
        <svg viewBox="0 0 14 14" fill="none" width="11" height="11">
          <circle cx="7" cy="7" r="5.5" stroke="currentColor" stroke-width="1.3"/>
          <path d="M4.5 7l1.8 1.8L9.5 5" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        Run check
      </button>
    </div>

  <!-- ── No peers ───────────────────────────────────────────────────────── -->
  {:else if results.length === 0}
    <div class="state-msg muted">No online peers found.</div>

  <!-- ── Results table ─────────────────────────────────────────────────── -->
  {:else}
    <div class="table-wrap">
      <table class="status-table">
        <thead>
          <tr>
            <th>Node</th>
            <th>Status</th>
            <th>Local path</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {#each results as r (r.hostname)}
            <tr class="row" class:row-drifted={isRowHighlighted(r.status)}>

              <!-- Node -->
              <td class="node-cell">
                <span class="node-dot {dotClass(r.status)}"></span>
                <span class="node-name">{r.hostname}</span>
              </td>

              <!-- Status badge — shows the raw server string -->
              <td>
                <span class="badge {badgeClass(r.status)}">{r.status}</span>
              </td>

              <!-- Local path on that node -->
              <td class="path-cell">
                <span class="path-val" title={r.local_path}>{r.local_path}</span>
              </td>

              <!-- Actions -->
              <td class="actions-cell">
                {#if isDrifted(r.status)}
                  <button
                    class="diff-link"
                    on:click={() => dispatch('diffNode', r.hostname)}
                    title="Open diff for {r.hostname}"
                  >
                    View diff →
                  </button>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}

</div>

<style>
  .status-pane {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    min-height: 0;
  }

  /* ── Summary bar ────────────────────────────────────────────────────── */
  .summary-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 9px 18px;
    border-bottom: 1px solid var(--border);
    background: var(--bg);
    flex-shrink: 0;
  }
  .sum-spacer { flex: 1; }

  .sum-chip {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-size: 11.5px;
    font-weight: 500;
    padding: 2px 9px;
    border-radius: 20px;
    border: 1px solid;
  }
  .sum-chip .dot {
    width: 6px; height: 6px;
    border-radius: 50%;
  }
  .sum-chip.ok      { background: #f0fdf4; border-color: #bbf7d0; color: #15803d; }
  .sum-chip.ok .dot { background: #22c55e; }
  .sum-chip.warn      { background: #fffbeb; border-color: #fde68a; color: #92400e; }
  .sum-chip.warn .dot { background: #f59e0b; }
  .sum-chip.missing      { background: #f5f3ff; border-color: #ddd6fe; color: #6d28d9; }
  .sum-chip.missing .dot { background: #a78bfa; }
  .sum-chip.err      { background: #fef2f2; border-color: #fecaca; color: #991b1b; }
  .sum-chip.err .dot { background: #ef4444; }

  .refresh-btn {
    display: flex;
    align-items: center;
    gap: 4px;
    background: none;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 3px 10px;
    font-size: 11.5px;
    color: var(--text-2);
    cursor: pointer;
    transition: all 0.1s;
  }
  .refresh-btn:hover { background: var(--bg-2); border-color: var(--border-2); }
  .refresh-btn:disabled { opacity: 0.4; pointer-events: none; }

  .spinning { animation: spin 0.8s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }

  /* ── States ─────────────────────────────────────────────────────────── */
  .state-msg {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 40px;
    font-size: 13px;
    color: var(--text-3);
    flex: 1;
  }
  .state-msg.muted { font-style: italic; }
  .err-msg { color: #dc2626; }

  .retry-btn {
    margin-left: 6px;
    background: none;
    border: 1px solid #fca5a5;
    border-radius: var(--radius);
    color: #dc2626;
    font-size: 11px;
    padding: 2px 8px;
    cursor: pointer;
  }

  .spinner {
    width: 14px; height: 14px;
    border: 2px solid var(--border-2);
    border-top-color: var(--text-3);
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
  }

  .empty-state {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 40px 20px;
    text-align: center;
  }
  .empty-icon { opacity: 0.4; margin-bottom: 4px; }
  .empty-title { font-size: 14px; font-weight: 500; color: var(--text); margin: 0; }
  .empty-sub   { font-size: 12.5px; color: var(--text-3); margin: 0; max-width: 320px; }

  .check-btn {
    display: flex;
    align-items: center;
    gap: 5px;
    margin-top: 8px;
    padding: 7px 18px;
    background: var(--text);
    color: white;
    border: none;
    border-radius: var(--radius);
    font-size: 12.5px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.1s;
  }
  .check-btn:hover { background: #3d3c38; }

  /* ── Table ───────────────────────────────────────────────────────────── */
  .table-wrap {
    flex: 1;
    overflow-y: auto;
    min-height: 0;
  }

  .status-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 12.5px;
  }
  .status-table th {
    text-align: left;
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-3);
    padding: 8px 14px;
    border-bottom: 1px solid var(--border);
    position: sticky;
    top: 0;
    background: white;
    z-index: 1;
    white-space: nowrap;
  }
  .status-table td {
    padding: 10px 14px;
    border-bottom: 1px solid var(--border);
    vertical-align: middle;
  }
  .row:last-child td { border-bottom: none; }
  .row:hover td { background: var(--bg); }
  .row-drifted td { background: #fffdf5; }
  .row-drifted:hover td { background: #fffbeb; }

  /* Node column */
  .node-cell {
    display: flex;
    align-items: center;
    gap: 8px;
    white-space: nowrap;
  }
  .node-dot {
    width: 7px; height: 7px;
    border-radius: 50%;
    flex-shrink: 0;
  }
  .dot-ok      { background: #22c55e; }
  .dot-warn    { background: #f59e0b; }
  .dot-missing { background: #a78bfa; }
  .dot-err     { background: #ef4444; }
  .node-name {
    font-family: var(--mono);
    font-size: 12px;
    font-weight: 500;
    color: var(--text);
  }

  /* Badge */
  .badge {
    display: inline-block;
    padding: 2px 8px;
    border-radius: 3px;
    font-size: 10.5px;
    font-weight: 600;
    border: 1px solid;
    white-space: nowrap;
  }
  .badge-ok      { background: #f0fdf4; border-color: #bbf7d0; color: #15803d; }
  .badge-warn    { background: #fffbeb; border-color: #fde68a; color: #92400e; }
  .badge-missing { background: #f5f3ff; border-color: #ddd6fe; color: #6d28d9; }
  .badge-err     { background: #fef2f2; border-color: #fecaca; color: #991b1b; }

  /* Path column */
  .path-cell {
    max-width: 320px;
    overflow: hidden;
  }
  .path-val {
    font-family: var(--mono);
    font-size: 11px;
    color: var(--text-3);
    display: block;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  /* Actions */
  .actions-cell { white-space: nowrap; }
  .diff-link {
    background: none;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 3px 9px;
    font-size: 11px;
    color: var(--text-2);
    cursor: pointer;
    transition: all 0.1s;
  }
  .diff-link:hover {
    background: var(--text);
    color: white;
    border-color: var(--text);
  }
</style>
