import { useEffect, useRef, useCallback } from 'react'
import { examAPI } from '@/services/api'
import { useExamStore } from '@/store/examStore'

interface Options {
  attemptID: string
  maxTabSwitch: number
  onPause: (reason: string) => void
  enabled: boolean
}

export function useAntiCheat({ attemptID, maxTabSwitch, onPause, enabled }: Options) {
  const tabSwitchCount = useRef(0)
  const cooldownRef    = useRef(false)
  const bufferMap      = useRef<Record<string, number>>({})
  const { incrementCheating } = useExamStore()

  const reportViolation = useCallback(async (eventType: string, detail?: string) => {
    if (!enabled || !attemptID) return

    // Buffer: only fire after 3 consecutive detections for AI events
    if (eventType.startsWith('face_') || eventType === 'multiple_faces') {
      bufferMap.current[eventType] = (bufferMap.current[eventType] ?? 0) + 1
      if (bufferMap.current[eventType] < 3) return
      bufferMap.current[eventType] = 0
    }

    // Cooldown: prevent spam (2 s window)
    if (cooldownRef.current) return
    cooldownRef.current = true
    setTimeout(() => { cooldownRef.current = false }, 2000)

    incrementCheating()

    try {
      const res = await examAPI.reportViolation(attemptID, eventType, detail)
      if (res.data?.data?.paused) {
        onPause(eventType)
      }
    } catch { /* silent */ }
  }, [attemptID, enabled, incrementCheating, onPause])

  // ── Tab visibility ──────────────────────────────────────────────────────────
  useEffect(() => {
    if (!enabled) return
    const handleVisibility = () => {
      if (document.visibilityState === 'hidden') {
        tabSwitchCount.current++
        reportViolation('tab_switch', `count:${tabSwitchCount.current}`)
        if (tabSwitchCount.current >= maxTabSwitch) {
          onPause('max_tab_switch_exceeded')
        }
      }
    }
    document.addEventListener('visibilitychange', handleVisibility)
    return () => document.removeEventListener('visibilitychange', handleVisibility)
  }, [enabled, maxTabSwitch, onPause, reportViolation])

  // ── Fullscreen exit ─────────────────────────────────────────────────────────
  useEffect(() => {
    if (!enabled) return
    const handleFSChange = () => {
      if (!document.fullscreenElement) {
        reportViolation('fullscreen_exit', 'exited fullscreen')
      }
    }
    document.addEventListener('fullscreenchange', handleFSChange)
    return () => document.removeEventListener('fullscreenchange', handleFSChange)
  }, [enabled, reportViolation])

  // ── Clipboard override ──────────────────────────────────────────────────────
  useEffect(() => {
    if (!enabled) return
    const blockCopy  = (e: ClipboardEvent) => { e.preventDefault(); e.clipboardData?.setData('text/plain', '?????') }
    const blockPaste = (e: ClipboardEvent) => { e.preventDefault() }
    const blockCtx   = (e: MouseEvent)     => { e.preventDefault() }
    document.addEventListener('copy',        blockCopy  as EventListener)
    document.addEventListener('cut',         blockCopy  as EventListener)
    document.addEventListener('paste',       blockPaste as EventListener)
    document.addEventListener('contextmenu', blockCtx)
    document.body.classList.add('exam-mode')
    return () => {
      document.removeEventListener('copy',        blockCopy  as EventListener)
      document.removeEventListener('cut',         blockCopy  as EventListener)
      document.removeEventListener('paste',       blockPaste as EventListener)
      document.removeEventListener('contextmenu', blockCtx)
      document.body.classList.remove('exam-mode')
    }
  }, [enabled])

  // ── Keyboard shortcuts (PrintScreen etc.) ──────────────────────────────────
  useEffect(() => {
    if (!enabled) return
    const handleKey = (e: KeyboardEvent) => {
      // Block F12, PrintScreen, common dev shortcuts
      if (e.key === 'F12' || e.key === 'PrintScreen' ||
          (e.ctrlKey && ['u','s','a','p'].includes(e.key.toLowerCase()))) {
        e.preventDefault()
        reportViolation('keyboard_shortcut', e.key)
      }
    }
    document.addEventListener('keydown', handleKey)
    return () => document.removeEventListener('keydown', handleKey)
  }, [enabled, reportViolation])

  const requestFullscreen = useCallback(() => {
    document.documentElement.requestFullscreen?.().catch(() => {})
  }, [])

  return { reportViolation, requestFullscreen, tabSwitchCount: tabSwitchCount.current }
}
