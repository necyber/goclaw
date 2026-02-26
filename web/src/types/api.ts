export type WorkflowState =
  | "pending"
  | "scheduled"
  | "running"
  | "completed"
  | "failed"
  | "cancelled";

export interface WorkflowSummary {
  id: string;
  name: string;
  status: WorkflowState;
  created_at: string;
  completed_at?: string | null;
  task_count: number;
}

export interface WorkflowTask {
  id: string;
  name: string;
  status: WorkflowState;
  depends_on?: string[];
  started_at?: string | null;
  completed_at?: string | null;
  error?: string;
  result?: unknown;
}

export interface WorkflowDetail {
  id: string;
  name: string;
  status: WorkflowState;
  created_at: string;
  started_at?: string | null;
  completed_at?: string | null;
  tasks: WorkflowTask[];
  metadata?: Record<string, string>;
  error?: string;
}

export interface WorkflowListResponse {
  workflows: WorkflowSummary[];
  total: number;
  limit: number;
  offset: number;
}

export interface SubmitTaskDefinition {
  id: string;
  name: string;
  type: "http" | "script" | "function";
  depends_on?: string[];
  config?: Record<string, unknown>;
  timeout?: number;
  retries?: number;
}

export interface SubmitWorkflowRequest {
  name: string;
  description?: string;
  tasks: SubmitTaskDefinition[];
  metadata?: Record<string, string>;
}

export interface SubmitWorkflowResponse {
  id: string;
  name: string;
  status: WorkflowState;
  message?: string;
}

export interface TaskResultResponse {
  workflow_id: string;
  task_id: string;
  status: WorkflowState;
  result?: unknown;
  error?: string;
  completed_at?: string | null;
}

export interface EngineStatus {
  state: "idle" | "running" | "stopped" | "error" | "unknown";
  uptime?: string;
  version?: string;
  active_workflows?: number;
  goroutines?: number;
  memory_bytes?: number;
}

export interface LaneStats {
  name: string;
  queue_depth: number;
  workers: number;
  throughput_per_sec: number;
  error_rate: number;
}

export interface AdminDebugInfo {
  generated_at: string;
  goroutines?: string;
  heap_summary?: Record<string, unknown>;
  system?: Record<string, unknown>;
}

export interface PrometheusSample {
  metric: string;
  labels: Record<string, string>;
  value: number;
}
