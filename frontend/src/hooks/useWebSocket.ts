import { useEffect, useRef, useCallback } from 'react'
import { useExamStore } from '@/store/examStore'

type WSEvent =
  | 'heartbeat' | 'answer' | 'tab_switch' | 'fullscreen_exit'
  | 'cheating_detected' | 'exam_paused' | 'exam_finished'
  | 'face_alert' | 'exam_resumed'

interface WSMessage {
  event: WSEvent
  attempt_id?: string
  peserta_id?: string
  payload?: unknown
  reason?: string
}

type EventHandler = (msg: WSMessage) => void

interface Options {
  attemptID?: string
  role: string
  onMessage?: EventHandler
  enabled?: boolean
}

export function useWebSocket({ attemptID, role, onMessage, enabled = true }: Options) {
  const wsRef      = useRef<WebSocket | null>(null)
  const handlersRef = useRef<EventHandler | null>(null)
  const { setStatus, setPause } = useExamStore()

  handlersRef.current = onMessage ?? null

  const connect = useCallback(() => {
    if (!enabled) return
    const token = localStorage.getItem('cbt_token')
    const url   = `${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/api/v1/ws` +
                  `?attempt_id=${attemptID ?? ''}&token=${token ?? ''}`

    const ws = new WebSocket(url)
    wsRef.current = ws

    ws.onopen = () => {
      console.log('[WS] Connected')
    }

    ws.onmessage = (e) => {
      try {
        const msg: WSMessage = JSON.parse(e.data)

        // Built-in handlers
        switch (msg.event) {
          case 'exam_paused':
            setPause(msg.reason ?? 'paused')
            break
          case 'exam_resumed':
            setStatus('ongoing')
            break
          case 'exam_finished':
            setStatus('finished')
            break
        }

        handlersRef.current?.(msg)
      } catch { /* malformed */ }
    }

    ws.onclose = () => {
      // Reconnect after 3 s (unless exam finished)
      const status = useExamStore.getState().status
      if (status !== 'finished' && status !== 'idle') {
        setTimeout(connect, 3000)
      }
    }

    ws.onerror = () => ws.close()
  }, [attemptID, enabled, setPause, setStatus])

  useEffect(() => {
    connect()
    return () => wsRef.current?.close()
  }, [connect])

  const send = useCallback((msg: Partial<WSMessage>) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(msg))
    }
  }, [])

  return { send, ws: wsRef }
}
