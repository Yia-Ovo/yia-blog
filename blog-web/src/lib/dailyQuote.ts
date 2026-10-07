import type { Locale } from '../i18n'

/** 本站自编短句及其译文，不引用或署名为名人名言。 */
export const dailyQuotes = [
  { 'zh-CN': '把今天写成一小步，明天便有了来处。', en: 'Take one small step today, and tomorrow will have a place to begin.', ja: '今日を小さな一歩にすれば、明日の出発点になる。', vi: 'Đi một bước nhỏ hôm nay, ngày mai sẽ có nơi để bắt đầu.' },
  { 'zh-CN': '偶尔慢下来，也是在认真经过生活。', en: 'Slowing down can be a way of paying attention to life.', ja: 'ときには歩みを緩めることも、暮らしを大切にする方法だ。', vi: 'Đôi khi chậm lại cũng là cách để sống thật trọn vẹn.' },
  { 'zh-CN': '留一点空白，新的故事才有地方落笔。', en: 'Leave a little space for a new story to begin.', ja: '少し余白を残そう。新しい物語を書き始められるように。', vi: 'Hãy chừa một chút khoảng trống để câu chuyện mới có chỗ bắt đầu.' },
  { 'zh-CN': '不必一次走很远，记得继续就好。', en: 'You do not have to go far all at once. Just keep going.', ja: '一度に遠くまで行かなくていい。また歩き出せれば、それでいい。', vi: 'Không cần đi thật xa trong một lần. Cứ tiếp tục là được.' },
  { 'zh-CN': '认真做过的小事，会在某天接住你。', en: 'The small things you do with care may one day support you.', ja: '心を込めた小さなことが、いつか自分を支えてくれる。', vi: 'Những việc nhỏ bạn làm bằng cả tấm lòng có thể sẽ nâng đỡ bạn một ngày nào đó.' },
  { 'zh-CN': '窗外有天气，心里也可以有自己的晴天。', en: 'Whatever the weather outside, there can be a little sunshine within.', ja: '窓の外がどんな天気でも、心には自分だけの晴れ間があっていい。', vi: 'Dù ngoài trời thế nào, trong lòng vẫn có thể có một khoảng nắng riêng.' },
  { 'zh-CN': '答案还没出现的时候，先照顾好提问的人。', en: 'While waiting for an answer, take care of the person asking.', ja: '答えがまだ見つからないときは、問い続ける自分をいたわろう。', vi: 'Khi câu trả lời chưa đến, hãy chăm sóc người đang đặt câu hỏi trước đã.' },
  { 'zh-CN': '热爱不一定响亮，也可以是每天多做一点。', en: 'Passion can be quiet: doing a little more each day.', ja: '好きな気持ちは、大きな声でなくてもいい。毎日少し続けることでも伝わる。', vi: 'Đam mê không cần ồn ào; đôi khi chỉ là làm thêm một chút mỗi ngày.' },
  { 'zh-CN': '把好奇留住，平常的路也会有新风景。', en: 'Stay curious, and familiar roads may offer new views.', ja: '好奇心を忘れなければ、いつもの道にも新しい景色が見つかる。', vi: 'Giữ lấy sự tò mò, con đường quen cũng sẽ có những cảnh sắc mới.' },
  { 'zh-CN': '休息不是把故事合上，只是换一口气再读。', en: 'Rest does not close the story. It gives you a breath before the next page.', ja: '休むことは物語を閉じることではない。次のページの前に、ひと息つくだけ。', vi: 'Nghỉ ngơi không phải là khép lại câu chuyện, chỉ là lấy hơi trước trang tiếp theo.' },
  { 'zh-CN': '今天没解决的事，不必变成今晚对自己的苛责。', en: 'What remains unfinished today need not become a reason to be harsh on yourself tonight.', ja: '今日できなかったことを、今夜の自分を責める理由にしなくていい。', vi: 'Những việc chưa xong hôm nay không cần trở thành lý do để trách mình tối nay.' },
  { 'zh-CN': '给普通的一天留个记号，它也曾独一无二。', en: 'Leave a small mark on an ordinary day. It, too, will happen only once.', ja: '何気ない一日にも、小さな印を残そう。その日も一度きりだから。', vi: 'Hãy để lại một dấu nhỏ cho một ngày bình thường. Ngày ấy cũng chỉ đến một lần.' },
] as const satisfies ReadonlyArray<Record<Locale, string>>

export function localDateKey(date: Date): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

/** 用本地日历日计算序号，避免夏令时让相邻日期重复或跳过。 */
export function dailyQuoteIndex(date: Date, offset = 0): number {
  const day = Math.floor(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()) / 86_400_000)
  return ((day + offset) % dailyQuotes.length + dailyQuotes.length) % dailyQuotes.length
}

export function nextLocalMidnightDelay(date: Date): number {
  const midnight = new Date(date)
  midnight.setHours(24, 0, 0, 0)
  return midnight.getTime() - date.getTime()
}
