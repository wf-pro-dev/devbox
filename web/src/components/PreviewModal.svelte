<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { api, formatBytes, formatDate, langColor } from "../api";
  import SendModal from "./SendModal.svelte";
  import UpdateFileModal from "./UpdateFileModal.svelte";
  import VersionRow from "./VersionRow.svelte";
  import FleetStatusTab from "./FleetStatusTab.svelte";
  import DiffTab from "./DiffTab.svelte";
  import type { File, Version, UpdateResponse } from "../types";
  import { toast } from "svelte-sonner";

  import { HighlightAuto, LineNumbers } from "svelte-highlight";
  import { github } from "svelte-highlight/styles";

  export let file: File;
  let language = "text";

  const dispatch = createEventDispatcher<{
    close: void;
    deleted: string;
    tagsUpdated: File;
  }>();

  type Tab = "preview" | "meta" | "versions" | "status" | "diff";
  let tab: Tab = "preview";

  // Diff tab: node pre-selected when "View diff →" is clicked from Status tab
  let diffPreselectedNode = '';

  // Preview tab state
  let content = "";
  let contentLoading = true;
  let editing = false;
  let editorContent = "";
  let saving = false;
  let copied = false;
  let showUpdateModal = false;

  // Meta tab state
  let newTag = "";
  let deleting = false;
  let editingDescription = false;
  let descriptionDraft = "";
  let savingDescription = false;

  // Versions tab state
  let versions: Version[] = [];
  let versionsLoading = false;

  // Shared
  let showDeliver = false;

  // ── Content loading ────────────────────────────────────────────────────────

  async function loadContent() {
    contentLoading = true;
    content = "";
    editing = false;
    try {
      const res = await fetch(`/files/${file.id}`);
      content = await res.text();
    } catch {
      content = "(could not load content)";
    } finally {
      contentLoading = false;
    }
  }

  // ── Preview: inline editor ─────────────────────────────────────────────────

  function startEdit() {
    editorContent = content;
    editing = true;
  }

  function cancelEdit() {
    editing = false;
    editorContent = "";
  }

  async function saveEdit() {
    toast(`Save "${shortText(file.file_name)}"?`, {
      action: {
        label: "Confirm",
        onClick: async () => {
          saving = true;
          try {
            const blob = new Blob([editorContent], { type: "text/plain" });
            const form = new FormData();
            form.append("file", blob, file.file_name);
            const result: UpdateResponse = await api.updateFile(file.id, form);
            file = result.file;
            content = editorContent;
            editing = false;
            versions = []; // invalidate version cache
            dispatch("tagsUpdated", file);
            toast.success(
              `Saved "${shortText(file.file_name)} v${file.version}"`,
            );
          } catch (e: unknown) {
            alert((e as Error).message);
          } finally {
            saving = false;
          }
        },
      },
    });
  }

  function handleUpdated(result: UpdateResponse) {
    file = result.file;
    showUpdateModal = false;
    versions = [];
    loadContent();
    dispatch("tagsUpdated", file);
  }

  function copyContent() {
    const text = editing ? editorContent : content;
    if (navigator.clipboard?.writeText) {
      navigator.clipboard.writeText(text).catch(() => copyFallback(text));
    } else {
      copyFallback(text);
    }
    copied = true;
    setTimeout(() => (copied = false), 1800);
  }

  function copyFallback(text: string) {
    const el = document.createElement("textarea");
    el.value = text;
    el.style.position = "fixed";
    el.style.opacity = "0";
    document.body.appendChild(el);
    el.select();
    try { document.execCommand("copy"); } catch { /* ignore */ }
    document.body.removeChild(el);
  }

  // ── Versions ───────────────────────────────────────────────────────────────

  async function loadVersions() {
    if (versions.length) return;
    versionsLoading = true;
    try {
      versions = await api.listVersions(file.id);
    } catch {
      versions = [];
    } finally {
      versionsLoading = false;
    }
  }

  async function handleRollback(v: number) {
    if (!confirm(`Rollback to version ${v}?`)) return;
    try {
      const updated = await api.rollback(file.id, v);
      dispatch("tagsUpdated", updated);
      file = updated;
      versions = [];
      await loadContent();
      await loadVersions();
    } catch (e: unknown) {
      alert((e as Error).message);
    }
  }

  // ── Meta ───────────────────────────────────────────────────────────────────

  function shortText(fileName: string, maxLength: number = 30) {
    return fileName.length < maxLength
      ? fileName
      : fileName.slice(0, maxLength) + "...";
  }

  async function deleteFile() {
    toast(`Delete "${shortText(file.file_name)}"?`, {
      action: {
        label: "Confirm",
        onClick: async () => {
          deleting = true;
          try {
            let file_name = shortText(file.file_name, 40);
            await api.deleteFile(file.id);
            dispatch("deleted", file.id);
            dispatch("close");
            toast.success(`Deleted "${file_name}"`);
          } catch (e: unknown) {
            toast.error((e as Error).message);
          } finally {
            deleting = false;
          }
        },
      },
    });
  }

  async function addTag() {
    const tag = newTag.trim().toLowerCase();
    if (!tag) return;
    try {
      await api.addTags(file.id, [tag]);
      const updated = await api.getFileMeta(file.id);
      dispatch("tagsUpdated", updated);
      file = updated;
      newTag = "";
      toast.success(`Added tag "${tag}"`);
    } catch (e: unknown) {
      toast.error((e as Error).message);
    }
  }

  async function removeTag(tag: string) {
    try {
      await api.removeTag(file.id, tag);
      const updated = await api.getFileMeta(file.id);
      dispatch("tagsUpdated", updated);
      file = updated;
      toast.success(`Removed tag "${tag}"`);
    } catch (e: unknown) {
      toast.error((e as Error).message);
    }
  }

  function startEditDescription() {
    descriptionDraft = file.description ?? "";
    editingDescription = true;
  }
  function cancelEditDescription() {
    editingDescription = false;
  }

  async function saveDescription() {
    savingDescription = true;
    try {
      const updated = await api.editMeta(file.id, {
        description: descriptionDraft,
      });
      file = { ...file, description: updated.description };
      dispatch("tagsUpdated", file);
      editingDescription = false;
      toast.success(`Saved new description`);
    } catch (e: unknown) {
      toast.error((e as Error).message);
    } finally {
      savingDescription = false;
    }
  }

  // ── Status → Diff cross-tab navigation ────────────────────────────────────

  function handleDiffNode(node: string) {
    diffPreselectedNode = node;
    tab = 'diff';
  }

  // ── Misc ───────────────────────────────────────────────────────────────────

  function onKey(e: KeyboardEvent) {
    if (e.key === "Escape" && !editing && !showDeliver && !showUpdateModal)
      dispatch("close");
  }

  function onTabChange(t: Tab) {
    tab = t;
    if (t === "versions") loadVersions();
    if (t !== "diff") diffPreselectedNode = '';
  }

  $: file.id, loadContent();
</script>

<svelte:head>
  {@html github}
</svelte:head>

<svelte:window on:keydown={onKey} />

<div
  class="backdrop"
  on:click={() => dispatch("close")}
  on:keydown={(e) => e.key === "Escape" && dispatch("close")}
  role="presentation"
>
  <div class="modal" on:click|stopPropagation role="dialog" aria-modal="true">
    <!-- ── Header ───────────────────────────────────────────────────────── -->
    <div class="header">
      <div class="header-left">
        <span class="fname">{file.file_name}</span>
        <span class="lang" style="--c:{langColor(file.language)}"
          >{file.language || "text"}</span
        >
        {#if file.version > 1}
          <span class="version-badge">v{file.version}</span>
        {/if}
      </div>
      <div class="header-right">
        <button class="icon-btn" title="Download">
          <a href="/files/{file.id}" download={file.file_name} class="dl-link">
            <svg viewBox="0 0 16 16" fill="none" width="13" height="13">
              <path
                d="M8 2v8M5 7l3 3 3-3"
                stroke="currentColor"
                stroke-width="1.4"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
              <path
                d="M2 12h12"
                stroke="currentColor"
                stroke-width="1.4"
                stroke-linecap="round"
              />
            </svg>
          </a>
        </button>
        <button
          class="icon-btn"
          title="Deliver"
          on:click={() => (showDeliver = true)}
        >
          <svg viewBox="0 0 16 16" fill="none" width="13" height="13">
            <path
              d="M2 8h10M8 4l4 4-4 4"
              stroke="currentColor"
              stroke-width="1.4"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </button>
        <button
          class="icon-btn danger"
          title="Delete"
          disabled={deleting}
          on:click={deleteFile}
        >
          <svg viewBox="0 0 16 16" fill="none" width="13" height="13">
            <path
              d="M3 4h10M6 4V3h4v1M5 4v8a1 1 0 001 1h4a1 1 0 001-1V4"
              stroke="currentColor"
              stroke-width="1.3"
              stroke-linecap="round"
            />
          </svg>
        </button>
        <div class="divider-v"></div>
        <button class="close-btn" on:click={() => dispatch("close")}>
          <svg viewBox="0 0 16 16" fill="none" width="14" height="14">
            <path
              d="M3 3l10 10M13 3L3 13"
              stroke="currentColor"
              stroke-width="1.5"
              stroke-linecap="round"
            />
          </svg>
        </button>
      </div>
    </div>

    <!-- ── Tabs ─────────────────────────────────────────────────────────── -->
    <div class="tab-bar">
      {#each (["preview", "meta", "versions", "status", "diff"] as Tab[]) as t}
        <button
          class="tab"
          class:active={tab === t}
          on:click={() => onTabChange(t)}
        >
          {#if t === "status"}
            <svg viewBox="0 0 10 10" fill="none" width="9" height="9" style="opacity:0.6">
              <circle cx="5" cy="5" r="4" stroke="currentColor" stroke-width="1.1"/>
              <circle cx="5" cy="5" r="1.5" fill="currentColor"/>
            </svg>
          {:else if t === "diff"}
            <svg viewBox="0 0 10 10" fill="none" width="9" height="9" style="opacity:0.6">
              <path d="M1 3h4M1 7h6M7 1.5v7" stroke="currentColor" stroke-width="1.1" stroke-linecap="round"/>
            </svg>
          {/if}
          {t.charAt(0).toUpperCase() + t.slice(1)}
          {#if t === "versions" && versions.length > 0}
            <span class="tab-count">{versions.length}</span>
          {/if}
        </button>
      {/each}
      <div class="tab-spacer"></div>
      <span class="path-crumb">{file.path}</span>
    </div>

    <!-- ── Body ─────────────────────────────────────────────────────────── -->
    <div class="body">
      <!-- PREVIEW TAB -->
      {#if tab === "preview"}
        <div class="preview-wrap">
          <div class="preview-toolbar">
            <span class="toolbar-info">{formatBytes(file.size)}</span>
            <div class="toolbar-actions">
              {#if !editing}
                <button class="tool-btn" on:click={copyContent}>
                  {#if copied}
                    <svg viewBox="0 0 12 12" fill="none" width="11" height="11">
                      <path
                        d="M2 6l2.5 2.5L10 3"
                        stroke="#16a34a"
                        stroke-width="1.5"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                      />
                    </svg>
                    Copied
                  {:else}
                    <svg viewBox="0 0 14 14" fill="none" width="11" height="11">
                      <rect
                        x="4"
                        y="4"
                        width="7"
                        height="8"
                        rx="1.2"
                        stroke="currentColor"
                        stroke-width="1.2"
                      />
                      <path
                        d="M2.5 9.5V2.5a1 1 0 011-1h5.5"
                        stroke="currentColor"
                        stroke-width="1.2"
                        stroke-linecap="round"
                      />
                    </svg>
                    Copy
                  {/if}
                </button>
                <div class="toolbar-sep"></div>
                <button class="tool-btn" on:click={startEdit}>
                  <svg viewBox="0 0 14 14" fill="none" width="11" height="11">
                    <path
                      d="M9.5 2.5l2 2L4 12H2v-2L9.5 2.5z"
                      stroke="currentColor"
                      stroke-width="1.2"
                      stroke-linejoin="round"
                    />
                  </svg>
                  Edit
                </button>
                <button
                  class="tool-btn"
                  on:click={() => (showUpdateModal = true)}
                >
                  <svg viewBox="0 0 14 14" fill="none" width="11" height="11">
                    <path
                      d="M7 1v8M4 4l3-3 3 3"
                      stroke="currentColor"
                      stroke-width="1.3"
                      stroke-linecap="round"
                      stroke-linejoin="round"
                    />
                    <path
                      d="M1 11h12"
                      stroke="currentColor"
                      stroke-width="1.3"
                      stroke-linecap="round"
                    />
                  </svg>
                  Replace
                </button>
              {:else}
                <span class="editing-badge">Editing</span>
                <button
                  class="tool-btn save"
                  on:click={saveEdit}
                  disabled={saving}
                >
                  {saving ? "Saving…" : "Save"}
                </button>
                <button class="tool-btn" on:click={cancelEdit}>Cancel</button>
              {/if}
            </div>
          </div>

          <div class="code-body">
            {#if contentLoading}
              <div class="state-msg">Loading…</div>
            {:else if editing}
              <textarea
                class="editor-tab"
                bind:value={editorContent}
                spellcheck="false"
                autocomplete="off"
              ></textarea>
            {:else}
              <HighlightAuto code={content} let:highlighted>
                <LineNumbers
                  {highlighted}
                  --line-number-color="rgba(0, 0, 0, 0.3)"
                  --border-color="rgba(0, 0, 0, 0.3)"
                  --padding-left="0.75em"
                  --padding-right="0.75em"
                  --highlighted-background="rgba(0, 0, 0, 0)"
                  wrapLines
                />
              </HighlightAuto>
            {/if}
          </div>
        </div>

        <!-- META TAB -->
      {:else if tab === "meta"}
        <div class="meta-pane">
          <div class="meta-grid">
            <span class="ml">File ID</span>
            <span class="mv mono">{file.id}</span>

            <span class="ml">File name</span>
            <span class="mv mono">{file.file_name}</span>

            <span class="ml">Path</span>
            <span class="mv mono">{file.path}</span>

            {#if file.local_path}
              <span class="ml">Local path</span>
              <span class="mv mono sha">{file.local_path}</span>
            {/if}

            <span class="ml">Size</span>
            <span class="mv">{formatBytes(file.size)}</span>

            <span class="ml">Language</span>
            <span class="mv"
              ><span class="lang" style="--c:{langColor(file.language)}"
                >{file.language || "—"}</span
              ></span
            >

            <span class="ml">Version</span>
            <span class="mv mono">v{file.version}</span>

            <span class="ml">Uploaded by</span>
            <span class="mv mono">{file.uploaded_by}</span>

            <span class="ml">Created</span>
            <span class="mv">{formatDate(file.created_at)}</span>

            <span class="ml">Updated</span>
            <span class="mv">{formatDate(file.updated_at)}</span>

            <span class="ml">Description</span>
            <div class="mv desc-cell">
              {#if editingDescription}
                <div class="desc-editor">
                  <input
                    class="desc-input"
                    bind:value={descriptionDraft}
                    placeholder="Add a description…"
                    on:keydown={(e) => {
                      if (e.key === "Enter") saveDescription();
                      if (e.key === "Escape") cancelEditDescription();
                    }}
                  />
                  <div class="desc-actions">
                    <button
                      class="desc-save"
                      on:click={saveDescription}
                      disabled={savingDescription}
                    >
                      {savingDescription ? "…" : "Save"}
                    </button>
                    <button class="desc-cancel" on:click={cancelEditDescription}
                      >Cancel</button
                    >
                  </div>
                </div>
              {:else}
                <button class="desc-display" on:click={startEditDescription}>
                  {#if file.description}
                    <span class="desc-text">{file.description}</span>
                  {:else}
                    <span class="desc-empty">Add a description…</span>
                  {/if}
                  <svg
                    class="desc-edit-icon"
                    viewBox="0 0 14 14"
                    fill="none"
                    width="11"
                    height="11"
                  >
                    <path
                      d="M9.5 2.5l2 2L4 12H2v-2L9.5 2.5z"
                      stroke="currentColor"
                      stroke-width="1.2"
                      stroke-linejoin="round"
                    />
                  </svg>
                </button>
              {/if}
            </div>

            <span class="ml">SHA-256</span>
            <span class="mv mono sha">{file.sha256}</span>
          </div>

          <div class="tags-section">
            <span class="section-label">Tags</span>
            <div class="tags-row">
              {#each file.tags ?? [] as tag}
                <span class="tag-pill">
                  #{tag}
                  <button class="trm" on:click={() => removeTag(tag)}>×</button>
                </span>
              {/each}
              <div class="tag-add">
                <input
                  placeholder="add tag…"
                  bind:value={newTag}
                  on:keydown={(e) => e.key === "Enter" && addTag()}
                />
                <button on:click={addTag} disabled={!newTag.trim()}>+</button>
              </div>
            </div>
          </div>
        </div>

        <!-- VERSIONS TAB -->
      {:else if tab === "versions"}
        <div class="versions-pane">
          {#if versionsLoading}
            <div class="state-msg">Loading versions…</div>
          {:else if versions.length === 0}
            <div class="state-msg muted">No version history found.</div>
          {:else}
            <table class="versions-table">
              <thead>
                <tr>
                  <th style="width:28px"></th>
                  <th>Ver</th>
                  <th>Size</th>
                  <th>By</th>
                  <th>Message</th>
                  <th>Date</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {#each versions as v (v.id)}
                  <VersionRow
                    version={v}
                    currentVersion={file.version}
                    fileId={file.id}
                    on:rollback={(e) => handleRollback(e.detail)}
                  />
                {/each}
              </tbody>
            </table>
          {/if}
        </div>

        <!-- STATUS TAB -->
      {:else if tab === "status"}
        <FleetStatusTab
          {file}
          on:diffNode={(e) => handleDiffNode(e.detail)}
        />

        <!-- DIFF TAB -->
      {:else if tab === "diff"}
        <DiffTab {file} preselectedNode={diffPreselectedNode} />

      {/if}
    </div>
  </div>
</div>

{#if showDeliver}
  <SendModal {file} on:close={() => (showDeliver = false)} />
{/if}

{#if showUpdateModal}
  <UpdateFileModal
    {file}
    on:close={() => (showUpdateModal = false)}
    on:updated={(e) => handleUpdated(e.detail)}
  />
{/if}

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.35);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 150;
    backdrop-filter: blur(3px);
  }
  .modal {
    background: white;
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    width: min(920px, 96vw);
    height: 86vh;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    box-shadow:
      0 20px 60px rgba(0, 0, 0, 0.16),
      0 4px 16px rgba(0, 0, 0, 0.08);
  }

  /* Header */
  .header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 18px;
    border-bottom: 1px solid var(--border);
    flex-shrink: 0;
    gap: 12px;
  }
  .header-left {
    display: flex;
    align-items: center;
    gap: 8px;
    overflow: hidden;
  }
  .fname {
    font-family: var(--mono);
    font-size: 14px;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .lang {
    flex-shrink: 0;
    font-family: var(--mono);
    font-size: 10.5px;
    font-weight: 500;
    padding: 2px 8px;
    border-radius: 20px;
    background: color-mix(in srgb, var(--c) 12%, transparent);
    color: color-mix(in srgb, var(--c) 80%, #000);
    border: 1px solid color-mix(in srgb, var(--c) 22%, transparent);
  }
  .version-badge {
    font-size: 10px;
    font-family: var(--mono);
    background: var(--bg-3);
    border: 1px solid var(--border);
    padding: 1px 6px;
    border-radius: 3px;
    color: var(--text-3);
    flex-shrink: 0;
  }
  .header-right {
    display: flex;
    align-items: center;
    gap: 4px;
    flex-shrink: 0;
  }
  .divider-v {
    width: 1px;
    height: 18px;
    background: var(--border);
    margin: 0 4px;
  }
  .icon-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    height: 30px;
    background: none;
    border: none;
    border-radius: var(--radius);
    color: var(--text-3);
    cursor: pointer;
    transition:
      background 0.1s,
      color 0.1s;
  }
  .icon-btn:hover {
    background: var(--bg-2);
    color: var(--text);
  }
  .icon-btn.danger:hover {
    background: #fef2f2;
    color: #dc2626;
  }
  .icon-btn:disabled {
    opacity: 0.4;
    pointer-events: none;
  }
  .dl-link {
    display: flex;
    align-items: center;
    justify-content: center;
    color: inherit;
    text-decoration: none;
    width: 100%;
    height: 100%;
  }
  .close-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    height: 30px;
    background: none;
    border: none;
    border-radius: var(--radius);
    color: var(--text-3);
    cursor: pointer;
    transition:
      background 0.1s,
      color 0.1s;
  }
  .close-btn:hover {
    background: var(--bg-2);
    color: var(--text);
  }

  /* Tabs */
  .tab-bar {
    display: flex;
    align-items: center;
    border-bottom: 1px solid var(--border);
    padding: 0 18px;
    background: var(--bg);
    flex-shrink: 0;
  }
  .tab {
    display: flex;
    align-items: center;
    gap: 5px;
    padding: 10px 16px;
    background: none;
    border: none;
    border-bottom: 2px solid transparent;
    font-size: 12.5px;
    font-weight: 500;
    color: var(--text-3);
    cursor: pointer;
    margin-bottom: -1px;
    transition:
      color 0.12s,
      border-color 0.12s;
  }
  .tab:hover {
    color: var(--text-2);
  }
  .tab.active {
    color: var(--text);
    border-bottom-color: var(--text);
  }
  .tab-count {
    font-size: 10px;
    font-family: var(--mono);
    background: var(--bg-3);
    border: 1px solid var(--border);
    padding: 0px 5px;
    border-radius: 8px;
    color: var(--text-3);
  }
  .tab-spacer {
    flex: 1;
  }
  .path-crumb {
    font-family: var(--mono);
    font-size: 11px;
    color: var(--text-3);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 300px;
  }

  /* Body */
  .body {
    flex: 1;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    min-height: 0;
  }

  /* Preview */
  .preview-wrap {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }
  .preview-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 7px 18px;
    border-bottom: 1px solid var(--border);
    background: var(--bg);
    flex-shrink: 0;
    gap: 12px;
  }
  .toolbar-info {
    font-family: var(--mono);
    font-size: 11px;
    color: var(--text-3);
  }
  .toolbar-actions {
    display: flex;
    align-items: center;
    gap: 4px;
  }
  .toolbar-sep {
    width: 1px;
    height: 16px;
    background: var(--border);
    margin: 0 2px;
  }
  .editing-badge {
    font-size: 10.5px;
    font-family: var(--mono);
    background: #fffbeb;
    border: 1px solid #fde68a;
    color: #92400e;
    padding: 1px 7px;
    border-radius: 3px;
    margin-right: 4px;
  }
  .tool-btn {
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
  .tool-btn:hover {
    background: var(--bg-2);
    border-color: var(--border-2);
  }
  .tool-btn.save {
    background: var(--text);
    color: white;
    border-color: var(--text);
  }
  .tool-btn.save:hover:not(:disabled) {
    background: #3d3c38;
  }
  .tool-btn:disabled {
    opacity: 0.4;
    pointer-events: none;
  }

  .code-body {
    flex: 1;
    overflow: auto;
    background: var(--bg);
    font-size: 12px;
    font-family: var(--mono);
  }

  .editor-tab {
    width: 100%;
    height: 100%;
    min-height: 300px;
    border: none;
    outline: none;
    resize: none;
    font-family: var(--mono);
    font-size: 12px;
    line-height: 1.65;
    color: var(--text);
    background: white;
    tab-size: 2;
    padding: 1em
  }

  /* Meta */
  .meta-pane {
    flex: 1;
    overflow-y: auto;
    padding: 20px;
    display: flex;
    flex-direction: column;
    gap: 20px;
  }
  .meta-grid {
    display: grid;
    grid-template-columns: 110px 1fr;
    gap: 8px 16px;
    align-items: start;
  }
  .ml {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-3);
    padding-top: 1px;
    white-space: nowrap;
  }
  .mv {
    font-size: 13px;
    color: var(--text);
  }
  .mono {
    font-family: var(--mono);
    font-size: 12.5px;
  }
  .sha {
    font-size: 11px;
    word-break: break-all;
    color: var(--text-3);
  }

  /* Description editor */
  .desc-cell {
    min-width: 0;
  }
  .desc-display {
    display: flex;
    align-items: flex-start;
    gap: 6px;
    background: none;
    border: 1px solid transparent;
    border-radius: var(--radius);
    padding: 3px 6px;
    margin: -3px -6px;
    cursor: pointer;
    text-align: left;
    width: 100%;
    transition:
      border-color 0.1s,
      background 0.1s;
    color: var(--text);
  }
  .desc-display:hover {
    border-color: var(--border);
    background: var(--bg-2);
  }
  .desc-display:hover .desc-edit-icon {
    opacity: 1;
  }
  .desc-text {
    font-size: 13px;
    flex: 1;
  }
  .desc-empty {
    font-size: 13px;
    color: var(--text-3);
    font-style: italic;
    flex: 1;
  }
  .desc-edit-icon {
    flex-shrink: 0;
    color: var(--text-3);
    opacity: 0;
    margin-top: 2px;
    transition: opacity 0.1s;
  }
  .desc-editor {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .desc-input {
    width: 100%;
    height: 32px;
    padding: 0 9px;
    border: 1px solid var(--border-2);
    border-radius: var(--radius);
    font-size: 13px;
    background: white;
    outline: none;
    transition: border-color 0.1s;
  }
  .desc-input:focus {
    border-color: var(--text);
  }
  .desc-actions {
    display: flex;
    gap: 5px;
  }
  .desc-save {
    height: 26px;
    padding: 0 12px;
    background: var(--text);
    color: white;
    border: none;
    border-radius: var(--radius);
    font-size: 11.5px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.1s;
  }
  .desc-save:hover:not(:disabled) {
    background: #3d3c38;
  }
  .desc-save:disabled {
    opacity: 0.5;
    pointer-events: none;
  }
  .desc-cancel {
    height: 26px;
    padding: 0 10px;
    background: none;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    font-size: 11.5px;
    color: var(--text-2);
    cursor: pointer;
    transition: background 0.1s;
  }
  .desc-cancel:hover {
    background: var(--bg-2);
  }

  /* Tags */
  .tags-section {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .section-label {
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--text-3);
  }
  .tags-row {
    display: flex;
    flex-wrap: wrap;
    gap: 5px;
    align-items: center;
  }
  .tag-pill {
    display: flex;
    align-items: center;
    gap: 3px;
    padding: 2px 8px;
    background: #eff6ff;
    border: 1px solid #bfdbfe;
    border-radius: 4px;
    font-size: 11px;
    font-family: var(--mono);
    color: #2563eb;
  }
  .trm {
    background: none;
    border: none;
    color: #93c5fd;
    font-size: 14px;
    line-height: 1;
    padding: 0 1px;
    cursor: pointer;
  }
  .trm:hover {
    color: #2563eb;
  }
  .tag-add {
    display: flex;
    border: 1px solid var(--border);
    border-radius: 4px;
    overflow: hidden;
  }
  .tag-add input {
    border: none;
    outline: none;
    padding: 3px 8px;
    font-size: 11px;
    font-family: var(--mono);
    width: 90px;
    background: var(--bg-2);
  }
  .tag-add button {
    border: none;
    background: var(--bg-3);
    border-left: 1px solid var(--border);
    padding: 2px 8px;
    font-size: 14px;
    color: var(--text-2);
    cursor: pointer;
    line-height: 1.3;
  }
  .tag-add button:hover:not(:disabled) {
    background: var(--text);
    color: white;
  }
  .tag-add button:disabled {
    opacity: 0.4;
  }

  /* Versions */
  .versions-pane {
    flex: 1;
    overflow-y: auto;
  }
  .state-msg {
    padding: 40px;
    text-align: center;
    font-size: 13px;
    color: var(--text-3);
  }
  .state-msg.muted {
    font-style: italic;
  }
  .versions-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 12.5px;
  }
  .versions-table th {
    text-align: left;
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-3);
    padding: 8px 10px;
    border-bottom: 1px solid var(--border);
    position: sticky;
    top: 0;
    background: white;
    z-index: 1;
  }
</style>