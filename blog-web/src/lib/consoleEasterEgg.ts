let previousColor = ''

// 读取已经应用的页面主题色；相同颜色不重复输出，避免页面更新和 StrictMode 刷屏。
export function logConsoleEasterEgg() {
  const color = getComputedStyle(document.documentElement).getPropertyValue('--accent-strong').trim() || '#b89270'
  if (color === previousColor) return
  previousColor = color

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
    `color: ${color}; font-size: 13px; line-height: 1.8;`
  )
}
