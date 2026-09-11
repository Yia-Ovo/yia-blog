import { request } from './http'
import type { ArticleDetail, ArticleInput, Category, Comment, CommentReview, Identity, LoginResult, ManagedArticle, ManagedComment, Page, Profile, PublicArticle, Query } from './types'

export const authApi = {
  login: (identity: Identity, name: string, password: string) => request<LoginResult>(`/${identity}/login`, {
    method: 'POST', body: { [identity === 'user' ? 'nick_name' : 'admin_name']: name, password },
  }),
  register: (nick_name: string, password: string) => request<null>('/user/register', { method: 'POST', body: { nick_name, password } }),
  profile: (identity: Identity, signal?: AbortSignal) => request<Profile>(`/${identity}/getInfo`, { identity, signal }),
  logout: (identity: Identity) => request<null>(`/${identity}/logout`, { identity, method: 'POST' }),
}

export const articleApi = {
  list: (query: Query, signal?: AbortSignal) => request<Page<PublicArticle>>('/article/listArticle', { query, signal }),
  detail: (id: number, signal?: AbortSignal) => request<ArticleDetail>('/article/getArticleById', { query: { id }, signal }),
  managedList: (identity: Identity, query: Query, signal?: AbortSignal) => request<Page<ManagedArticle>>(`/article/${identity === 'user' ? 'listMyArticles' : 'listAdminArticles'}`, { identity, query, signal }),
  managedDetail: (identity: Identity, id: number, signal?: AbortSignal) => request<ManagedArticle>(`/article/${identity === 'user' ? 'getMyArticle' : 'getAdminArticle'}`, { identity, query: { id }, signal }),
  save: (identity: Identity, input: ArticleInput, id?: number) => request<null>(`/article/${identity === 'admin' ? 'admin' : ''}${id ? (identity === 'admin' ? 'UpdateArticle' : 'updateArticle') : (identity === 'admin' ? 'CreateArticle' : 'createArticle')}`, { identity, method: 'POST', body: id ? { id, ...input } : input }),
  remove: (identity: Identity, id: number) => request<null>(`/article/${identity === 'admin' ? 'adminDeleteArticle' : 'deleteArticle'}`, { identity, method: 'POST', body: { id } }),
}

export const categoryApi = {
  list: (query: Query, signal?: AbortSignal) => request<Page<Category>>('/category/listCategory', { query, signal }),
  async all(signal?: AbortSignal): Promise<Category[]> {
    const result: Category[] = []
    let pageNum = 1
    let pages = 1
    do {
      const page = await categoryApi.list({ pageNum, pageSize: 100, status: 1 }, signal)
      result.push(...page.list)
      pages = page.pages
      pageNum++
    } while (pageNum <= pages)
    return result
  },
  save: (category: Pick<Category, 'name' | 'sort' | 'status'>, id?: number) => request<null>(`/category/${id ? 'updateCategory' : 'createCategory'}`, { identity: 'admin', method: 'POST', body: id ? { id, ...category } : category }),
  remove: (id: number) => request<null>('/category/deleteCategory', { identity: 'admin', method: 'POST', body: { id } }),
}

export const commentApi = {
  list: (articleID: number, page: number, size: number, signal?: AbortSignal) => request<{ list: Comment[]; total: number }>('/comment/listComment', { query: { articleID, page, size, orderBy: 'create_at', orderDir: 'desc' }, signal }),
  create: (article_id: number, content: string, parent_id = 0) => request<null>('/comment/createComment', { identity: 'user', method: 'POST', body: { article_id, content, parent_id } }),
  like: (id: number, cancel = false) => request<null>(`/comment/${cancel ? 'unlikeComment' : 'likeComment'}`, { identity: 'user', method: 'POST', body: { id } }),
  managedList: (query: Query, signal?: AbortSignal) => request<Page<ManagedComment>>('/comment/listAdminComments', { identity: 'admin', query, signal }),
  review: (body: CommentReview) => request<null>('/comment/reviewComment', { identity: 'admin', method: 'POST', body }),
}
