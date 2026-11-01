import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from 'react-router-dom'
import { router } from './router'
import './style.css'


console.log(
  `%c我望向人群以上
某道星光很漂亮
岁月旅途除了荒凉
还是有太多事可拍掌
要令我爱人微笑着
要令朋友痛快一场
这是你完成使命时
我所想

Ya yi ya hoo~

—— 周柏豪《小白》`,
  'color: #b89270; font-size: 13px; line-height: 1.8;'
)

createRoot(document.getElementById('root')!).render(<StrictMode><RouterProvider router={router} /></StrictMode>)
