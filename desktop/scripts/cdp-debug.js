// CDP 调试：检查页面加载状态 + 重新触发检查后查看更新条
const http = require('http')

function getTargets() {
  return new Promise((resolve, reject) => {
    http
      .get('http://127.0.0.1:9222/json/list', (res) => {
        let data = ''
        res.on('data', (c) => (data += c))
        res.on('end', () => resolve(JSON.parse(data)))
      })
      .on('error', reject)
  })
}

function evaluate(ws, id, expression) {
  return new Promise((resolve) => {
    ws.onmessage = (ev) => {
      const msg = JSON.parse(ev.data)
      if (msg.id === id) resolve(msg.result)
    }
    ws.send(JSON.stringify({ id, method: 'Runtime.evaluate', params: { expression, returnByValue: true } }))
  })
}

async function main() {
  const targets = await getTargets()
  const page = targets.find((t) => t.type === 'page')
  if (!page) throw new Error('no page')
  const ws = new WebSocket(page.webSocketDebuggerUrl)
  await new Promise((r, j) => ((ws.onopen = r), (ws.onerror = j)))

  // 1. 页面状态
  const state = await evaluate(
    ws,
    1,
    `JSON.stringify({
      readyState: document.readyState,
      rootLen: (document.getElementById('root')?.innerHTML || '').length,
      title: document.title,
    })`,
  )
  console.log('STATE:', state?.result?.value)

  // 2. 手动触发一次更新检查（走完整 IPC 链路）
  await evaluate(ws, 2, `window.hrmsUpdater.checkNow()`)
  console.log('checkNow() invoked, waiting 3s...')
  await new Promise((r) => setTimeout(r, 3000))

  // 3. 再查更新条
  const after = await evaluate(
    ws,
    3,
    `JSON.stringify({
      hasBar: !!document.getElementById('hrms-update-bar'),
      display: document.getElementById('hrms-update-bar')?.style.display || null,
      text: document.getElementById('hrms-update-bar')?.textContent || null,
    })`,
  )
  console.log('AFTER_RECHECK:', after?.result?.value)
  ws.close()
}

main().catch((e) => {
  console.error('ERR:', e.message)
  process.exit(1)
})
