import { request } from './http'

export interface AISource { path: string; line: number; updated_at: string; excerpt: string }
export interface AIMessage { id: string; role: 'user' | 'assistant'; content: string; status: 'pending' | 'completed' | 'failed'; create_at: string; sources: AISource[] }
export type AIContextMode = 'general' | 'project'
export interface AIConversation { id: string; context_mode: AIContextMode; origin_import_id?: string; title: string; update_at: string; messages?: AIMessage[] }
export interface AIImportedThread { id: string; title: string; updated_at: string; total_messages: number; next_before: number; messages?: AIMessage[] }
export interface AISyncImport { id: string; title: string; workspace: 'yia-blog' | 'yia-project'; updated_at: string; messages: Pick<AIMessage, 'id' | 'role' | 'content' | 'create_at'>[] }
export interface AIOverview { model: string; memory: string; project_memory?: string; daily_limit: number; used: number; knowledge_documents: number; private_knowledge_enabled: boolean; conversations: AIConversation[] }
export interface AICodeJob { id: string; prompt: string; context_mode: AIContextMode; status: string; summary: string; diff?: string; checks?: string; branch: string; url?: string; tests_passed: boolean; review_digest?: string; created_at: string; updated_at: string }
export interface AIWorkspace { enabled: boolean; daily_limit: number; jobs: AICodeJob[] }

export const aiApi = {
  overview: (signal?: AbortSignal) => request<AIOverview>('/admin/ai/overview', { identity: 'admin', signal }),
  conversation: (id: string, signal?: AbortSignal) => request<AIConversation>('/admin/ai/conversation', { identity: 'admin', query: { id }, signal }),
  send: (conversation_id: string, request_id: string, content: string, signal?: AbortSignal, context_mode: AIContextMode = 'general', import_id?: string) => request<AIConversation>('/admin/ai/send', { identity: 'admin', method: 'POST', body: { conversation_id, request_id, content, context_mode, import_id }, signal, preferServerMessage: true }),
  imports: (signal?: AbortSignal) => request<AIImportedThread[]>('/admin/ai/imports', { identity: 'admin', signal }),
  imported: (id: string, before = 0, signal?: AbortSignal) => request<AIImportedThread>('/admin/ai/import', { identity: 'admin', query: { id, before }, signal }),
  syncImport: (thread: AISyncImport, signal?: AbortSignal) => request<null>('/admin/ai/import', { identity: 'admin', method: 'POST', body: thread, signal, preferServerMessage: true }),
  memory: (memory: string, signal?: AbortSignal, context_mode: AIContextMode = 'general') => request<null>('/admin/ai/memory', { identity: 'admin', method: 'POST', body: { memory, context_mode }, signal, preferServerMessage: true }),
  remove: (id: string) => request<null>('/admin/ai/delete', { identity: 'admin', method: 'POST', body: { id } }),
  workspace: (signal?: AbortSignal) => request<AIWorkspace>('/admin/ai/workspace', { identity: 'admin', signal }),
  createJob: (id: string, prompt: string, context_mode: AIContextMode) => request<AICodeJob>('/admin/ai/job', { identity: 'admin', method: 'POST', body: { id, prompt, context_mode }, preferServerMessage: true }),
  publishJob: (id: string, review_digest: string) => request<null>('/admin/ai/job/publish', { identity: 'admin', method: 'POST', body: { id, review_digest }, preferServerMessage: true }),
}
