export function getApiErrorPayload(error) {
  return error?.response?.data || {};
}

export function getApiFieldErrors(error) {
  const payload = getApiErrorPayload(error);
  const fields = payload.errors?.fields || {};
  return typeof fields === 'object' && fields !== null ? fields : {};
}

export function getApiErrorMessage(error, fallback = 'Terjadi kesalahan. Periksa kembali data yang dikirim.') {
  const payload = getApiErrorPayload(error);
  const errors = payload.errors;
  if (errors?.message) return errors.message;
  if (typeof errors === 'string') return errors;
  if (payload.message) return payload.message;
  if (error?.message) return error.message;
  return fallback;
}

export function getApiErrorDetail(error) {
  const payload = getApiErrorPayload(error);
  const errors = payload.errors;
  if (!errors || typeof errors === 'string') return '';
  return errors.safe_detail || '';
}

export function applyApiFieldErrors(error, setFieldError, fieldAliases = {}) {
  const fields = getApiFieldErrors(error);
  Object.entries(fields).forEach(([field, message]) => {
    const target = fieldAliases[field] || field;
    setFieldError(target, { type: 'server', message: String(message) });
  });
}
