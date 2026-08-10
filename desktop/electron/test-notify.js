// 系统通知 IPC 打桩验证（headless）：注入 mock ipcMain/Notification，
// 断言「ipcMain.on('notify') 注册 + new Notification().show() 触发」。
// 运行：node test-notify.js
const { registerNotifyIpc } = require('./notify')

let pass = 0
let fail = 0

function assert(cond, name) {
  if (cond) {
    pass++
    console.log(`  ✔ ${name}`)
  } else {
    fail++
    console.error(`  ✘ ${name}`)
  }
}

// mock 主进程 IPC：只捕获 notify 通道的处理器
const handlers = {}
const mockIpcMain = {
  on: (channel, handler) => {
    handlers[channel] = handler
  },
}

// mock 原生 Notification：记录实例化参数，支持 isSupported/isSupported=返回假 两种场景
let isSupportedValue = true
const shown = []
function MockNotification({ title, body, silent }) {
  this.title = title
  this.body = body
  this.silent = silent
  shown.push({ title, body, silent })
}
MockNotification.isSupported = () => isSupportedValue
MockNotification.prototype.show = function () {}

// 1) 注册：notify 通道处理器已挂上
registerNotifyIpc(mockIpcMain, MockNotification)
assert(typeof handlers.notify === 'function', 'ipcMain.on("notify") 处理器已注册')

// 2) 正常弹窗：payload 传入 title/body，实例化为 silent:false
const prevShown = shown.length
handlers.notify(null, { title: '新的请假申请待审批', body: '张三 提交了请假申请' })
assert(shown.length === prevShown + 1, 'new Notification().show() 被调用一次')
assert(shown[prevShown].title === '新的请假申请待审批', '通知标题透传')
assert(shown[prevShown].body === '张三 提交了请假申请', '通知内容透传')
assert(shown[prevShown].silent === false, 'silent 固定 false（响铃提示）')

// 3) 缺省 payload：兜底默认标题，body 空串
const prevShown2 = shown.length
handlers.notify(null)
assert(shown.length === prevShown2 + 1, '无 payload 仍弹通知')
assert(shown[prevShown2].title === 'HRMS 新通知', '缺省标题用默认「HRMS 新通知」')

// 4) 过长标题/内容截断（128/500）
handlers.notify(null, { title: '标'.repeat(300), body: '内'.repeat(800) })
const last = shown[shown.length - 1]
assert(last.title.length === 128, '标题截断到 128')
assert(last.body.length === 500, '内容截断到 500')

// 5) 平台不支持：isSupported 为 false 时不弹
isSupportedValue = false
const prevShown3 = shown.length
handlers.notify(null, { title: '不弹', body: 'x' })
assert(shown.length === prevShown3, 'Notification.isSupported()=false 时不弹（跳过）')
isSupportedValue = true

console.log(`\n${pass} 通过，${fail} 失败`)
process.exit(fail ? 1 : 0)
