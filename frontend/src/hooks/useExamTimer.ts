import { useEffect, useRef, useCallback } from 'react'
import { useExamStore } from '@/store/examStore'
import { examAPI } from '@/services/api'

export function useExamTimer(attemptID: string | null) {
  const { sisaDetik, status, setTimer } = useExamStore()
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const lastSaveRef = useRef(0)

  const tick = useCallback(() => {
    const s = useExamStore.getState()
    if (s.status !== 'ongoing' || s.sisaDetik <= 0) return

    setTimer(s.sisaDetik - 1)

    // Auto-save timer to server every 30 s
    if (s.sisaDetik % 30 === 0 && attemptID) {
      examAPI.saveAnswer({
        attempt_id: attemptID,
        soal_id: '__heartbeat__',
        sisa_detik: s.sisaDetik,
      }).catch(() => {})
    }

    lastSaveRef.current = s.sisaDetik
  }, [attemptID, setTimer])

  useEffect(() => {
    if (status === 'ongoing') {
      intervalRef.current = setInterval(tick, 1000)
    } else {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
    return () => { if (intervalRef.current) clearInterval(intervalRef.current) }
  }, [status, tick])

  const formatTime = (secs: number) => {
    const h = Math.floor(secs / 3600)
    const m = Math.floor((secs % 3600) / 60)
    const s = secs % 60
    if (h > 0) return `${h}:${String(m).padStart(2,'0')}:${String(s).padStart(2,'0')}`
    return `${String(m).padStart(2,'0')}:${String(s).padStart(2,'0')}`
  }

  return {
    sisaDetik,
    formatted: formatTime(sisaDetik),
    isWarning: sisaDetik < 300 && sisaDetik > 0,
    isDanger:  sisaDetik < 60  && sisaDetik > 0,
    isExpired: sisaDetik <= 0,
  }
}
