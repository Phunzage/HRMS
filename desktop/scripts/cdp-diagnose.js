// CDP 诊断：脚本元素、资源加载、错误捕获
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

  // 捕获控制台错误
  ws.send(JSON.stringify({ id: 99, method: 'Runtime.enable' }))
  await new Promise((r) => setTimeout(r, 500))
  ws.onmessage = (ev) => {
    const msg = JSON.parse(ev.data)
    if (msg.method === 'Runtime.exceptionThrown') {
      console.log('EXCEPTION:', JSON.stringify(msg.params.exceptionDetails?.exception?.description || msg.params.exceptionDetails?.text))
    }
    if (msg.method === 'Runtime.consoleAPICalled' && msg.params.type === 'error') {
      console.log('CONSOLE_ERROR:', msg.params.args.map((a) => a.value || a.description).join(' '))
    }
  }

  const scripts = await evaluate(ws, 1, `Array.from(document.scripts).map(function(s){return {type:s.type,src:s.src,textLen:(s.text||'').length}})`)
  console.log('SCRIPTS:', JSON.stringify(scripts?.result?.value))

  const perf = await evaluate(ws, 2, `JSON.stringify(performance.getEntriesByType('resource').map(function(e){return e.name}).filter(function(n){return n.includes('assets')}))`)
  console.log('PERF_ASSETS:', perf?.result?.value)

  const check = await evaluate(ws, 3, `JSON.stringify({rootLen:(document.getElementById('root')?.innerHTML||'').length, url:location.href})`)
  console.log('CHECK:', check?.result?.value)

  ws.close()
  process.exit(0)
}

main().catch((e) => {
  console.error('ERR:', e.message)
  process.exit(1)
})
