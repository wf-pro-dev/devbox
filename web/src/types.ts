
export interface Blob {
  sha256: string;
  size: number;
  ref_count: number;
  created_at: string;
}

export interface File {
  id: string;
  path: string;
  local_path?: string; // absolute path on the target node where the file lives
  file_name: string;
  description: string;
  language: string;
  size: number;
  sha256: string;
  uploaded_by: string;
  version: number;
  created_at: string;
  updated_at: string;
  tags?: string[];
}

export interface FileTag {
  file_id: string;
  tag_id: number;
}

export interface Tag {
  id: number;
  name: string;
}

export interface Transfer {
  id: number;
  from_host: string;
  to_host: string;
  file_path: string;
  size: number;
  duration_ms: number;
  created_at: string;
}

export interface Version {
  id: number;
  file_id: string;
  version: number;
  sha256: string;
  size: number;
  uploaded_by: string;
  message: string;
  created_at: string;
}

// ---------------------------------------------------------------------------
// API response shapes
// ---------------------------------------------------------------------------

export interface Peer {
  status: any;
  tailkit: Tailkit;
}

export interface Tailkit {
  status: any;
}

export interface HealthResponse {
  status: string;
  service: string;
  caller_host?: string;
  caller_user?: string;
  caller_ip?: string;
}

export interface SendResult {
  target: string;
  success: boolean;
  error?: string;
  local_path?: string;
  written_to?: string;
  dest_machine?: string;
}

export interface SendDirResult {
  [key: string]: SendResult[];
}

// Virtual directory — resolved from path prefix on the server
export interface Directory {
  id: string;
  name: string;
  prefix: string;
  file_count: number;
  size: number;
  tags?: string[];
  files?: File[];
}

export interface UpdateResponse {
  result: string;
  file: File;
}

export interface TreeNode {
  segment: string;
  prefix: string;
  dir: Directory;
  children: TreeNode[];
}

// ---------------------------------------------------------------------------
// Drift / Status types
// ---------------------------------------------------------------------------

/**
 * One entry per peer returned by GET /files/:id/status
 *
 * Actual server response shape:
 *   { "hostname": "...", "status": "MATCH (latest)", "local_path": "/..." }
 *
 * `status` is a raw string from the server:
 *   "MATCH (latest)"    → file matches the vault copy
 *   "NOT FOUND"         → file doesn't exist on the node
 *   "NO TAILKITD FOUND" → tailkitd agent is unreachable on that peer
 *   "DRIFTED"           → file exists but hashes differ
 */
export interface NodeDriftResult {
  hostname: string;
  status: string;
  local_path: string;
}

// ---------------------------------------------------------------------------
// Diff types
// ---------------------------------------------------------------------------

/**
 * Response from:
 *   GET  /files/:id/diff/node?node=hostname[&version=N]
 *   POST /files/:id/diff/local  (multipart field: file)
 */
export interface DiffResult {
  /** Raw unified diff text (--- / +++ / @@ headers + lines) */
  unified: string;
  /** True when both sides are byte-identical (unified will be empty) */
  identical: boolean;
  /** Label for the vault (left) side */
  vault_label: string;
  /** Label for the node/local (right) side */
  node_label: string;
}

/** A parsed hunk from the unified diff string */
export interface ParsedHunk {
  header: string;
  oldStart: number;
  newStart: number;
  lines: ParsedLine[];
}

/** A single parsed line inside a hunk */
export interface ParsedLine {
  type: '+' | '-' | ' ';
  content: string;
  /** Line number in the old (left) file, null for additions */
  oldNo: number | null;
  /** Line number in the new (right) file, null for deletions */
  newNo: number | null;
}

// ---------------------------------------------------------------------------
// UI helpers
// ---------------------------------------------------------------------------

export type MainTab = 'files' | 'directories';
export type PreviewTab = 'preview' | 'meta' | 'versions' | 'status' | 'diff';
