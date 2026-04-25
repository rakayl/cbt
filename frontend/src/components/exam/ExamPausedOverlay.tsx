interface Props {
  reason: string
  cheatingScore: number
  onRequestResume?: () => void
}

const reasonLabels: Record<string, string> = {
  tab_switch:              'Perpindahan Tab',
  fullscreen_exit:         'Keluar Fullscreen',
  multiple_faces:          'Banyak Wajah Terdeteksi',
  no_face:                 'Wajah Tidak Terdeteksi',
  face_mismatch:           'Identitas Wajah Berbeda',
  cheating_detected:       'Kecurangan Terdeteksi',
  max_tab_switch_exceeded: 'Batas Tab Switch Terlampaui',
  keyboard_shortcut:       'Shortcut Keyboard Dilarang',
}

export default function ExamPausedOverlay({ reason, cheatingScore, onRequestResume }: Props) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/90 backdrop-blur-sm">
      <div className="bg-white rounded-2xl shadow-2xl max-w-md w-full mx-4 overflow-hidden animate-slide-up">
        {/* Header */}
        <div className="bg-red-600 px-6 py-5 flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-white/20 flex items-center justify-center flex-shrink-0">
            <svg className="w-6 h-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
            </svg>
          </div>
          <div>
            <h2 className="text-white font-bold text-lg leading-tight">Ujian Dihentikan Sementara</h2>
            <p className="text-red-200 text-sm">Pelanggaran terdeteksi</p>
          </div>
        </div>

        {/* Body */}
        <div className="px-6 py-6">
          <div className="bg-red-50 border border-red-200 rounded-xl p-4 mb-5">
            <p className="text-sm font-semibold text-red-800 mb-1">Alasan penghentian:</p>
            <p className="text-red-700 text-sm">{reasonLabels[reason] ?? reason}</p>
          </div>

          <div className="grid grid-cols-2 gap-3 mb-5">
            <div className="bg-slate-50 rounded-xl p-3 text-center">
              <p className="text-2xl font-bold text-slate-800">{cheatingScore}</p>
              <p className="text-xs text-slate-500 mt-0.5">Skor Pelanggaran</p>
            </div>
            <div className="bg-slate-50 rounded-xl p-3 text-center">
              <p className="text-2xl font-bold text-red-600">PAUSE</p>
              <p className="text-xs text-slate-500 mt-0.5">Status Ujian</p>
            </div>
          </div>

          <div className="bg-amber-50 border border-amber-200 rounded-xl p-4 mb-5">
            <p className="text-xs text-amber-800 leading-relaxed">
              <strong>Penting:</strong> Ujian Anda telah dihentikan sementara oleh sistem. 
              Mohon tunggu pengawas untuk melanjutkan, atau hubungi pengawas ujian segera.
              Jangan menutup browser ini.
            </p>
          </div>

          <div className="flex items-center justify-center gap-2 text-slate-400">
            <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"/>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
            </svg>
            <span className="text-sm">Menunggu persetujuan pengawas...</span>
          </div>
        </div>
      </div>
    </div>
  )
}
