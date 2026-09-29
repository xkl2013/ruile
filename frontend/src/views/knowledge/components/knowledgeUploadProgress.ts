export type KnowledgeUploadProgressStatus =
  | 'uploading'
  | 'paused'
  | 'completed'
  | 'failed'
  | 'cancelled'

export interface KnowledgeUploadProgressState {
  title: string
  total: number
  processed: number
  failed: number
  currentProgress: number
  currentName: string
  status: KnowledgeUploadProgressStatus
}
