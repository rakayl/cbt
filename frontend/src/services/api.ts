import axios from 'axios'

const api = axios.create({ baseURL: '/api/v1', headers: { 'Content-Type': 'application/json' } })

api.interceptors.request.use(cfg => {
  const token = localStorage.getItem('cbt_token')
  if (token) cfg.headers.Authorization = `Bearer ${token}`
  return cfg
})
api.interceptors.response.use(r => r, err => {
  if (err.response?.status === 401) { localStorage.removeItem('cbt_token'); window.location.href = '/login' }
  return Promise.reject(err)
})

export default api

export const authAPI = {
  login:    (email: string, password: string) => api.post('/auth/login', { email, password }),
  register: (data: Record<string, unknown>)   => api.post('/auth/register', data),
}

export const ujianAPI = {
  list:       ()                                         => api.get('/ujian'),
  get:        (id: string)                               => api.get(`/ujian/${id}`),
  create:     (data: Record<string, unknown>)            => api.post('/ujian', data),
  update:     (id: string, data: Record<string, unknown>)=> api.put(`/ujian/${id}`, data),
  setStatus:  (id: string, status: string)               => api.put(`/ujian/${id}/status`, { status }),
  addSoal:    (id: string, soalID: string, urutan: number) => api.post(`/ujian/${id}/soal`, { soal_id: soalID, urutan }),
  addPeserta: (id: string, pesertaID: string)            => api.post(`/ujian/${id}/peserta`, { peserta_id: pesertaID }),
}

export const examAPI = {
  start:           (ujianID: string) => api.post('/exam/start', { ujian_id: ujianID }),
  saveAnswer:      (p: { attempt_id:string;soal_id:string;opsi_id?:string;teks_jawaban?:string;sisa_detik:number }) => api.post('/exam/answer', p),
  finish:          (attemptID: string) => api.post('/exam/finish', { attempt_id: attemptID }),
  reportViolation: (attemptID: string, eventType: string, detail?: string) => api.post('/exam/violation', { attempt_id: attemptID, event_type: eventType, detail }),
  sendFrame:       (attemptID: string, imageBase64: string, baseEmbedding?: number[]) => api.post('/exam/proctor', { attempt_id: attemptID, image_base64: imageBase64, base_embedding: baseEmbedding }),
  getAttempt:      (id: string) => api.get(`/exam/attempt/${id}`),
}

export const adminAPI = {
  getOnline:  ()              => api.get('/admin/online'),
  getResults: (ujianID: string) => api.get(`/admin/ujian/${ujianID}/results`),
  unpause:    (attemptID: string) => api.post(`/admin/attempt/${attemptID}/unpause`),
}
