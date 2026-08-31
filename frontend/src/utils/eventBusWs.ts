/**
 * 管理端 eventbus WebSocket：自动重连 + 主动关闭不重连。
 * 服务端会主动断开死连接（写超时），前端需自动重连恢复推送。
 */
export class EventBusWs {
  private ws: WebSocket | null = null
  private retry: ReturnType<typeof setTimeout> | null = null
  private closed = false
  // 重连退避：3s 起步，翻倍封顶 60s；连上后重置（会话过期时避免每 3s 打一次 401）
  private retryDelay = 3000
  private readonly maxRetryDelay = 60000

  constructor(
    private readonly url: string,
    private readonly onMessage: (evt: MessageEvent) => void,
    private readonly onStatus?: (connected: boolean) => void
  ) {}

  connect() {
    if (this.ws) {
      return
    }
    this.closed = false
    const ws = (this.ws = new WebSocket(this.url))
    ws.onopen = () => {
      this.retryDelay = 3000
      this.onStatus?.(true)
    }
    ws.onmessage = (evt) => this.onMessage(evt)
    ws.onclose = () => {
      this.ws = null
      this.onStatus?.(false)
      if (this.closed) {
        return // 主动 close() 触发的关闭，不重连
      }
      const delay = this.retryDelay
      this.retryDelay = Math.min(this.retryDelay * 2, this.maxRetryDelay)
      this.retry = setTimeout(() => this.connect(), delay)
    }
  }

  close() {
    this.closed = true
    if (this.retry) {
      clearTimeout(this.retry)
      this.retry = null
    }
    if (this.ws) {
      this.ws.close()
    }
  }
}
