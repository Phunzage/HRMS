// CDP：直接导航到 asar 内 index.html，捕获控制台错误/异常/资源加载结果
const http = require('http')

const ASAR_INDEX =
  'file:///C:/Users/Phunzage/Documents/Code/HRMS/desktop/dist/win-unpacked/resources/app.asar/build-web/index.html'

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
      if (type === 'error') console.log('[CONSOLE_ERROR]', text.slice(0, 400))
    }
    if (msg.method === 'Runtime.exceptionThrown') {
      console.log('[EXCEPTION]', (msg.params.exceptionDetails?.exception?.description || msg.params.exceptionDetails?.text || '').slice(0, 400))
    }
    if (msg.method === 'Network.loadingFailed') {
      console.log('[LOAD_FAILED]', msg.params.errorText, msg.params.canceled ? '(canceled)' : '', msg.params.type)
    }
    if (msg.method === 'Page.frameNavigated') console.log('[NAVIGATED]', msg.params.frame.url)
    if (msg.method === 'Page.navigatedWithinDocument') console.log('[URL_CHANGED]', msg.params.url)
  }

  ws.send(JSON.stringify({ id: 1, method: 'Runtime.enable' }))
  ws.send(JSON.stringify({ id: 2, method: 'Page.enable' }))
  ws.send(JSON.stringify({ id: 3, method: 'Network.enable' }))
  await new Promise((r) => setTimeout(r, 300))

  console.log('--- navigating to asar index ---')
  ws.send(JSON.stringify({ id: 4, method: 'Page.navigate', params: { url: ASAR_INDEX } }))
  await new Promise((r) => setTimeout(r, 8000))

  const final = await new Promise((resolve) => {
    const handler = (ev) => {
      const msg = JSON.parse(ev.data)
      if (msg.id === 5) {
        ws.removeEventListener('message', handler)
        resolve(msg.result?.result?.value)
      }
    }
    ws.addEventListener('message', handler)
    ws.send(JSON.stringify({ id: 5, method: 'Runtime.evaluate', params: { expression: `JSON.stringify({url:location.href, rootLen:(document.getElementById('root')?.innerHTML||'').length, reactHook:!!window.__REACT_DEVTOOLS_GLOBAL_HOOK__})`, returnByValue: true } }))
  })
  console.log('FINAL:', final)
  ws.close()
  process.exit(0)
}

main().catch((e) => {
  console.error('ERR:', e.message)
  process.exit(1)
})
