declare const uni: {
  request(options: {
    url: string
    method?: 'GET' | 'POST' | 'PUT' | 'DELETE'
    data?: string | object | ArrayBuffer
    header?: Record<string, string>
    success?: (res: { statusCode: number; data: unknown }) => void
    fail?: (err: { errMsg?: string }) => void
  }): void
}
