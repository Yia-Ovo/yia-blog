import { Link } from 'react-router-dom'
import StatePanel from '../components/StatePanel'

export default function NotFoundView() {
  return <div className="container reading-page"><StatePanel kind="empty" title="这页记录还不存在" message="链接可能有误，回到首页继续看看吧。"><Link className="button primary small" to="/">返回首页</Link></StatePanel></div>
}
