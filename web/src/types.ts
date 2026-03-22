
export interface Blob {
    sha256: string;
    size: number;
    ref_count: number;
    created_at: string;
  }
  
  export interface File {
    id: string;
    path: string;
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
    hostname: string;
    ip: string;
    online: boolean;
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
  
  export interface SendResponse {
    results: SendResult[];
  }
  
  // Virtual directory — resolved from path prefix on the server
  export interface Directory {
    id: string;         // synthetic ID / path prefix
    name: string;       // last path segment
    prefix: string;     // full path prefix, e.g. "scripts/deploy/"
    file_count: number;
    size: number;
    tags?: string[];
    files?: File[];     // populated when GET /dirs/{dir}
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
  // UI helpers
  // ---------------------------------------------------------------------------
  
  export type MainTab = 'files' | 'directories';
  export type PreviewTab = 'preview' | 'meta' | 'versions';