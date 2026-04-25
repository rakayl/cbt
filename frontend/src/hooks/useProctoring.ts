import { useEffect, useRef, useCallback } from 'react'
import { examAPI } from '@/services/api'
import { useExamStore } from '@/store/examStore'

const FRAME_INTERVAL_MS = 2500

interface Options {
  attemptID: string
  baseEmbedding?: number[]
  onViolation?: (type: string) => void
  enabled: boolean
}

export function useProctoring({ attemptID, baseEmbedding, onViolation, enabled }: Options) {
  const videoRef   = useRef<HTMLVideoElement | null>(null)
  const canvasRef  = useRef<HTMLCanvasElement | null>(null)
  const streamRef  = useRef<MediaStream | null>(null)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const { status } = useExamStore()

  const captureFrame = useCallback((): string | null => {
    const video  = videoRef.current
    const canvas = canvasRef.current
    if (!video || !canvas || video.readyState < 2) return null
    const ctx = canvas.getContext('2d')
    if (!ctx) return null
    canvas.width  = 320
    canvas.height = 240
    ctx.drawImage(video, 0, 0, 320, 240)
    return canvas.toDataURL('image/jpeg', 0.7).split(',')[1]
  }, [])

  const sendFrame = useCallback(async () => {
    if (status !== 'ongoing' || !enabled) return
    const frame = captureFrame()
    if (!frame) return
    try {
      const res = await examAPI.sendFrame(attemptID, frame, baseEmbedding)
      const { violation } = res.data?.data ?? {}
      if (violation) onViolation?.(violation)
    } catch { /* AI service down — skip silently */ }
  }, [attemptID, baseEmbedding, captureFrame, enabled, onViolation, status])

  // Start webcam
  useEffect(() => {
    if (!enabled) return
    navigator.mediaDevices
      .getUserMedia({ video: { width: 320, height: 240, facingMode: 'user' }, audio: false })
      .then(stream => {
        streamRef.current = stream
        if (videoRef.current) videoRef.current.srcObject = stream
      })
      .catch(err => console.warn('[Proctoring] Webcam error:', err))

    return () => {
      streamRef.current?.getTracks().forEach(t => t.stop())
    }
  }, [enabled])

  // Start frame capture loop
  useEffect(() => {
    if (!enabled) return
    intervalRef.current = setInterval(sendFrame, FRAME_INTERVAL_MS)
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [enabled, sendFrame])

  return { videoRef, canvasRef }
}
