export type AgentRunStatus = 'queued' | 'running' | 'waiting_input' | 'succeeded' | 'failed' | 'cancelled'

export type AgentRunPhase = 'intake' | 'planning' | 'drafting' | 'reviewing' | 'revising' | 'packaging' | 'completed'

export interface ExpertIntakeQuestion {
  id: string
  label: string
  type: 'text' | 'single_choice' | 'multi_choice' | 'date' | 'number'
  required: boolean
  options?: string[]
  description?: string
}

export interface ExpertIntakeInteraction {
  schema_version: 'intake_request_v1'
  questions: ExpertIntakeQuestion[]
}

export interface AgentRunQualityIssue {
  code: string
  severity: 'warning' | 'error' | 'red_line'
  section?: string
  message: string
  instruction: string
}

export interface AgentRunQuality {
  score?: number
  passed?: boolean
  summary?: string
  red_lines?: string[]
  dimensions?: Record<string, number>
  issues?: AgentRunQualityIssue[]
}

export interface AgentRunStep {
  id: string
  run_id: string
  sequence: number
  step_type: AgentRunPhase
  status: 'running' | 'succeeded' | 'failed'
  model_id?: string
  input?: Record<string, unknown>
  output?: Record<string, unknown>
  error?: string
  started_at?: string
  finished_at?: string
  created_at?: string
  updated_at?: string
}

export interface AgentRunCardV1 {
  schema_version: 'service_card_v1'
  title: string
  summary: string
  next_action: string
}

export interface StructuredReportSectionV1 {
  type: 'facts' | 'analysis' | 'risks' | 'missing_information' | 'recommended_actions' | 'talk_track' | 'evidence'
  title: string
  content?: string
  items?: string[]
}

export interface StructuredReportV1 {
  format: 'structured_report_v1'
  title: string
  executive_summary: string
  sections: StructuredReportSectionV1[]
  evidence_refs: string[]
}

export interface AgentRunArtifactResultV1 {
  id?: string
  version_id?: string
  version?: number
  run_id?: string
  kind: 'text' | 'report' | 'html' | 'image' | 'pdf' | 'document' | 'spreadsheet' | 'presentation' | 'audio' | 'video' | 'data'
  role: 'primary' | 'supporting'
  title: string
  format?: string
  mime_type?: string
  original_name?: string
  size_bytes?: number
  resource_ref?: string
  lifecycle?: 'temporary' | 'saved' | 'shared' | 'archived'
  previewable?: boolean
  downloadable?: boolean
  shareable?: boolean
  created_at?: string
  metadata?: Record<string, unknown>
  content?: StructuredReportV1 | Record<string, unknown>
}

export interface AgentRunEvidenceRefV1 {
  source_type: string
  source_id: string
  relation: string
  excerpt?: string
}

export interface AgentRunResultValidation {
  contract: 'agent_result_v1'
  valid: boolean
  errors: string[]
}

export interface AgentRunResult extends Record<string, unknown> {
  schema_version?: 'agent_result_v1'
  decision?: {
    should_create_card: boolean
    confidence: number
    reason: string
  }
  card?: AgentRunCardV1
  artifacts?: AgentRunArtifactResultV1[]
  evidence?: AgentRunEvidenceRefV1[]
  validation?: AgentRunResultValidation
  artifact_type?: string
}

export interface AgentRun {
  id: string
  tenant_id: number
  user_id: string
  service_id?: string
  thread_id?: string
  profile_id?: string
  parent_run_id?: string
  requirement_snapshot_id?: string
  run_type: string
  agent_ref?: string
  agent_version?: string
  trigger_type?: string
  trigger_id?: string
  status: AgentRunStatus
  phase?: AgentRunPhase
  input?: Record<string, unknown>
  interaction?: ExpertIntakeInteraction | Record<string, unknown>
  quality?: AgentRunQuality
  result?: AgentRunResult
  error_code?: string
  error_message?: string
  task_id?: string
  attempt?: number
  queued_at?: string
  resumed_at?: string
  started_at?: string
  finished_at?: string
  created_at?: string
  updated_at?: string
}

export interface AgentRunEvent {
  type: string
  id?: string
  runId?: string
  sequence?: number
  status?: AgentRunStatus
  phase?: AgentRunPhase
  createdAt?: string
  startedAt?: string
  finishedAt?: string
  durationMs?: number
  totalDurationMs?: number
  queueDurationMs?: number
  [key: string]: unknown
}
