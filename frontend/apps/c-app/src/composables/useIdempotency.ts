/**
 * 生成 UUID v4 格式的幂等键，用于写操作请求头 X-Idempotency-Key。
 * 纯前端实现，不依赖第三方库。
 */
export function generateIdempotencyKey(): string {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0
    const v = c === 'x' ? r : (r & 0x3) | 0x8
    return v.toString(16)
  })
}
