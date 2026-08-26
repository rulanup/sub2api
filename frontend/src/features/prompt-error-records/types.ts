export interface PromptErrorRecord {
  id: number
  request_id: string
  user_id?: number
  username_snapshot: string
  user_email_snapshot: string
  api_key_id?: number
  api_key_name_snapshot: string
  group_id?: number
  group_name: string
  provider: string
  endpoint: string
  protocol: string
  model: string
  prompt_hash: string
  full_prompt: string
  prompt_length: number
  message_count: number
  error_status: number
  error_body: string
  error_type: string
  created_at: string
}

export interface PromptErrorFilters {
  keyword: string
  model: string
  error_status: string
  group_id: string
  user_id: string
  api_key_id: string
  request_id: string
  prompt_hash: string
  start_at: string
  end_at: string
}

export interface PromptErrorPage {
  items: PromptErrorRecord[]
  total: number
  page: number
  page_size: number
  pages: number
}

export interface PromptErrorDeletePreview {
  matched_count: number
  filter_summary: Record<string, unknown>
  snapshot_max_id: number
  filter_hash: string
  confirmation_token: string
  expires_at: string
}

export interface PromptErrorDeleteResult {
  deleted: number
}
