// 通过 CDP 检查渲染进程更新提示条状态（验证更新链路）
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

async function main() {
  const targets = await getTargets()
  const page = targets.find((t) => t.type === 'page')
  if (!page) {
    console.log('NO_PAGE_TARGET')
    return
  }
  const ws = new WebSocket(page.webSocketDebuggerUrl)
  await new Promise((resolve, reject) => {
    ws.onopen = resolve
    ws.onerror = reject
  })
  const expr = `JSON.stringify({
    hasBar: !!document.getElementById('hrms-update-bar'),
    display: document.getElementById('hrms-update-bar')?.style.display || null,
    text: document.getElementById('hrms-update-bar')?.textContent || null,
    hasUpdater: !!window.hrmsUpdater,
  })`
  const result = await new Promise((resolve, reject) => {
    ws.onmessage = (ev) => {
      const msg = JSON.parse(ev.data)
      if (msg.id === 1) resolve(msg.result)
    }
    ws.send(
      JSON.stringify({
        id: 1,
        method: 'Runtime.evaluate',
        params: { expression: expr, returnByValue: true },
      }),
    )
  })
  console.log('CDP_RESULT:', result?.result?.value)
  ws.close()
}

main().catch((e) => {
  console.error('CDP_ERROR:', e.message)
  process.exit(1)
})
