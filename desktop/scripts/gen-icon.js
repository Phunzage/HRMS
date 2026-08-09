// 生成 256x256 占位图标 build/icon.png（蓝底白字 "H"，后续可美化替换）
// 纯 Node 实现，不依赖外部图像库：手工构造 PNG（IHDR + zlib IDAT + IEND）
const fs = require('fs')
const path = require('path')
const zlib = require('zlib')

const SIZE = 256
const BG = [29, 78, 216, 255] // #1d4ed8 蓝
const FG = [255, 255, 255, 255] // 白

// 简易 CRC32（PNG chunk 校验）
function crc32(buf) {
  const table = []
  for (let n = 0; n < 256; n++) {
    let c = n
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1
    table[n] = c >>> 0
  }
  let crc = 0xffffffff
  for (let i = 0; i < buf.length; i++) crc = table[(crc ^ buf[i]) & 0xff] ^ (crc >>> 8)
  return (crc ^ 0xffffffff) >>> 0
}

function chunk(type, data) {
  const len = Buffer.alloc(4)
  len.writeUInt32BE(data.length)
  const typeBuf = Buffer.from(type, 'ascii')
  const crcBuf = Buffer.alloc(4)
  crcBuf.writeUInt32BE(crc32(Buffer.concat([typeBuf, data])))
  return Buffer.concat([len, typeBuf, data, crcBuf])
}

// 绘制白色 "H"（两个竖条 + 一条横梁）
function isH(x, y) {
  const barW = 44
  const barH = 170
  const gap = 34
  const top = 43
  const left = (SIZE - 2 * barW - gap) / 2
  const crossY = top + barH / 2 - 20
  const inLeft = x >= left && x < left + barW && y >= top && y < top + barH
  const inRight = x >= left + barW + gap && x < left + 2 * barW + gap && y >= top && y < top + barH
  const inCross = x >= left - 12 && x < left + 2 * barW + gap + 12 && y >= crossY && y < crossY + 40
  return inLeft || inRight || inCross
}

// 逐像素构造 RGBA 位图（每行前置 filter byte 0）
const raw = Buffer.alloc(SIZE * (1 + SIZE * 4))
let p = 0
for (let y = 0; y < SIZE; y++) {
  raw[p++] = 0
  for (let x = 0; x < SIZE; x++) {
    const c = isH(x, y) ? FG : BG
    raw[p++] = c[0]
    raw[p++] = c[1]
    raw[p++] = c[2]
    raw[p++] = c[3]
  }
}

const ihdr = Buffer.alloc(13)
ihdr.writeUInt32BE(SIZE, 0)
ihdr.writeUInt32BE(SIZE, 4)
ihdr[8] = 8 // bit depth
ihdr[9] = 6 // color type: RGBA
ihdr[10] = 0
ihdr[11] = 0
ihdr[12] = 0

const png = Buffer.concat([
  Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
  chunk('IHDR', ihdr),
  chunk('IDAT', zlib.deflateSync(raw)),
  chunk('IEND', Buffer.alloc(0)),
])

const out = path.join(__dirname, '..', 'build', 'icon.png')
fs.mkdirSync(path.dirname(out), { recursive: true })
fs.writeFileSync(out, png)
console.log(`icon written: ${out} (${png.length} bytes)`)
