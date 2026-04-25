import { clsx } from 'clsx'
import { useExamStore } from '@/store/examStore'

interface Props {
  onSelect: (idx: number) => void
}

export default function SoalNavigator({ onSelect }: Props) {
  const { soalList, jawabans, currentIdx } = useExamStore()

  return (
    <div className="card p-4 w-64 flex-shrink-0">
      <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">
        Navigasi Soal
      </h3>

      {/* Legend */}
      <div className="flex items-center gap-3 mb-4 text-xs text-slate-500">
        <span className="flex items-center gap-1">
          <span className="w-3 h-3 rounded bg-blue-500 inline-block"/> Dijawab
        </span>
        <span className="flex items-center gap-1">
          <span className="w-3 h-3 rounded bg-slate-200 inline-block"/> Belum
        </span>
        <span className="flex items-center gap-1">
          <span className="w-3 h-3 rounded border-2 border-blue-500 inline-block"/> Aktif
        </span>
      </div>

      {/* Grid */}
      <div className="grid grid-cols-5 gap-1.5">
        {soalList.map((as, idx) => {
          const answered = !!jawabans[as.soal_id]
          const active   = idx === currentIdx
          return (
            <button
              key={as.id}
              onClick={() => onSelect(idx)}
              title={`Soal ${idx + 1}`}
              className={clsx(
                'w-full aspect-square rounded-lg text-xs font-semibold transition-all duration-100',
                'focus:outline-none focus:ring-2 focus:ring-blue-400 focus:ring-offset-1',
                active    && 'ring-2 ring-blue-500 ring-offset-1',
                answered  && !active && 'bg-blue-500 text-white hover:bg-blue-600',
                !answered && !active && 'bg-slate-100 text-slate-500 hover:bg-slate-200',
                active    && answered && 'bg-blue-600 text-white',
                active    && !answered && 'bg-white text-blue-600 border-2 border-blue-500',
              )}
            >
              {idx + 1}
            </button>
          )
        })}
      </div>

      {/* Summary */}
      <div className="mt-4 pt-4 border-t border-slate-100">
        <div className="flex justify-between text-xs">
          <span className="text-slate-500">Dijawab</span>
          <span className="font-semibold text-slate-800">
            {Object.keys(jawabans).length} / {soalList.length}
          </span>
        </div>
        <div className="mt-2 w-full h-1.5 bg-slate-100 rounded-full overflow-hidden">
          <div
            className="h-full bg-blue-500 rounded-full transition-all duration-300"
            style={{ width: `${(Object.keys(jawabans).length / Math.max(soalList.length, 1)) * 100}%` }}
          />
        </div>
      </div>
    </div>
  )
}
