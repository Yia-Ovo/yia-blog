import { lazy } from 'react'
import { createBrowserRouter } from 'react-router-dom'
import App from './App'
import RequireSession from './components/RequireSession'
import StatePanel from './components/StatePanel'

const HomeView = lazy(() => import('./views/HomeView'))
const ArticleView = lazy(() => import('./views/ArticleView'))
const AuthView = lazy(() => import('./views/AuthView'))
const ArticlesView = lazy(() => import('./views/ArticlesView'))
const EditorView = lazy(() => import('./views/EditorView'))
const ModerationView = lazy(() => import('./views/ModerationView'))
const CategoriesView = lazy(() => import('./views/CategoriesView'))
const NotFoundView = lazy(() => import('./views/NotFoundView'))

export const router = createBrowserRouter([{
  element: <App />,
  errorElement: <StatePanel kind="error" title="页面暂时无法打开" message="请重新加载页面后再试。" onRetry={() => window.location.reload()} />,
  children: [
    { path: '/', element: <HomeView />, handle: { title: '文章' } },
    { path: '/articles/:id', element: <ArticleView />, handle: { title: '阅读' } },
    { path: '/login', element: <AuthView key="user-login" identity="user" />, handle: { title: '登录' } },
    { path: '/register', element: <AuthView key="register" identity="user" register />, handle: { title: '注册' } },
    { path: '/admin/login', element: <AuthView key="admin-login" identity="admin" />, handle: { title: '管理登录' } },
    { path: '/me/articles', element: <RequireSession identity="user"><ArticlesView identity="user" /></RequireSession>, handle: { title: '我的文章' } },
    { path: '/write/new', element: <RequireSession identity="user" preserveEditor><EditorView identity="user" /></RequireSession>, handle: { title: '写文章' } },
    { path: '/write/:id', element: <RequireSession identity="user" preserveEditor><EditorView identity="user" /></RequireSession>, handle: { title: '编辑文章' } },
    { path: '/admin/articles', element: <RequireSession identity="admin"><ArticlesView identity="admin" /></RequireSession>, handle: { title: '文章管理' } },
    { path: '/admin/articles/new', element: <RequireSession identity="admin" preserveEditor><EditorView identity="admin" /></RequireSession>, handle: { title: '写文章' } },
    { path: '/admin/articles/:id/edit', element: <RequireSession identity="admin" preserveEditor><EditorView identity="admin" /></RequireSession>, handle: { title: '编辑文章' } },
    { path: '/admin/comments', element: <RequireSession identity="admin"><ModerationView /></RequireSession>, handle: { title: '评论审核' } },
    { path: '/admin/categories', element: <RequireSession identity="admin" superOnly><CategoriesView /></RequireSession>, handle: { title: '分类管理' } },
    { path: '*', element: <NotFoundView />, handle: { title: '页面不存在' } },
  ],
}])
