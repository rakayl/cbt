import { useState } from 'react';
import { Download } from 'lucide-react';
import { api } from '../lib/api';

export default function ReportsPage() {
  const [reportType, setReportType] = useState('tenant_analytics');
  const [busy, setBusy] = useState(false);

  async function exportReport(format) {
    setBusy(true);
    try {
      const response = await api.post('/reports/export', { report_type: reportType, format }, { responseType: 'blob' });
      const extension = format === 'pdf' ? 'pdf' : 'csv';
      const url = URL.createObjectURL(response.data);
      const link = document.createElement('a');
      link.href = url;
      link.download = `${reportType}.${extension}`;
      link.click();
      URL.revokeObjectURL(url);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Report Generator</h1>
      <div className="panel p-4 flex flex-wrap gap-3">
        <select className="input max-w-xs" value={reportType} onChange={(event) => setReportType(event.target.value)}>
          <option value="tenant_analytics">Tenant analytics</option>
          <option value="lecturer_report">Lecturer report</option>
          <option value="transcript">Transcript</option>
        </select>
        <button className="btn btn-primary" disabled={busy} onClick={() => exportReport('pdf')}><Download size={18} /> Export PDF</button>
        <button className="btn btn-ghost" disabled={busy} onClick={() => exportReport('excel')}><Download size={18} /> Export Excel</button>
      </div>
    </div>
  );
}
