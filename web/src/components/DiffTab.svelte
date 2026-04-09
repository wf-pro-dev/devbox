<script lang="ts">
  import { api } from '../api';
  import type { File, DiffResult, ParsedHunk, ParsedLine } from '../types';

  export let file: File;
  /** Pre-selected node hostname when jumping here from the Status tab */
  export let preselectedNode: string = '';

  // ── Source ────────────────────────────────────────────────────────────────
  type DiffSource = 'node' | 'local';
  let source: DiffSource = 'node';

  let nodeInput  = preselectedNode;
  let versionInput = '';

  let localFile: globalThis.File | null = null;
  let localFileInput: HTMLInputElement;

  // ── Result ────────────────────────────────────────────────────────────────
  let result: DiffResult | null = null;
  let hunks: ParsedHunk[] = [];
  let additions = 0;
  let deletions = 0;
  let loading = false;
  let error = '';

  // ── View ──────────────────────────────────────────────────────────────────
  type ViewMode = 'inline' | 'split';
  let viewMode: ViewMode = 'inline';

  // When the parent sets preselectedNode reactively, sync it
  $: {
    if (preselectedNode) {
      nodeInput = preselectedNode;
      source = 'node';
    }
  }

  // ── Unified diff parser ───────────────────────────────────────────────────
  // Parses the raw "--- / +++ / @@ ... @@\n±lines" format into typed hunks.

  function parseUnified(raw: string): ParsedHunk[] {
    const parsed: ParsedHunk[] = [];
    if (!raw || !raw.trim()) return parsed;

    let currentHunk: ParsedHunk | null = null;
    let oldNo = 0;
    let newNo = 0;

    for (const line of raw.split('\n')) {
      // Hunk header: @@ -116,25 +116,13 @@
      const hunkMatch = line.match(/^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/);
      if (hunkMatch) {
        if (currentHunk) parsed.push(currentHunk);
        oldNo = parseInt(hunkMatch[1], 10);
        newNo = parseInt(hunkMatch[2], 10);
        currentHunk = {
          header: line,
          oldStart: oldNo,
          newStart: newNo,
          lines: [],
        };
        continue;
      }

      // Skip file header lines (--- / +++)
      if (line.startsWith('--- ') || line.startsWith('+++ ')) continue;

      if (!currentHunk) continue;

      if (line.startsWith('+')) {
        currentHunk.lines.push({ type: '+', content: line.slice(1), oldNo: null, newNo: newNo++ });
      } else if (line.startsWith('-')) {
        currentHunk.lines.push({ type: '-', content: line.slice(1), oldNo: oldNo++, newNo: null });
      } else {
        // Context line (starts with ' ' or is empty at end of hunk)
        const content = line.startsWith(' ') ? line.slice(1) : line;
        currentHunk.lines.push({ type: ' ', content, oldNo: oldNo++, newNo: newNo++ });
      }
    }

    if (currentHunk) parsed.push(currentHunk);
    return parsed;
  }

  function countChanges(hs: ParsedHunk[]): { additions: number; deletions: number } {
    let a = 0, d = 0;
    for (const h of hs) {
      for (const l of h.lines) {
        if (l.type === '+') a++;
        else if (l.type === '-') d++;
      }
    }
    return { additions: a, deletions: d };
  }

  // ── Run diff ──────────────────────────────────────────────────────────────
  async function runDiff() {
    if (source === 'node' && !nodeInput.trim()) {
      error = 'Node hostname is required.';
      return;
    }
    if (source === 'local' && !localFile) {
      error = 'Please select a local file.';
      return;
    }

    error  = '';
    result = null;
    hunks  = [];
    loading = true;

    try {
      if (source === 'node') {
        const ver = versionInput.trim() ? parseInt(versionInput.trim(), 10) : undefined;
        result = await api.diffNode(file.id, nodeInput.trim(), ver);
      } else {
        result = await api.diffLocal(file.id, localFile!);
      }

      if (result && !result.identical) {
        hunks = parseUnified(result.unified);
        const c = countChanges(hunks);
        additions = c.additions;
        deletions = c.deletions;
      }
    } catch (e: unknown) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  function handleFileInput(e: Event) {
    const input = e.target as HTMLInputElement;
    localFile = input.files?.[0] ?? null;
  }

  // ── Rendering helpers ─────────────────────────────────────────────────────

  function lineClass(type: ParsedLine['type']): string {
    if (type === '+') return 'line-add';
    if (type === '-') return 'line-del';
    return 'line-ctx';
  }

  function esc(s: string): string {
    return s
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;');
  }

  // Pair up deletions and additions for the split view
  function splitPairs(lines: ParsedLine[]): Array<{ left: ParsedLine | null; right: ParsedLine | null }> {
    const pairs: Array<{ left: ParsedLine | null; right: ParsedLine | null }> = [];
    const pendingDels: ParsedLine[] = [];

    for (const line of lines) {
      if (line.type === '-') {
        pendingDels.push(line);
      } else if (line.type === '+') {
        if (pendingDels.length) {
          pairs.push({ left: pendingDels.shift()!, right: line });
        } else {
          pairs.push({ left: null, right: line });
        }
      } else {
        // Context — flush pending deletions first
        while (pendingDels.length) pairs.push({ left: pendingDels.shift()!, right: null });
        pairs.push({ left: line, right: line });
      }
    }
    while (pendingDels.length) pairs.push({ left: pendingDels.shift()!, right: null });
    return pairs;
  }

  function copyRaw() {
    if (!result?.unified) return;
    navigator.clipboard.writeText(result.unified).catch(() => {});
  }
</script>

<div class="diff-pane">

  <!-- ── Controls bar ───────────────────────────────────────────────────── -->
  <div class="controls-bar">

    <!-- Source toggle -->
    <div class="seg-ctrl">
      <button class="seg" class:active={source === 'node'}
        on:click={() => { source = 'node'; result = null; hunks = []; }}>
        <svg viewBox="0 0 12 12" fill="none" width="10" height="10">
          <rect x="1" y="2" width="10" height="8" rx="1" stroke="currentColor" stroke-width="1.1"/>
          <path d="M4 5h4M4 7h2" stroke="currentColor" stroke-width="1.1" stroke-linecap="round"/>
        </svg>
        Node
      </button>
      <button class="seg" class:active={source === 'local'}
        on:click={() => { source = 'local'; result = null; hunks = []; }}>
        <svg viewBox="0 0 12 12" fill="none" width="10" height="10">
          <path d="M6 1v7M3 5l3 3 3-3" stroke="currentColor" stroke-width="1.1" stroke-linecap="round" stroke-linejoin="round"/>
          <path d="M1 9.5h10" stroke="currentColor" stroke-width="1.1" stroke-linecap="round"/>
        </svg>
        Local file
      </button>
    </div>

    <div class="ctrl-divider"></div>

    {#if source === 'node'}
      <input
        class="ctrl-input"
        placeholder="Hostname"
        bind:value={nodeInput}
        style="width:140px"
        on:keydown={(e) => e.key === 'Enter' && runDiff()}
      />
      <input
        class="ctrl-input"
        placeholder="Version (latest)"
        bind:value={versionInput}
        style="width:115px"
        on:keydown={(e) => e.key === 'Enter' && runDiff()}
      />
    {:else}
      <input
        type="file"
        bind:this={localFileInput}
        on:change={handleFileInput}
        style="display:none"
      />
      <button class="file-pick-btn" on:click={() => localFileInput.click()}>
        {#if localFile}
          <svg viewBox="0 0 12 12" fill="none" width="10" height="10">
            <path d="M2 6l2.5 2.5L10 3" stroke="#16a34a" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
          {localFile.name}
        {:else}
          Choose file…
        {/if}
      </button>
    {/if}

    <button class="run-btn" on:click={runDiff} disabled={loading}>
      {#if loading}
        <span class="mini-spinner"></span> Diffing…
      {:else}
        Compare
      {/if}
    </button>

    <!-- View toggle + copy — only when diff is loaded and has changes -->
    {#if result && !result.identical && hunks.length > 0}
      <div class="ctrl-divider"></div>
      <div class="seg-ctrl small">
        <button class="seg" class:active={viewMode === 'inline'} on:click={() => viewMode = 'inline'}>Inline</button>
        <button class="seg" class:active={viewMode === 'split'}  on:click={() => viewMode = 'split'}>Split</button>
      </div>
      <button class="ctrl-icon-btn" title="Copy raw diff" on:click={copyRaw}>
        <svg viewBox="0 0 12 12" fill="none" width="11" height="11">
          <rect x="3.5" y="3.5" width="6" height="7" rx="1" stroke="currentColor" stroke-width="1.1"/>
          <path d="M2 8V2.5A.5.5 0 012.5 2H7" stroke="currentColor" stroke-width="1.1" stroke-linecap="round"/>
        </svg>
      </button>
    {/if}
  </div>

  <!-- ── Error bar ──────────────────────────────────────────────────────── -->
  {#if error}
    <div class="err-bar">
      <svg viewBox="0 0 12 12" fill="none" width="11" height="11">
        <circle cx="6" cy="6" r="5" stroke="#ef4444" stroke-width="1.1"/>
        <path d="M6 3.5v2.8M6 8v.3" stroke="#ef4444" stroke-width="1.2" stroke-linecap="round"/>
      </svg>
      {error}
    </div>
  {/if}

  <!-- ── Body ───────────────────────────────────────────────────────────── -->
  <div class="diff-body">

    {#if loading}
      <div class="center-state">
        <div class="spinner-lg"></div>
        <p class="state-sub">Computing diff…</p>
      </div>

    {:else if !result}
      <!-- Prompt state -->
      <div class="center-state">
        <div class="empty-icon">
          <svg viewBox="0 0 52 52" fill="none" width="46" height="46">
            <rect x="4"  y="10" width="18" height="32" rx="2" stroke="var(--border-2)" stroke-width="1.4"/>
            <rect x="30" y="10" width="18" height="32" rx="2" stroke="var(--border-2)" stroke-width="1.4"/>
            <path d="M22 26h8M27 23l3 3-3 3" stroke="var(--border-2)" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M9 18h8M9 22h5M9 26h8M9 30h4"    stroke="var(--border-2)" stroke-width="1.2" stroke-linecap="round"/>
            <path d="M35 18h8M35 22h5M35 26h8M35 30h4" stroke="var(--border-2)" stroke-width="1.2" stroke-linecap="round"/>
          </svg>
        </div>
        <p class="state-title">No diff loaded</p>
        <p class="state-sub">Select a node or upload a local file, then click <strong>Compare</strong>.</p>
      </div>

    {:else if result.identical}
      <!-- Identical -->
      <div class="diff-labels">
        <span class="label-old">{result.vault_label}</span>
        <span class="label-arrow">→</span>
        <span class="label-new">{result.node_label}</span>
      </div>
      <div class="center-state">
        <svg viewBox="0 0 16 16" fill="none" width="24" height="24">
          <circle cx="8" cy="8" r="6.5" stroke="#22c55e" stroke-width="1.3"/>
          <path d="M5 8l2 2 4-4" stroke="#22c55e" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        <p class="identical-msg">Files are byte-identical</p>
      </div>

    {:else}
      <!-- Diff header bar -->
      <div class="diff-labels">
        <span class="label-old">{result.vault_label}</span>
        <span class="label-arrow">→</span>
        <span class="label-new">{result.node_label}</span>
        <span class="diff-stat-add">+{additions}</span>
        <span class="diff-stat-del">-{deletions}</span>
      </div>

      <!-- Hunks -->
      {#each hunks as hunk, hi (hi)}
        <div class="hunk-header">{hunk.header}</div>

        {#if viewMode === 'inline'}
          <!-- INLINE VIEW -->
          <table class="diff-table">
            <colgroup>
              <col class="col-lno"/>
              <col class="col-lno"/>
              <col class="col-gutter"/>
              <col class="col-code"/>
            </colgroup>
            <tbody>
              {#each hunk.lines as line, li (li)}
                <tr class="diff-row {lineClass(line.type)}">
                  <td class="lno">{line.type !== '+' && line.oldNo !== null ? line.oldNo : ''}</td>
                  <td class="lno">{line.type !== '-' && line.newNo !== null ? line.newNo : ''}</td>
                  <td class="gutter">{line.type === ' ' ? '' : line.type}</td>
                  <td class="code"><pre>{@html esc(line.content)}</pre></td>
                </tr>
              {/each}
            </tbody>
          </table>

        {:else}
          <!-- SPLIT VIEW -->
          <table class="diff-table split">
            <colgroup>
              <col class="col-lno"/>
              <col class="col-code-half"/>
              <col class="col-lno"/>
              <col class="col-code-half"/>
            </colgroup>
            <tbody>
              {#each splitPairs(hunk.lines) as pair, pi (pi)}
                <tr class="diff-row split-row">
                  <td class="lno {pair.left?.type === '-' ? 'lno-del' : ''}">{pair.left?.oldNo ?? ''}</td>
                  <td class="code {pair.left ? lineClass(pair.left.type) : 'line-empty'}">
                    {#if pair.left}
                      <span class="split-gutter">{pair.left.type === ' ' ? '' : pair.left.type}</span>
                      <pre>{@html esc(pair.left.content)}</pre>
                    {/if}
                  </td>
                  <td class="lno split-sep {pair.right?.type === '+' ? 'lno-add' : ''}">{pair.right?.newNo ?? ''}</td>
                  <td class="code {pair.right ? lineClass(pair.right.type) : 'line-empty'}">
                    {#if pair.right}
                      <span class="split-gutter">{pair.right.type === ' ' ? '' : pair.right.type}</span>
                      <pre>{@html esc(pair.right.content)}</pre>
                    {/if}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        {/if}
      {/each}
    {/if}

  </div>
</div>

<style>
  .diff-pane {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    min-height: 0;
  }

  /* ── Controls ─────────────────────────────────────────────────────────── */
  .controls-bar {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 14px;
    border-bottom: 1px solid var(--border);
    background: var(--bg);
    flex-shrink: 0;
    flex-wrap: wrap;
  }
  .ctrl-divider {
    width: 1px; height: 18px;
    background: var(--border);
    margin: 0 2px;
  }

  .seg-ctrl {
    display: flex;
    background: var(--bg-2);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    overflow: hidden;
  }
  .seg-ctrl.small .seg { padding: 3px 9px; font-size: 11px; }

  .seg {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 4px 10px;
    font-size: 11.5px;
    color: var(--text-2);
    background: none;
    border: none;
    cursor: pointer;
    transition: color 0.1s;
    white-space: nowrap;
  }
  .seg:hover { color: var(--text); }
  .seg.active {
    background: white;
    color: var(--text);
    font-weight: 500;
    box-shadow: 0 1px 3px rgba(0,0,0,0.08);
  }

  .ctrl-input {
    height: 28px;
    padding: 0 9px;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    font-size: 12px;
    font-family: var(--mono);
    background: white;
    color: var(--text);
    outline: none;
    transition: border-color 0.1s;
  }
  .ctrl-input:focus { border-color: var(--text); }

  .file-pick-btn {
    display: flex;
    align-items: center;
    gap: 5px;
    height: 28px;
    padding: 0 10px;
    border: 1px dashed var(--border-2);
    border-radius: var(--radius);
    font-size: 11.5px;
    font-family: var(--mono);
    color: var(--text-2);
    background: var(--bg-2);
    cursor: pointer;
    max-width: 200px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    transition: border-color 0.1s;
  }
  .file-pick-btn:hover { border-color: var(--text); color: var(--text); }

  .run-btn {
    display: flex;
    align-items: center;
    gap: 5px;
    height: 28px;
    padding: 0 14px;
    background: var(--text);
    color: white;
    border: none;
    border-radius: var(--radius);
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    white-space: nowrap;
    transition: background 0.1s;
  }
  .run-btn:hover:not(:disabled) { background: #3d3c38; }
  .run-btn:disabled { opacity: 0.5; pointer-events: none; }

  .ctrl-icon-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px; height: 28px;
    background: none;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    color: var(--text-2);
    cursor: pointer;
    transition: all 0.1s;
  }
  .ctrl-icon-btn:hover { background: var(--bg-2); color: var(--text); }

  .mini-spinner {
    display: inline-block;
    width: 10px; height: 10px;
    border: 1.5px solid rgba(255,255,255,0.35);
    border-top-color: white;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  /* ── Error bar ───────────────────────────────────────────────────────── */
  .err-bar {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 16px;
    background: #fef2f2;
    border-bottom: 1px solid #fecaca;
    font-size: 12px;
    color: #dc2626;
    flex-shrink: 0;
  }

  /* ── Body ────────────────────────────────────────────────────────────── */
  .diff-body {
    flex: 1;
    overflow-y: auto;
    overflow-x: auto;
    min-height: 0;
    background: white;
  }

  /* ── Center states ───────────────────────────────────────────────────── */
  .center-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 10px;
    padding: 56px 20px;
    text-align: center;
  }
  .empty-icon { opacity: 0.45; }
  .state-title { font-size: 14px; font-weight: 500; color: var(--text); margin: 0; }
  .state-sub   { font-size: 12.5px; color: var(--text-3); margin: 0; max-width: 320px; }
  .identical-msg { font-size: 13.5px; font-weight: 500; color: #16a34a; margin: 0; }
  .spinner-lg {
    width: 22px; height: 22px;
    border: 2px solid var(--border-2);
    border-top-color: var(--text-3);
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
  }

  /* ── Diff labels bar ────────────────────────────────────────────────── */
  .diff-labels {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 7px 16px;
    border-bottom: 1px solid var(--border);
    background: var(--bg);
    font-size: 11px;
    position: sticky;
    top: 0;
    z-index: 3;
    overflow: hidden;
    min-width: 0;
  }
  .label-old {
    font-family: var(--mono);
    color: var(--text-3);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    min-width: 0;
    flex-shrink: 1;
  }
  .label-arrow { color: var(--text-3); flex-shrink: 0; }
  .label-new {
    font-family: var(--mono);
    color: var(--text-2);
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    min-width: 0;
    flex-shrink: 2;
  }
  .diff-stat-add { margin-left: auto; flex-shrink: 0; font-family: var(--mono); font-size: 11.5px; font-weight: 600; color: #16a34a; }
  .diff-stat-del { flex-shrink: 0; font-family: var(--mono); font-size: 11.5px; font-weight: 600; color: #dc2626; }

  /* ── Hunk header ────────────────────────────────────────────────────── */
  .hunk-header {
    padding: 4px 14px;
    background: #eff6ff;
    border-top: 1px solid #bfdbfe;
    border-bottom: 1px solid #bfdbfe;
    font-family: var(--mono);
    font-size: 11px;
    color: #3b82f6;
    position: sticky;
    top: 33px; /* below .diff-labels */
    z-index: 2;
  }

  /* ── Diff table ─────────────────────────────────────────────────────── */
  .diff-table {
    width: 100%;
    border-collapse: collapse;
    font-family: var(--mono);
    font-size: 12px;
    line-height: 1.6;
  }

  .col-lno       { width: 46px; }
  .col-gutter    { width: 20px; }
  .col-code      { width: auto; }
  .col-code-half { width: 50%; }

  .diff-row td { padding: 0; border: none; vertical-align: top; }

  /* Line number cells */
  .lno {
    padding: 0 6px;
    text-align: right;
    color: rgba(0,0,0,0.28);
    font-size: 11px;
    user-select: none;
    border-right: 1px solid rgba(0,0,0,0.06);
    background: rgba(0,0,0,0.014);
    min-width: 38px;
    white-space: nowrap;
  }
  .lno-del { background: rgba(220,38,38,0.06); }
  .lno-add { background: rgba(22,163,74,0.06); }

  /* Gutter +/- sigil (inline mode) */
  .gutter {
    padding: 0 4px;
    text-align: center;
    color: rgba(0,0,0,0.28);
    user-select: none;
    font-size: 13px;
    width: 18px;
    min-width: 18px;
  }

  /* Split gutter (prepended inside code cell) */
  .split-gutter {
    display: inline-block;
    width: 14px;
    color: rgba(0,0,0,0.28);
    user-select: none;
    font-size: 13px;
    text-align: center;
  }

  /* Split vertical divider */
  .split-sep { border-left: 2px solid var(--border); }

  /* Code cells */
  .code {
    padding: 0 6px;
    white-space: pre;
    overflow: hidden;
  }
  .code pre {
    margin: 0;
    padding: 0;
    font-family: inherit;
    font-size: inherit;
    line-height: inherit;
    white-space: pre;
    display: inline;
  }

  /* Line colours */
  .line-add  { background: #f0fdf4; }
  .line-del  { background: #fff5f5; }
  .line-ctx  { background: white; }
  .line-empty { background: #f9fafb; }
</style>
