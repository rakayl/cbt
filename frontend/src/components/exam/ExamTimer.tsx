import { clsx } from 'clsx'
import { useExamTimer } from '@/hooks/useExamTimer'

interface Props {
  attemptID: string | null
}

export default function ExamTimer({ attemptID }: Props) {
  const { formatted, isWarning, isDanger } = useExamTimer(attemptID)

  return (
    <div className={clsx(
      'flex items-center gap-2 px-4 py-2 rounded-xl font-mono font-semibold text-lg tabular-nums select-none transition-all duration-500',
      isDanger  && 'bg-red-100 text-red-700 animate-pulse',
      isWarning && !isDanger && 'bg-amber-50 text-amber-700',
      !isWarning && !isDanger && 'bg-slate-100 text-slate-700',
    )}>
      <svg className={clsx('w-4 h-4', isDanger && 'animate-bounce')} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
      </svg>
      {formatted}
    </div>
  )
}
