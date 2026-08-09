// CDP 追踪：重载页面，捕获控制台/异常/导航，定位 file:///C:/ 来源
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
  if (!page) throw new Error('no page')
  const ws = new WebSocket(page.webSocketDebuggerUrl)
  await new Promise((r, j) => ((ws.onopen = r), (ws.onerror = j)))

  ws.onmessage = (ev) => {
    const msg = JSON.parse(ev.data)
    if (msg.method === 'Runtime.consoleAPICalled') {
      const type = msg.params.type
      const text = msg.params.args.map((a) => a.value ?? a.description ?? '').join(' ')
      if (type === 'error' || type === 'warning') console.log(`[${type}]`, text.slice(0, 300))
    }
    if (msg.method === 'Runtime.exceptionThrown') {
      console.log('[EXCEPTION]', (msg.params.exceptionDetails?.exception?.description || msg.params.exceptionDetails?.text || '').slice(0, 500))
    }
    if (msg.method === 'Page.frameNavigated') {
      console.log('[NAVIGATED]', msg.params.frame.url)
    }
    if (msg.method === 'Page.navigatedWithinDocument') {
      console.log('[URL_CHANGED]', msg.params.url)
    }
  }

  ws.send(JSON.stringify({ id: 1, method: 'Runtime.enable' }))
  ws.send(JSON.stringify({ id: 2, method: 'Page.enable' }))
  await new Promise((r) => setTimeout(r, 500))
  console.log('--- reloading ---')
  ws.send(JSON.stringify({ id: 3, method: 'Page.reload', params: { ignoreCache: true } }))
  await new Promise((r) => setTimeout(r, 9000))

  // 最终状态
  const final = await new Promise((resolve) => {
    const handler = (ev) => {
      const msg = JSON.parse(ev.data)
      if (msg.id === 4) {
        ws.removeEventListener('message', handler)
        resolve(msg.result?.result?.value)
      }
    }
    ws.addEventListener('message', handler)
    ws.send(JSON.stringify({ id: 4, method: 'Runtime.evaluate', params: { expression: `JSON.stringify({url:location.href, rootLen:(document.getElementById('root')?.innerHTML||'').length, bodyText:(document.body.innerText||'').slice(0,60)})`, returnByValue: true } }))
  })
  console.log('FINAL:', final)
  ws.close()
  process.exit(0)
}

main().catch((e) => {
  console.error('ERR:', e.message)
  process.exit(1)
})
