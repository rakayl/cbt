import { clsx } from 'clsx'
import type { AttemptSoal } from '@/store/examStore'
import { useExamStore } from '@/store/examStore'

interface Props {
  attemptSoal: AttemptSoal
  number: number
  total: number
}

export default function SoalCard({ attemptSoal, number, total }: Props) {
  const { soal } = attemptSoal
  const { jawabans, setJawaban, attemptID, sisaDetik } = useExamStore()
  const currentAnswer = jawabans[soal.id] ?? ''

  // Parse shuffled opsi order
  let opsiList = soal.opsi ?? []
  if (attemptSoal.opsi_order) {
    try {
      const order: string[] = JSON.parse(attemptSoal.opsi_order)
      opsiList = order.map(id => soal.opsi.find(o => o.id === id)!).filter(Boolean)
    } catch { /* use original */ }
  }

  const handleSelect = (opsiID: string) => {
    setJawaban(soal.id, opsiID)
    // Autosave to API
    if (attemptID) {
      import('@/services/api').then(({ examAPI }) => {
        examAPI.saveAnswer({
          attempt_id: attemptID,
          soal_id: soal.id,
          opsi_id: opsiID,
          sisa_detik: sisaDetik,
        }).catch(() => {})
      })
    }
  }

  const handleEssayChange = (val: string) => {
    setJawaban(soal.id, val)
  }

  const handleEssaySave = () => {
    if (!attemptID) return
    import('@/services/api').then(({ examAPI }) => {
      examAPI.saveAnswer({
        attempt_id: attemptID,
        soal_id: soal.id,
        teks_jawaban: currentAnswer,
        sisa_detik: sisaDetik,
      }).catch(() => {})
    })
  }

  const letters = 'ABCDE'

  return (
    <div className="flex-1 min-h-0 overflow-y-auto">
      <div className="card p-6 animate-fade-in">
        {/* Header */}
        <div className="flex items-center justify-between mb-5">
          <span className="text-xs font-medium text-slate-500 uppercase tracking-wider">
            Soal {number} dari {total}
          </span>
          <div className="flex items-center gap-2">
            <span className={clsx(
              'badge',
              soal.tipe === 'pilihan_ganda' ? 'badge-blue' : 'badge-yellow'
            )}>
              {soal.tipe === 'pilihan_ganda' ? 'Pilihan Ganda' : 'Essay'}
            </span>
            <span className="badge badge-gray">{soal.poin} poin</span>
          </div>
        </div>

        {/* Progress bar */}
        <div className="w-full h-1 bg-slate-100 rounded-full mb-6">
          <div
            className="h-1 bg-blue-500 rounded-full transition-all duration-300"
            style={{ width: `${(number / total) * 100}%` }}
          />
        </div>

        {/* Question */}
        <div className="prose prose-sm max-w-none mb-6">
          <p className="text-slate-800 text-base leading-relaxed font-medium select-none">
            {soal.pertanyaan}
          </p>
        </div>

        {/* Answer options */}
        {soal.tipe === 'pilihan_ganda' ? (
          <div className="space-y-3">
            {opsiList.map((opsi, idx) => {
              const selected = currentAnswer === opsi.id
              return (
                <button
                  key={opsi.id}
                  onClick={() => handleSelect(opsi.id)}
                  className={clsx(
                    'w-full flex items-center gap-4 px-4 py-3.5 rounded-xl border-2 text-left',
                    'transition-all duration-150 group focus:outline-none focus:ring-2 focus:ring-blue-400',
                    selected
                      ? 'border-blue-500 bg-blue-50 shadow-sm'
                      : 'border-slate-200 bg-white hover:border-blue-300 hover:bg-blue-50/30'
                  )}
                >
                  <span className={clsx(
                    'flex-shrink-0 w-8 h-8 rounded-lg flex items-center justify-center text-sm font-bold transition-colors',
                    selected
                      ? 'bg-blue-600 text-white'
                      : 'bg-slate-100 text-slate-500 group-hover:bg-blue-100 group-hover:text-blue-700'
                  )}>
                    {letters[idx]}
                  </span>
                  <span className={clsx(
                    'text-sm transition-colors',
                    selected ? 'text-blue-800 font-medium' : 'text-slate-700'
                  )}>
                    {opsi.teks}
                  </span>
                  {selected && (
                    <svg className="ml-auto w-5 h-5 text-blue-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd"/>
                    </svg>
                  )}
                </button>
              )
            })}
          </div>
        ) : (
          <div>
            <textarea
              value={currentAnswer}
              onChange={e => handleEssayChange(e.target.value)}
              onBlur={handleEssaySave}
              rows={8}
              placeholder="Tulis jawaban Anda di sini..."
              className="w-full px-4 py-3 text-sm border-2 border-slate-200 rounded-xl focus:outline-none
                         focus:ring-2 focus:ring-blue-400 focus:border-transparent resize-none transition
                         text-slate-800 placeholder:text-slate-400 leading-relaxed"
            />
            <div className="flex justify-between mt-2">
              <span className="text-xs text-slate-400">{currentAnswer.length} karakter</span>
              <button
                onClick={handleEssaySave}
                className="text-xs text-blue-600 hover:text-blue-800 font-medium transition-colors"
              >
                Simpan
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
