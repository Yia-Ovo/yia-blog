import { useEffect, useRef, useState } from 'react'
import { articleApi } from '../api'
import { errorMessage } from '../api/http'
import type { ArticleImage, Identity } from '../api/types'
import { useSessions } from './useSessions'
import { getSessions } from '../stores/session'
import { validArticleImage } from '../lib/articleImages'

/** 上传只交付给发起时的会话；换账号、离开编辑器后取消并丢弃旧结果。 */
export function useArticleImageUpload(identity: Identity) {
  const token = useSessions()[identity]?.token
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState('')
  const controller = useRef<AbortController | null>(null)
  const generation = useRef(0)

  useEffect(() => {
    setUploading(false)
    return () => {
      generation.current++
      controller.current?.abort()
      controller.current = null
    }
  }, [identity, token])

  async function upload(files: File[], onUploaded: (image: ArticleImage, file: File) => void) {
    if (!files.length || controller.current) return
    if (!token || getSessions()[identity]?.token !== token) { setError('请先重新登录，再上传图片。'); return }
    if (files.length !== 1) { setError('请一次上传一张图片。'); return }
    const file = files[0]!
    if (!['image/jpeg', 'image/png'].includes(file.type)) { setError('请选择 JPG 或 PNG 图片。'); return }
    if (!file.size || file.size > 5 * 1024 * 1024) { setError('图片不能为空，大小不能超过 5 MB。'); return }
    const current = generation.current
    const abort = new AbortController()
    controller.current = abort
    setError(''); setUploading(true)
    try {
      const result = await articleApi.uploadImage(identity, file, abort.signal)
      if (current !== generation.current || abort.signal.aborted || getSessions()[identity]?.token !== token) return
      if (!validArticleImage(result)) throw new Error('图片响应无效，请重新上传。')
      onUploaded(result, file)
    } catch (cause) {
      if (current === generation.current && !abort.signal.aborted) setError(errorMessage(cause))
    } finally {
      if (controller.current === abort) {
        controller.current = null
        setUploading(false)
      }
    }
  }

  return { uploading, error, upload, isUploading: () => controller.current !== null }
}
