import client from './client'

// 上传图片，返回可公开访问的相对路径（/uploads/YYYYMM/xxx.png）
export function uploadFile(file: File) {
  const formData = new FormData()
  formData.append('file', file)
  return client.post('/upload', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  }) as Promise<{ url: string }>
}
