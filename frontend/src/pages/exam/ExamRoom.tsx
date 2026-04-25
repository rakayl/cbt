import { useEffect, useState, useCallback } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { examAPI } from '@/services/api'
import { useExamStore } from '@/store/examStore'
import { useAuthStore } from '@/store/authStore'
import { useAntiCheat } from '@/hooks/useAntiCheat'
import { useProctoring } from '@/hooks/useProctoring'
import { useWebSocket } from '@/hooks/useWebSocket'
import ExamTimer from '@/components/exam/ExamTimer'
import SoalCard from '@/components/exam/SoalCard'
import SoalNavigator from '@/components/exam/SoalNavigator'
import ProctorCamera from '@/components/exam/ProctorCamera'
import ExamPausedOverlay from '@/components/exam/ExamPausedOverlay'

export default function ExamRoom() {
  const [searchParams]    = useSearchParams()
  const ujianID           = searchParams.get('ujian_id') ?? ''
  const navigate          = useNavigate()
  const { user }          = useAuthStore()
  const [loading, setLoading]     = useState(true)
  const [error, setError]         = useState('')
  const [violation, setViolation] = useState('')
  const [finishing, setFinishing] = useState(false)
  const [showConfirmFinish, setShowConfirmFinish] = useState(false)

  const {
    attemptID, soalList, currentIdx, status, cheatingScore, pauseReason,
    setAttempt, setSoalList, setTimer, setStatus, setCurrentIdx, setPause, reset,
  } = useExamStore()

  // ── Handlers ──────────────────────────────────────────────────────────────
  const handlePause = useCallback((reason: string) => {
    setPause(reason)
  }, [setPause])

  const handleViolation = useCallback((type: string) => {
    setViolation(type)
    setTimeout(() => setViolation(''), 5000)
  }, [])

  // ── Anti-cheat ────────────────────────────────────────────────────────────
  const { requestFullscreen } = useAntiCheat({
    attemptID:    attemptID ?? '',
    maxTabSwitch: 3,
    onPause:      handlePause,
    enabled:      status === 'ongoing',
  })

  // ── Proctoring ────────────────────────────────────────────────────────────
  const { videoRef, canvasRef } = useProctoring({
    attemptID:  attemptID ?? '',
    onViolation: handleViolation,
    enabled:    status === 'ongoing',
  })

  // ── WebSocket ─────────────────────────────────────────────────────────────
  useWebSocket({
    attemptID: attemptID ?? '',
    role:      user?.role ?? 'peserta',
    enabled:   !!attemptID,
  })

  // ── Start exam ────────────────────────────────────────────────────────────
  useEffect(() => {
    if (!ujianID) { navigate('/exam'); return }

    const start = async () => {
      try {
        const res = await examAPI.start(ujianID)
        const { attempt, soal_list } = res.data.data
        setAttempt(attempt.id, ujianID)
        setSoalList(soal_list)
        setTimer(attempt.sisa_detik)
        setStatus(attempt.status === 'paused' ? 'paused' : 'ongoing')
        if (attempt.status === 'paused') setPause('previous_pause')
      } catch (err: any) {
        setError(err.response?.data?.error ?? 'Gagal memulai ujian')
      } finally {
        setLoading(false)
      }
    }

    start()
    requestFullscreen()

    return () => { reset() }
  }, [ujianID]) // eslint-disable-line

  // ── Finish exam ───────────────────────────────────────────────────────────
  const finishExam = async () => {
    if (!attemptID || finishing) return
    setFinishing(true)
    try {
      await examAPI.finish(attemptID)
      setStatus('finished')
      navigate('/exam/result?attempt_id=' + attemptID)
    } catch {
      setFinishing(false)
    }
  }

  // ── Navigation ────────────────────────────────────────────────────────────
  const goNext = () => {
    if (currentIdx < soalList.length - 1) setCurrentIdx(currentIdx + 1)
  }
  const goPrev = () => {
    if (currentIdx > 0) setCurrentIdx(currentIdx - 1)
  }

  // ── Render states ─────────────────────────────────────────────────────────
  if (loading) return (
    <div className="min-h-screen bg-slate-50 flex items-center justify-center">
      <div className="text-center">
        <svg className="w-8 h-8 text-blue-600 animate-spin mx-auto mb-3" fill="none" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"/>
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
        </svg>
        <p className="text-slate-600 font-medium">Menyiapkan ujian...</p>
      </div>
    </div>
  )

  if (error) return (
    <div className="min-h-screen bg-slate-50 flex items-center justify-center p-4">
      <div className="card p-8 max-w-sm w-full text-center">
        <svg className="w-12 h-12 text-red-400 mx-auto mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
        </svg>
        <h2 className="font-bold text-slate-800 mb-2">Tidak Dapat Memulai Ujian</h2>
        <p className="text-slate-500 text-sm mb-5">{error}</p>
        <button onClick={() => navigate('/exam')} className="btn-primary w-full justify-center">Kembali</button>
      </div>
    </div>
  )

  const currentSoal = soalList[currentIdx]

  return (
    <div className="min-h-screen bg-slate-50 flex flex-col">

      {/* Pause overlay */}
      {status === 'paused' && (
        <ExamPausedOverlay reason={pauseReason} cheatingScore={cheatingScore} />
      )}

      {/* Finish confirm dialog */}
      {showConfirmFinish && (
        <div className="fixed inset-0 z-40 bg-black/50 flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl p-6 max-w-sm w-full shadow-2xl animate-slide-up">
            <h3 className="font-bold text-slate-800 mb-2">Selesaikan Ujian?</h3>
            <p className="text-sm text-slate-500 mb-2">
              Anda telah menjawab{' '}
              <strong>{Object.keys(useExamStore.getState().jawabans).length}</strong> dari{' '}
              <strong>{soalList.length}</strong> soal.
            </p>
            <p className="text-xs text-amber-600 bg-amber-50 rounded-lg px-3 py-2 mb-5">
              Setelah diselesaikan, jawaban tidak dapat diubah lagi.
            </p>
            <div className="flex gap-3">
              <button onClick={() => setShowConfirmFinish(false)} className="btn-secondary flex-1 justify-center">
                Batal
              </button>
              <button onClick={finishExam} disabled={finishing} className="btn-primary flex-1 justify-center">
                {finishing ? 'Menyimpan...' : 'Selesai'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Header */}
      <header className="bg-white border-b border-slate-200 px-4 py-3 flex items-center justify-between flex-shrink-0 z-10">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-lg bg-blue-600 flex items-center justify-center">
            <svg className="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2"/>
            </svg>
          </div>
          <div>
            <p className="text-sm font-semibold text-slate-800 leading-none">CBT Enterprise</p>
            <p className="text-xs text-slate-400 mt-0.5">{user?.nama}</p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          {/* Cheating score indicator */}
          {cheatingScore > 0 && (
            <div className="flex items-center gap-1.5 px-2.5 py-1 bg-red-50 border border-red-200 rounded-lg">
              <svg className="w-3.5 h-3.5 text-red-500" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd"/>
              </svg>
              <span className="text-xs font-semibold text-red-700">{cheatingScore} pelanggaran</span>
            </div>
          )}
          <ExamTimer attemptID={attemptID} />
        </div>
      </header>

      {/* Main content */}
      <div className="flex-1 flex overflow-hidden">
        {/* Left: Question */}
        <main className="flex-1 overflow-hidden p-4 flex flex-col gap-4">
          {currentSoal && (
            <SoalCard
              attemptSoal={currentSoal}
              number={currentIdx + 1}
              total={soalList.length}
            />
          )}

          {/* Navigation buttons */}
          <div className="flex items-center justify-between flex-shrink-0">
            <button
              onClick={goPrev} disabled={currentIdx === 0}
              className="btn-secondary"
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7"/>
              </svg>
              Sebelumnya
            </button>

            <span className="text-sm text-slate-500">
              {currentIdx + 1} / {soalList.length}
            </span>

            {currentIdx < soalList.length - 1 ? (
              <button onClick={goNext} className="btn-primary">
                Selanjutnya
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7"/>
                </svg>
              </button>
            ) : (
              <button onClick={() => setShowConfirmFinish(true)} className="btn-danger">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7"/>
                </svg>
                Selesai
              </button>
            )}
          </div>
        </main>

        {/* Right sidebar */}
        <aside className="flex flex-col gap-4 p-4 flex-shrink-0">
          <ProctorCamera videoRef={videoRef} canvasRef={canvasRef} violation={violation || undefined} />
          <SoalNavigator onSelect={setCurrentIdx} />
        </aside>
      </div>
    </div>
  )
}
