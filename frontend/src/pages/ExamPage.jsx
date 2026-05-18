import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Camera, CheckCircle2, ChevronLeft, ChevronRight, Maximize2, ShieldAlert, WifiOff } from 'lucide-react';
import { useNavigate, useParams } from 'react-router-dom';
import { api } from '../lib/api';
import { getDeviceFingerprint, getDeviceName } from '../lib/deviceFingerprint';
import { connectExamSocket } from '../lib/examSocket';
import { getApiErrorDetail, getApiErrorMessage } from '../lib/apiError';
import { listOffline, putOffline, queueSync, deleteOffline } from '../lib/indexedDb';
import { useAuthStore } from '../stores/authStore';

const criticalSnapshotEvents = new Set(['fullscreen_exit', 'tab_switch', 'multiple_face', 'no_face', 'devtools_suspected', 'multiple_monitor', 'audio_noise']);

export default function ExamPage() {
  const { sessionId } = useParams();
  const navigate = useNavigate();
  const [seconds, setSeconds] = useState(0);
  const [offline, setOffline] = useState(!navigator.onLine);
  const [events, setEvents] = useState([]);
  const [socketStatus, setSocketStatus] = useState('disconnected');
  const [recoveryState, setRecoveryState] = useState(null);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [answers, setAnswers] = useState({});
  const [autoSubmitNotice, setAutoSubmitNotice] = useState('');
  const [submitConfirmOpen, setSubmitConfirmOpen] = useState(false);
  const [timerReady, setTimerReady] = useState(false);
  const seq = useRef(Number(localStorage.getItem('exam.clientSeq') || 0));
  const autoSubmitTriggeredRef = useRef(false);
  const hasSeenPositiveTimerRef = useRef(false);
  const videoRef = useRef(null);
  const streamRef = useRef(null);
  const lastFaceDetectedAt = useRef(0);
  const serverSyncRef = useRef({ remaining: 0, receivedAt: Date.now(), paused: false });
  const accessToken = useAuthStore((state) => state.accessToken);
  const tenantId = useAuthStore((state) => state.tenantId);
  const authHeaders = useMemo(() => ({ Authorization: `Bearer ${accessToken}`, 'X-Tenant-ID': tenantId || '' }), [accessToken, tenantId]);

  useEffect(() => {
    window.desktopApp?.enterExamMode?.();
    return () => {
      window.desktopApp?.exitExamMode?.();
    };
  }, []);

  const examQuery = useQuery({
    queryKey: ['exam-session-questions', sessionId],
    enabled: Boolean(sessionId && accessToken),
    queryFn: async () => {
      const { data } = await api.get(`/exam-sessions/${sessionId}/questions`);
      return data.data;
    },
  });

  const questions = examQuery.data?.questions || [];
  const currentQuestion = questions[selectedIndex] || null;
  const canGoPrevious = selectedIndex > 0;
  const canGoNext = selectedIndex < questions.length - 1;
  const sessionCompleted = examQuery.data?.status === 'completed';
  const answeredCount = useMemo(() => questions.filter((question) => isAnsweredWithMap(question, answers)).length, [answers, questions]);
  const unansweredCount = Math.max(0, questions.length - answeredCount);

  const clock = useMemo(() => {
    const safeSeconds = Math.max(0, Math.floor(Number(seconds) || 0));
    const h = String(Math.floor(safeSeconds / 3600)).padStart(2, '0');
    const m = String(Math.floor((safeSeconds % 3600) / 60)).padStart(2, '0');
    const s = String(safeSeconds % 60).padStart(2, '0');
    return `${h}:${m}:${s}`;
  }, [seconds]);

  const syncTimerFromServer = useCallback((payload = {}) => {
    const completed = payload.status === 'completed';
    const paused = completed || boolValue(payload.timer_paused) || payload.recovery_status === 'paused';
    const nextRemaining = completed ? 0 : Math.max(0, Math.floor(Number(payload.remaining_seconds ?? 0)));
    if (nextRemaining > 0) hasSeenPositiveTimerRef.current = true;

    setSeconds((current) => {
      const currentValue = Math.max(0, Math.floor(Number(current) || 0));
      const drift = Math.abs(currentValue - nextRemaining);
      const displayValue = paused || currentValue === 0 || drift > 1
        ? nextRemaining
        : Math.min(currentValue, nextRemaining);

      serverSyncRef.current = {
        remaining: displayValue,
        receivedAt: Date.now(),
        paused,
      };

      return currentValue === displayValue ? current : displayValue;
    });
  }, []);

  useEffect(() => {
    const ticker = setInterval(() => {
      const sync = serverSyncRef.current;
      if (!sync) return;

      if (sync.paused || sessionCompleted) {
        setSeconds((current) => {
          const next = Math.max(0, Math.floor(sync.remaining));
          return current === next ? current : next;
        });
        return;
      }

      const elapsed = Math.floor((Date.now() - sync.receivedAt) / 1000);
      const next = Math.max(0, Math.floor(sync.remaining - elapsed));
      setSeconds((current) => (current === next ? current : next));
    }, 250);

    return () => clearInterval(ticker);
  }, [sessionCompleted]);

  const flushQueues = useCallback(async () => {
    const queued = await listOffline('syncQueue');
    for (const item of queued) {
      try {
        await api.post(item.endpoint, item.payload);
        await deleteOffline('syncQueue', item.id);
      } catch {
        break;
      }
    }
    const reconnect = await api.post(`/exam-sessions/${sessionId}/reconnect`, {
      last_client_seq: seq.current,
      device_fingerprint: getDeviceFingerprint(),
      device_name: getDeviceName(),
      user_agent: navigator.userAgent,
    }).catch(() => null);
    if (reconnect?.data?.data?.remaining_seconds !== undefined) {
      syncTimerFromServer(reconnect.data.data);
      setRecoveryState(reconnect.data.data);
    }
  }, [sessionId, syncTimerFromServer]);

  const uploadSnapshot = useCallback(async (eventType = 'webcam_snapshot', severity = 'low', metadata = {}) => {
    const video = videoRef.current;
    if (!sessionId || !video || video.readyState < 2 || !video.videoWidth || !video.videoHeight) return;
    const canvas = document.createElement('canvas');
    const maxWidth = 640;
    const ratio = Math.min(1, maxWidth / video.videoWidth);
    canvas.width = Math.max(1, Math.round(video.videoWidth * ratio));
    canvas.height = Math.max(1, Math.round(video.videoHeight * ratio));
    const context = canvas.getContext('2d');
    context.drawImage(video, 0, 0, canvas.width, canvas.height);
    const payload = {
      exam_session_id: sessionId,
      event_type: eventType,
      severity,
      image_data: canvas.toDataURL('image/jpeg', 0.65),
      face_count: Number.isFinite(metadata.face_count) ? Number(metadata.face_count) : undefined,
      metadata: { ...metadata, capture_width: canvas.width, capture_height: canvas.height, user_agent: navigator.userAgent },
      client_time: new Date().toISOString(),
    };
    if (!navigator.onLine) {
      await queueSync('/proctoring/snapshots', payload, authHeaders);
      setEvents((current) => [{
        id: crypto.randomUUID(),
        event_type: 'webcam_snapshot_queued',
        severity,
        client_time: payload.client_time,
      }, ...current].slice(0, 10));
      return;
    }
    try {
      await api.post('/proctoring/snapshots', payload);
      setEvents((current) => [{
        id: crypto.randomUUID(),
        event_type: 'webcam_snapshot',
        severity,
        client_time: payload.client_time,
      }, ...current].slice(0, 10));
    } catch {
      await queueSync('/proctoring/snapshots', payload, authHeaders);
      setEvents((current) => [{
        id: crypto.randomUUID(),
        event_type: 'webcam_snapshot_queued',
        severity,
        client_time: payload.client_time,
      }, ...current].slice(0, 10));
    }
  }, [authHeaders, sessionId]);

  const recordEvent = useCallback(async (eventType, severity = 'medium', metadata = {}) => {
    const event = {
      id: crypto.randomUUID(),
      exam_session_id: sessionId,
      event_type: eventType,
      severity,
      metadata: { ...metadata, user_agent: navigator.userAgent, screen_count_hint: window.screen?.isExtended ? 2 : 1 },
      client_time: new Date().toISOString(),
    };
    setEvents((current) => [event, ...current].slice(0, 10));
    await putOffline('proctoringEvents', event);
    if (!navigator.onLine) {
      await queueSync('/proctoring/events', event, authHeaders);
    } else {
      try {
        await api.post('/proctoring/events', event);
      } catch {
        await queueSync('/proctoring/events', event, authHeaders);
      }
    }
    if (severity === 'critical' || criticalSnapshotEvents.has(eventType)) {
      await uploadSnapshot(eventType, severity, metadata);
    }
  }, [authHeaders, sessionId, uploadSnapshot]);

  const autosave = useCallback(async (question, payload) => {
    if (!question?.question_id) return;
    seq.current += 1;
    localStorage.setItem('exam.clientSeq', String(seq.current));
    await putOffline('answers', { id: `${sessionId}:${question.question_id}`, session_id: sessionId, question_id: question.question_id, payload, client_seq: seq.current });
    const request = { session_id: sessionId, question_id: question.question_id, payload, client_seq: seq.current };
    if (!navigator.onLine) {
      await queueSync('/exam-sessions/autosave', request, authHeaders);
      return;
    }
    try {
      await api.post('/exam-sessions/autosave', request);
    } catch {
      await queueSync('/exam-sessions/autosave', request, authHeaders);
    }
  }, [authHeaders, sessionId]);

  useEffect(() => {
    if (!examQuery.data) return;
    syncTimerFromServer(examQuery.data);
    setTimerReady(true);
    setRecoveryState(examQuery.data);
    const initialAnswers = {};
    for (const question of examQuery.data.questions || []) {
      initialAnswers[question.question_id] = question.answer_payload || {};
    }
    setAnswers(initialAnswers);
  }, [examQuery.data, syncTimerFromServer]);

  useEffect(() => {
    if (selectedIndex >= questions.length && questions.length > 0) {
      setSelectedIndex(0);
    }
  }, [questions.length, selectedIndex]);

  function updateAnswer(question, payload) {
    if (sessionCompleted) return;
    setAnswers((current) => ({ ...current, [question.question_id]: payload }));
    autosave(question, payload);
  }

  function toggleOption(question, optionID) {
    const current = answers[question.question_id] || {};
    if (question.answer_mode === 'multiple') {
      const selected = new Set(current.selected_option_ids || []);
      if (selected.has(optionID)) selected.delete(optionID);
      else selected.add(optionID);
      updateAnswer(question, { selected_option_ids: Array.from(selected) });
      return;
    }
    updateAnswer(question, { selected_option_ids: [optionID] });
  }

  function isAnswered(question) {
    return isAnsweredWithMap(question, answers);
  }

  const submitExam = useMutation({
    mutationFn: async () => (await api.post(`/exam-sessions/${sessionId}/submit`, { client_seq: seq.current })).data.data,
    onSuccess: (data) => {
      setSubmitConfirmOpen(false);
      navigate(`/student/history/${data?.session_id || sessionId}`, { replace: true });
    },
    onError: (_error, variables) => {
      if (!variables?.automatic) {
        autoSubmitTriggeredRef.current = false;
        return;
      }
      setAutoSubmitNotice('Submit otomatis belum berhasil. Sistem akan mencoba lagi saat koneksi pulih, atau hubungi pengawas.');
    },
  });

  const autoSubmitExpiredExam = useCallback(async (reason = 'timer_finished') => {
    if (autoSubmitTriggeredRef.current || sessionCompleted || !sessionId || submitExam.isPending) return;
    autoSubmitTriggeredRef.current = true;
    setAutoSubmitNotice('Waktu ujian telah selesai. Sistem sedang menyimpan dan mengirim jawaban...');
    try {
      await flushQueues();
    } catch {
      // Submit tetap dicoba; backend adalah sumber kebenaran status dan waktu ujian.
    }
    try {
      await recordEvent('auto_submit_timer_finished', 'medium', { reason });
    } catch {
      // Event anti-cheat tidak boleh menghambat submit final.
    }
    submitExam.mutate({ automatic: true });
  }, [flushQueues, recordEvent, sessionCompleted, sessionId, submitExam]);

  useEffect(() => {
    if (!timerReady || !examQuery.data || sessionCompleted || submitExam.isSuccess) return;
    if (seconds > 0) return;
    if (!hasSeenPositiveTimerRef.current && examQuery.data.status !== 'started' && examQuery.data.status !== 'reconnecting') return;
    if (!hasSeenPositiveTimerRef.current && Number(examQuery.data.remaining_seconds || 0) > 0) return;
    if (offline) {
      setAutoSubmitNotice('Waktu ujian telah selesai. Menunggu koneksi untuk submit otomatis...');
      return;
    }
    autoSubmitExpiredExam('countdown_zero');
  }, [autoSubmitExpiredExam, examQuery.data, offline, seconds, sessionCompleted, submitExam.isSuccess, timerReady]);

  useEffect(() => {
    if (sessionCompleted) {
      navigate(`/student/history/${sessionId}`, { replace: true });
    }
  }, [navigate, sessionCompleted, sessionId]);

  useEffect(() => {
    if (!offline && timerReady && autoSubmitNotice && seconds <= 0 && examQuery.data && !sessionCompleted && !submitExam.isSuccess) {
      autoSubmitExpiredExam('online_after_countdown_zero');
    }
  }, [autoSubmitExpiredExam, autoSubmitNotice, examQuery.data, offline, seconds, sessionCompleted, submitExam.isSuccess, timerReady]);

  function handleSubmitExam() {
    if (sessionCompleted || submitExam.isPending || questions.length === 0) return;
    setSubmitConfirmOpen(true);
  }

  function confirmSubmitExam() {
    submitExam.mutate({ automatic: false });
  }

  function goPrevious() {
    setSelectedIndex((value) => Math.max(0, value - 1));
  }

  function goNext() {
    setSelectedIndex((value) => Math.min(Math.max(0, questions.length - 1), value + 1));
  }

  useEffect(() => {
    const socket = connectExamSocket({
      sessionId,
      token: accessToken,
      onStatus: setSocketStatus,
      onMessage: (message) => {
        if (message.type === 'timer.tick' && message.payload?.remaining_seconds !== undefined) {
          syncTimerFromServer(message.payload);
          setRecoveryState((current) => ({ ...(current || {}), ...message.payload }));
        }
      },
    });
    return () => socket.close();
  }, [accessToken, sessionId, syncTimerFromServer]);

  useEffect(() => {
    const onOnline = async () => {
      setOffline(false);
      await flushQueues();
    };
    const onOffline = () => {
      setOffline(true);
      recordEvent('abnormal_reconnect', 'high');
    };
    const onVisibility = () => {
      if (document.hidden) recordEvent('tab_switch', 'high');
    };
    const onFullscreen = () => {
      if (!document.fullscreenElement) recordEvent('fullscreen_exit', 'high');
    };
    const blocked = (eventType) => (event) => {
      event.preventDefault();
      recordEvent(eventType, 'medium');
    };
    const copyPaste = blocked('copy_paste');
    const contextMenu = blocked('right_click');
    const devToolsProbe = setInterval(() => {
      const widthGap = window.outerWidth - window.innerWidth;
      const heightGap = window.outerHeight - window.innerHeight;
      if (widthGap > 180 || heightGap > 180) recordEvent('devtools_suspected', 'high', { width_gap: widthGap, height_gap: heightGap });
      if (window.screen?.isExtended) recordEvent('multiple_monitor', 'high');
    }, 10000);
    let mediaStream;
    let audioContext;
    let audioProbe;
    let faceProbe;
    navigator.mediaDevices?.getUserMedia?.({ video: true, audio: true }).then((stream) => {
      mediaStream = stream;
      streamRef.current = stream;
      if (videoRef.current) {
        videoRef.current.srcObject = stream;
        videoRef.current.play().catch(() => null);
      }
      recordEvent('webcam_ready', 'low', { video: true, audio: true });
      uploadSnapshot('webcam_snapshot', 'low', { trigger: 'camera_ready' });
      if ('FaceDetector' in window) {
        const video = document.createElement('video');
        video.muted = true;
        video.srcObject = stream;
        video.play().catch(() => null);
        const detector = new window.FaceDetector({ fastMode: true });
        faceProbe = setInterval(async () => {
          try {
            const faces = await detector.detect(video);
            if (faces.length === 0) recordEvent('no_face', 'high', { face_count: 0 });
            if (faces.length === 1 && Date.now() - lastFaceDetectedAt.current > 30000) {
              lastFaceDetectedAt.current = Date.now();
              recordEvent('face_detected', 'low', { face_count: 1 });
            }
            if (faces.length > 1) recordEvent('multiple_face', 'critical', { face_count: faces.length });
          } catch {
            clearInterval(faceProbe);
          }
        }, 7000);
      } else {
        recordEvent('face_detection_unavailable', 'low', {
          supported: false,
          reason: 'native_face_detector_not_supported_by_browser',
        });
      }
      audioContext = new AudioContext();
      const analyser = audioContext.createAnalyser();
      audioContext.createMediaStreamSource(stream).connect(analyser);
      const data = new Uint8Array(analyser.frequencyBinCount);
      audioProbe = setInterval(() => {
        analyser.getByteFrequencyData(data);
        const average = data.reduce((sum, value) => sum + value, 0) / Math.max(1, data.length);
        if (average > 80) recordEvent('audio_noise', 'medium', { average_volume: average });
      }, 8000);
    }).catch(() => recordEvent('webcam_unavailable', 'high'));
    const snapshotProbe = setInterval(() => {
      uploadSnapshot('webcam_snapshot', 'low', { trigger: 'periodic' });
    }, 60000);

    addEventListener('online', onOnline);
    addEventListener('offline', onOffline);
    document.addEventListener('visibilitychange', onVisibility);
    document.addEventListener('fullscreenchange', onFullscreen);
    document.addEventListener('copy', copyPaste);
    document.addEventListener('paste', copyPaste);
    document.addEventListener('contextmenu', contextMenu);
    flushQueues();
    return () => {
      removeEventListener('online', onOnline);
      removeEventListener('offline', onOffline);
      document.removeEventListener('visibilitychange', onVisibility);
      document.removeEventListener('fullscreenchange', onFullscreen);
      document.removeEventListener('copy', copyPaste);
      document.removeEventListener('paste', copyPaste);
      document.removeEventListener('contextmenu', contextMenu);
      clearInterval(devToolsProbe);
      clearInterval(snapshotProbe);
      clearInterval(audioProbe);
      clearInterval(faceProbe);
      mediaStream?.getTracks().forEach((track) => track.stop());
      streamRef.current = null;
      audioContext?.close?.();
    };
  }, [flushQueues, recordEvent, uploadSnapshot]);

  return (
    <main className="min-h-screen bg-field">
      <header className="h-16 bg-white border-b border-line px-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <ShieldAlert size={20} />
          <b>Secure Exam</b>
          <span className="text-xs px-2 py-1 rounded bg-field border border-line">{socketStatus}</span>
        </div>
        <span className="font-mono text-xl">{clock}</span>
      </header>
      <video ref={videoRef} className="pointer-events-none fixed h-1 w-1 opacity-0" muted playsInline />
      {offline && <div className="bg-warn text-white px-6 py-2 flex items-center gap-2"><WifiOff size={18} /> Offline. Jawaban dan event pengawasan disimpan IndexedDB dan akan disinkronkan. Timer resmi akan dipulihkan dari server saat reconnect.</div>}
      {autoSubmitNotice ? (
        <div className="bg-[#fff8dd] px-6 py-2 text-sm font-semibold text-[#7e5b00]">
          {autoSubmitNotice}
        </div>
      ) : null}
      {recoveryState?.timer_paused || recoveryState?.recovery_status === 'paused' ? (
        <div className="bg-[#f1faff] px-6 py-2 text-sm font-semibold text-[#009ef7]">
          Recovery aktif: timer server sedang dipause pada {clock}. Ujian akan lanjut dari sisa waktu ini setelah koneksi kembali.
        </div>
      ) : null}
      {recoveryState?.review_required || recoveryState?.recovery_status === 'requires_review' ? (
        <div className="bg-[#fff5f8] px-6 py-2 text-sm font-semibold text-[#f1416c]">
          Sesi membutuhkan review proctor/admin karena batas recovery, reconnect, atau perubahan device terdeteksi.
        </div>
      ) : null}
      <section className="p-6 grid md:grid-cols-[1fr_280px] gap-4">
        <article className="panel p-5 min-h-96">
          <div className="flex items-center justify-between gap-3">
            <h1 className="font-bold text-xl">
              Question {currentQuestion?.position || selectedIndex + 1}
              {currentQuestion?.question_tag_name ? <span className="text-[#7e8299]"> -- {currentQuestion.question_tag_name}</span> : null}
            </h1>
            <button className="btn btn-ghost" onClick={() => document.documentElement.requestFullscreen?.()}>
              <Maximize2 size={18} /> Fullscreen
            </button>
          </div>
          {examQuery.isLoading ? (
            <p className="mt-4 text-slate-500">Memuat soal...</p>
          ) : currentQuestion ? (
            <>
              <p className="mt-4 whitespace-pre-wrap text-lg leading-8">{currentQuestion.text}</p>
              <ExamMediaGrid media={currentQuestion.media || []} />
              {currentQuestion.options?.length ? (
                <div className="mt-5 space-y-3">
                  {currentQuestion.options.map((option) => {
                    const selected = (answers[currentQuestion.question_id]?.selected_option_ids || []).includes(option.id);
                    return (
                      <button
                        key={option.id}
                        className={selected ? 'flex w-full items-start gap-3 rounded-lg border border-[#009ef7] bg-[#f1faff] p-4 text-left font-semibold text-[#181c32]' : 'flex w-full items-start gap-3 rounded-lg border border-line bg-white p-4 text-left font-semibold text-[#181c32]'}
                        disabled={sessionCompleted}
                        onClick={() => toggleOption(currentQuestion, option.id)}
                      >
                        <span className={selected ? 'grid h-8 w-8 shrink-0 place-items-center rounded-lg bg-[#009ef7] text-white' : 'grid h-8 w-8 shrink-0 place-items-center rounded-lg bg-field text-[#3f4254]'}>
                          {option.label}
                        </span>
                        <span className="min-w-0 flex-1 pt-1">
                          {option.text ? <span className="block">{option.text}</span> : null}
                          <ExamMediaGrid media={option.media || []} compact />
                        </span>
                      </button>
                    );
                  })}
                </div>
              ) : (
                <textarea
                  className="input mt-4 min-h-48"
                  placeholder="Answer"
                  disabled={sessionCompleted}
                  value={answers[currentQuestion.question_id]?.text || ''}
                  onChange={(event) => updateAnswer(currentQuestion, { text: event.target.value })}
                />
              )}
              <div className="mt-6 flex flex-col gap-3 border-t border-line pt-5 sm:flex-row sm:items-center sm:justify-between">
                <button className="btn btn-ghost justify-center" disabled={!canGoPrevious} onClick={goPrevious}>
                  <ChevronLeft size={18} />
                  Previous
                </button>
                <div className="text-center text-sm font-semibold text-slate-500">
                  {selectedIndex + 1} dari {questions.length} soal
                </div>
                <button className="btn btn-primary justify-center" disabled={!canGoNext} onClick={goNext}>
                  Next
                  <ChevronRight size={18} />
                </button>
              </div>
              <div className="mt-4 rounded-lg border border-line bg-field p-4">
                <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <div className="text-sm font-semibold text-slate-600">
                    Terjawab {answeredCount} dari {questions.length} soal. Belum dijawab {unansweredCount}.
                  </div>
                  <button className="btn btn-primary justify-center" disabled={sessionCompleted || submitExam.isPending || questions.length === 0} onClick={handleSubmitExam}>
                    <CheckCircle2 size={18} />
                    {submitExam.isPending ? 'Submitting...' : sessionCompleted ? 'Sudah Submitted' : 'Submit Ujian'}
                  </button>
                </div>
                {submitExam.error ? (
                  <div className="mt-3 rounded-lg bg-[#fff5f8] px-4 py-3 text-sm font-bold text-[#f1416c]">
                    <div>{getApiErrorMessage(submitExam.error)}</div>
                    {getApiErrorDetail(submitExam.error) ? <div className="mt-1 text-xs font-semibold opacity-80">{getApiErrorDetail(submitExam.error)}</div> : null}
                  </div>
                ) : null}
              </div>
            </>
          ) : (
            <p className="mt-4 text-slate-500">Belum ada soal pada sesi ini.</p>
          )}
        </article>
        <aside className="space-y-4">
          <div className="panel p-4 grid grid-cols-5 gap-2 content-start">
            {(questions.length ? questions : Array.from({ length: 40 }, (_, i) => ({ position: i + 1 }))).map((question, i) => (
              <button
                key={question.session_question_id || i}
                className={
                  i === selectedIndex
                    ? 'btn btn-primary justify-center'
                    : question.question_id && isAnswered(question)
                      ? 'btn justify-center bg-[#e8fff3] text-[#50cd89]'
                      : 'btn btn-ghost justify-center'
                }
                onClick={() => setSelectedIndex(i)}
              >
                {question.position || i + 1}
              </button>
            ))}
          </div>
          <div className="panel p-4">
            <h2 className="font-semibold">Activity</h2>
            <div className="mt-3 rounded-lg bg-field p-3 text-xs font-semibold text-slate-600">
              <div>Recovery: {recoveryState?.recovery_status || 'normal'}</div>
              <div>Reconnect: {recoveryState?.reconnect_count || 0}</div>
              <div>Pause: {Math.round(Number(recoveryState?.total_pause_seconds || 0) / 60)} menit</div>
              <div>Risk score: {Number(recoveryState?.suspicious_score || 0).toFixed(0)}</div>
            </div>
            <div className="mt-3 space-y-2 text-sm">
              {events.length === 0 && <p className="text-slate-500">No suspicious activity.</p>}
              {events.map((event) => (
                <div key={event.id} className="flex justify-between gap-2">
                  <span className="inline-flex items-center gap-2">
                    {event.event_type?.includes('snapshot') ? <Camera size={14} /> : null}
                    {event.event_type}
                  </span>
                  <span>{event.severity}</span>
                </div>
              ))}
            </div>
          </div>
        </aside>
      </section>
      {submitConfirmOpen ? (
        <div className="fixed inset-0 z-50 grid place-items-center bg-[#181c32]/40 px-4">
          <div className="w-full max-w-md overflow-hidden rounded-lg bg-white shadow-2xl">
            <div className="border-b border-line px-6 py-5">
              <h2 className="text-lg font-extrabold text-[#181c32]">Submit Ujian?</h2>
              <p className="mt-2 text-sm font-semibold leading-6 text-[#7e8299]">
                Setelah disubmit, jawaban tidak bisa diubah lagi.
              </p>
            </div>
            <div className="grid grid-cols-3 gap-3 px-6 py-5 text-center">
              <div className="rounded-lg bg-field p-3">
                <div className="text-xs font-bold uppercase text-[#a1a5b7]">Total</div>
                <div className="mt-1 text-xl font-extrabold text-[#181c32]">{questions.length}</div>
              </div>
              <div className="rounded-lg bg-[#e8fff3] p-3">
                <div className="text-xs font-bold uppercase text-[#50cd89]">Terjawab</div>
                <div className="mt-1 text-xl font-extrabold text-[#181c32]">{answeredCount}</div>
              </div>
              <div className="rounded-lg bg-[#fff5f8] p-3">
                <div className="text-xs font-bold uppercase text-[#f1416c]">Kosong</div>
                <div className="mt-1 text-xl font-extrabold text-[#181c32]">{unansweredCount}</div>
              </div>
            </div>
            <div className="flex flex-col-reverse gap-3 border-t border-line px-6 py-5 sm:flex-row sm:justify-end">
              <button className="btn btn-ghost justify-center" disabled={submitExam.isPending} onClick={() => setSubmitConfirmOpen(false)}>
                Tidak
              </button>
              <button className="btn btn-primary justify-center" disabled={submitExam.isPending} onClick={confirmSubmitExam}>
                <CheckCircle2 size={18} />
                {submitExam.isPending ? 'Submitting...' : 'Ya, Submit'}
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </main>
  );
}

function isAnsweredWithMap(question, answers) {
  if (!question?.question_id) return false;
  const answer = answers[question.question_id] || {};
  return Boolean(answer.text?.trim?.() || answer.selected_option_ids?.length);
}

function boolValue(value) {
  return value === true || value === 'true' || value === 1 || value === '1';
}

function ExamMediaGrid({ media = [], compact = false }) {
  if (!media.length) return null;
  return (
    <div className={compact ? 'mt-3 grid gap-2 sm:grid-cols-2' : 'mt-5 grid gap-3 md:grid-cols-2'}>
      {media.map((item) => (
        <ExamSecureImage key={item.id} media={item} compact={compact} />
      ))}
    </div>
  );
}

function ExamSecureImage({ media, compact }) {
  const [src, setSrc] = useState('');
  useEffect(() => {
    let active = true;
    let objectUrl = '';
    api.get(media.url, { responseType: 'blob' }).then((response) => {
      if (!active) return;
      objectUrl = URL.createObjectURL(response.data);
      setSrc(objectUrl);
    }).catch(() => {
      if (active) setSrc('');
    });
    return () => {
      active = false;
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [media.url]);

  if (!src) {
    return <div className={compact ? 'grid h-28 place-items-center rounded-lg bg-[#f5f8fa] text-xs text-[#a1a5b7]' : 'grid h-56 place-items-center rounded-lg bg-[#f5f8fa] text-sm text-[#a1a5b7]'}>Memuat gambar...</div>;
  }
  if (compact) {
    return (
      <span className="block overflow-hidden rounded-lg border border-line bg-[#f5f8fa]">
        <img className="h-32 w-full object-contain" src={src} alt="Media opsi jawaban" />
      </span>
    );
  }
  return (
    <a href={src} target="_blank" rel="noreferrer" className="block overflow-hidden rounded-lg border border-line bg-[#f5f8fa]">
      <img className="h-72 w-full object-contain" src={src} alt="Media soal" />
    </a>
  );
}
