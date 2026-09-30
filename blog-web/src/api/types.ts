export type Identity = 'user' | 'admin'
export type ArticleStatus = 0 | 1 | 2
export type CommentStatus = 0 | 1 | 2

export interface ArticleImage { url: string; width: number; height: number; content_type?: string }

export interface Profile {
  id: number
  avatar: string
  status: number
  nick_name?: string
  admin_name?: string
  role?: 0 | 1
  signature?: string
  remark?: string
  email?: string
}

export interface LoginResult { token: string; expire: string; data: Profile }
export interface Page<T> { total: number; pages: number; list: T[]; pageNum: number; pageSize: number }
export interface Category { id: number; name: string; sort: number; status: 0 | 1 }
export interface Author { type: Identity; name: string; avatar: string; signature: string }

export interface PublicArticle {
  id: number
  title: string
  cover: string
  summary: string
  category_id: number
  is_top: number
  publish_at: string | null
  created_at?: string
  create_at?: string
  author: Author | null
}

/** 所有更新都发送完整可编辑字段，不能只发 status；不含作者和统计字段。 */
export interface ArticleInput {
  title: string
  cover: string
  summary: string
  content: string
  category_id: number
  tag_ids: string
  status: ArticleStatus
  sort: number
  is_top: number
  comment_status: number
  publish_at: string | null
}

export interface ManagedArticle extends Omit<ArticleInput, 'content'> {
  // 管理列表省略正文；详情的历史空正文也可能被后端 omitempty 省略。
  content?: string
  id: number
  user_id: number
  admin_id: number
  create_at: string
  update_at: string
}

export interface ArticleDetail extends ManagedArticle { content: string; author: Author | null }

export interface Comment {
  id: number
  article_id: number
  parent_id: number
  nickname: string
  avatar: string
  content: string
  likes: number
  create_at: string
}

export interface ManagedComment extends Comment { status: CommentStatus; update_at: string }
export interface CommentReview { id: number; expected_content: string; expected_status: CommentStatus; status: 1 | 2 }

export type Query = Record<string, string | number | undefined | null>
