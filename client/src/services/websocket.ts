// WebSocket 消息类型
export enum MessageType {
  NewNotification = 'new_notification',
  UnreadCountUpdate = 'unread_count_update',
  Ping = 'ping',
  Pong = 'pong',
}

// WebSocket 消息结构
export interface WebSocketMessage {
  type: MessageType
  data: any
}

// WebSocket 事件回调
export type MessageHandler = (message: WebSocketMessage) => void

// WebSocket 服务类
export class WebSocketService {
  private ws: WebSocket | null = null
  private url: string
  private token: string
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 3000
  private reconnectTimer: number | null = null
  private pingTimer: number | null = null
  private messageHandlers: Map<MessageType, MessageHandler[]> = new Map()
  private isManualClose = false

  constructor(url: string, token: string) {
    this.url = url
    this.token = token
  }

  // 连接 WebSocket
  connect() {
    if (this.ws?.readyState === WebSocket.OPEN) {
      console.log('[WebSocket] 已经连接')
      return
    }

    this.isManualClose = false
    const wsUrl = `${this.url}?token=${this.token}`

    try {
      this.ws = new WebSocket(wsUrl)
      this.setupEventHandlers()
    } catch (error) {
      console.error('[WebSocket] 连接失败:', error)
      this.scheduleReconnect()
    }
  }

  // 设置事件处理器
  private setupEventHandlers() {
    if (!this.ws) return

    this.ws.onopen = () => {
      console.log('[WebSocket] 连接成功')
      this.reconnectAttempts = 0
      this.startPing()
    }

    this.ws.onmessage = (event) => {
      try {
        const message: WebSocketMessage = JSON.parse(event.data)
        console.log('[WebSocket] 收到消息:', message)
        this.handleMessage(message)
      } catch (error) {
        console.error('[WebSocket] 解析消息失败:', error)
      }
    }

    this.ws.onerror = (error) => {
      console.error('[WebSocket] 错误:', error)
    }

    this.ws.onclose = () => {
      console.log('[WebSocket] 连接关闭')
      this.stopPing()

      if (!this.isManualClose) {
        this.scheduleReconnect()
      }
    }
  }

  // 处理接收到的消息
  private handleMessage(message: WebSocketMessage) {
    const handlers = this.messageHandlers.get(message.type)
    if (handlers) {
      handlers.forEach(handler => handler(message))
    }
  }

  // 注册消息处理器
  on(type: MessageType, handler: MessageHandler) {
    if (!this.messageHandlers.has(type)) {
      this.messageHandlers.set(type, [])
    }
    this.messageHandlers.get(type)!.push(handler)
  }

  // 移除消息处理器
  off(type: MessageType, handler: MessageHandler) {
    const handlers = this.messageHandlers.get(type)
    if (handlers) {
      const index = handlers.indexOf(handler)
      if (index > -1) {
        handlers.splice(index, 1)
      }
    }
  }

  // 发送消息
  send(type: MessageType, data: any) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type, data }))
    } else {
      console.warn('[WebSocket] 连接未打开，无法发送消息')
    }
  }

  // 启动心跳
  private startPing() {
    this.stopPing()
    this.pingTimer = window.setInterval(() => {
      this.send(MessageType.Ping, {})
    }, 30000) // 每30秒发送一次心跳
  }

  // 停止心跳
  private stopPing() {
    if (this.pingTimer) {
      clearInterval(this.pingTimer)
      this.pingTimer = null
    }
  }

  // 安排重连
  private scheduleReconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
    }

    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.log('[WebSocket] 达到最大重连次数，停止重连')
      return
    }

    this.reconnectAttempts++
    const delay = this.reconnectDelay * this.reconnectAttempts
    console.log(`[WebSocket] ${delay}ms 后尝试第 ${this.reconnectAttempts} 次重连`)

    this.reconnectTimer = window.setTimeout(() => {
      this.connect()
    }, delay)
  }

  // 断开连接
  disconnect() {
    this.isManualClose = true
    this.stopPing()

    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }

    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  // 获取连接状态
  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }
}
