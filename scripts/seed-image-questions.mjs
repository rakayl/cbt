const baseUrl = process.env.BASE_URL || 'http://localhost';
const tenantId = process.env.TENANT_ID || '10000000-0000-0000-0000-000000000001';
const email = process.env.SEED_EMAIL || 'admin@example.edu';
const password = process.env.SEED_PASSWORD || 'ChangeMe123!';

const pngBase64 =
  'iVBORw0KGgoAAAANSUhEUgAAAEAAAABACAYAAACqaXHeAAAAPElEQVR4nO3PsQ0AIBDAMMC/5+ONAvZoFSzZnZ2ZAAAAAAAAAAAAAPAH21wAAADgGyAAAABgGyAAAABgG3gBNxQB9xzql2YAAAAASUVORK5CYII=';

function apiPath(path) {
  return `${baseUrl}${path}`;
}

async function json(method, path, body, token) {
  const response = await fetch(apiPath(path), {
    method,
    headers: {
      'Content-Type': 'application/json',
      'X-Tenant-ID': tenantId,
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(`${method} ${path} failed: ${payload.message || response.statusText} ${payload.detail || ''}`);
  }
  return payload.data;
}

async function uploadMedia(questionId, optionId, usageType, filename, token) {
  const bytes = Uint8Array.from(Buffer.from(pngBase64, 'base64'));
  const form = new FormData();
  form.append('usage_type', usageType);
  if (optionId) form.append('option_id', optionId);
  form.append('file', new Blob([bytes], { type: 'image/png' }), filename);
  const response = await fetch(apiPath(`/api/v1/questions/${questionId}/media`), {
    method: 'POST',
    headers: {
      'X-Tenant-ID': tenantId,
      Authorization: `Bearer ${token}`,
    },
    body: form,
  });
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(`media upload failed: ${payload.message || response.statusText} ${payload.detail || ''}`);
  }
  return payload.data;
}

async function firstOrCreate(path, search, body, token) {
  const list = await json('GET', `${path}/?page=1&limit=50&search=${encodeURIComponent(search)}`, null, token);
  const existing = (list.items || []).find((item) => String(item.code).toLowerCase() === body.code.toLowerCase());
  if (existing) return existing;
  return json('POST', `${path}/`, body, token);
}

async function main() {
  const login = await json('POST', '/api/v1/auth/login', {
    email,
    password,
    tenant_id: tenantId,
    device_name: 'seed-image-questions',
    fingerprint: 'seed-image-questions',
  });
  const token = login.access_token;
  const lecturerList = await json('GET', '/api/v1/lecturers/?page=1&limit=1', null, token);
  const lecturerId = lecturerList.items?.[0]?.id;
  if (!lecturerId) {
    throw new Error('seed requires at least one lecturer. Create a guru account first, then rerun this script.');
  }

  const bank = await firstOrCreate('/api/v1/question-banks', 'DEMO-IMG-BANK', {
    code: 'DEMO-IMG-BANK',
    name: 'Demo Bank Soal Gambar',
    lecturer_id: lecturerId,
    description: 'Seed data QA untuk pertanyaan gambar dan jawaban gambar.',
    status: 'active',
    metadata: { seed: true, image_question_seed: true },
  }, token);

  const specs = [
    { code: 'IMG-MATH', tag: 'Matematika Gambar', prompt: 'Perhatikan gambar bidang berikut, pilih jawaban yang paling tepat.', answerMode: 'single' },
    { code: 'IMG-BIO', tag: 'Biologi Gambar', prompt: 'Perhatikan ilustrasi organ berikut, pilih dua ciri yang benar.', answerMode: 'multiple' },
    { code: 'IMG-FIS', tag: 'Fisika Gambar', prompt: 'Perhatikan diagram gaya berikut, pilih jawaban yang sesuai.', answerMode: 'single' },
  ];

  for (const spec of specs) {
    const tag = await firstOrCreate('/api/v1/question-tags', spec.code, {
      code: spec.code,
      name: spec.tag,
      lecturer_id: lecturerId,
      description: 'Tag seed QA untuk komposisi publish exam berbasis gambar.',
      status: 'active',
      metadata: { seed: true, image_question_seed: true },
    }, token);

    const questionCode = `${spec.code}-Q-001`;
    const existing = await json('GET', `/api/v1/questions/?page=1&limit=10&search=${encodeURIComponent(questionCode)}`, null, token);
    if ((existing.items || []).some((item) => item.code === questionCode)) {
      console.log(`skip existing ${questionCode}`);
      continue;
    }

    const question = await json('POST', '/api/v1/questions/', {
      code: questionCode,
      name: `${spec.tag} - Soal Gambar`,
      question_bank_id: bank.id,
      lecturer_id: lecturerId,
      question_text: spec.prompt,
      answer_mode: spec.answerMode,
      difficulty: 'medium',
      score: 1,
      explanation: '',
      status: 'active',
      tag_ids: [tag.id],
      metadata: { seed: true, image_question_seed: true },
      options: [
        { label: 'A', text: 'Pilihan gambar A', is_correct: true },
        { label: 'B', text: 'Pilihan gambar B', is_correct: spec.answerMode === 'multiple' },
        { label: 'C', text: 'Pilihan gambar C', is_correct: false },
        { label: 'D', text: 'Pilihan gambar D', is_correct: false },
      ],
    }, token);

    await uploadMedia(question.id, null, 'question', `${questionCode}-question.png`, token);
    for (const option of question.options || []) {
      await uploadMedia(question.id, option.id, 'option', `${questionCode}-option-${option.label}.png`, token);
    }
    console.log(`created ${questionCode}`);
  }
}

main().catch((error) => {
  console.error(error.message);
  process.exit(1);
});
