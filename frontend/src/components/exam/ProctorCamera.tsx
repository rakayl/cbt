import { clsx } from 'clsx'

interface Props {
  videoRef: React.RefObject<HTMLVideoElement>
  canvasRef: React.RefObject<HTMLCanvasElement>
  violation?: string
}

export default function ProctorCamera({ videoRef, canvasRef, violation }: Props) {
  return (
    <div className="relative">
      <div className={clsx(
        'relative w-32 h-24 rounded-xl overflow-hidden border-2 transition-all duration-300',
        violation ? 'border-red-500 shadow-lg shadow-red-500/30' : 'border-slate-300'
      )}>
        <video
          ref={videoRef}
          autoPlay muted playsInline
          className="w-full h-full object-cover scale-x-[-1]"
        />

        {/* Recording indicator */}
        <div className="absolute top-1.5 right-1.5 flex items-center gap-1">
          <span className="w-1.5 h-1.5 rounded-full bg-red-500 animate-pulse"/>
        </div>

        {/* Violation overlay */}
        {violation && (
          <div className="absolute inset-0 bg-red-500/20 flex items-center justify-center">
            <svg className="w-8 h-8 text-red-600" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd"/>
            </svg>
          </div>
        )}
      </div>

      {/* Hidden canvas for frame capture */}
      <canvas ref={canvasRef} className="hidden" />

      {/* Status label */}
      <p className={clsx(
        'text-center text-xs mt-1 font-medium',
        violation ? 'text-red-600' : 'text-slate-400'
      )}>
        {violation
          ? violation === 'no_face'        ? 'Wajah tidak terdeteksi'
          : violation === 'multiple_faces' ? 'Banyak wajah!'
          : violation === 'face_mismatch'  ? 'Wajah berbeda!'
          : 'Pelanggaran terdeteksi'
          : 'Proctoring aktif'}
      </p>
    </div>
  )
}
