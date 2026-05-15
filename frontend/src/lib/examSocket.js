export function connectExamSocket({ sessionId, token, onMessage, onStatus }) {
  let socket;
  let closed = false;
  let attempt = 0;
  let heartbeat;

  const base = import.meta.env.VITE_WS_URL || `${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/api/v1`;
  const url = `${base}/exam-sessions/ws?session_id=${encodeURIComponent(sessionId)}&token=${encodeURIComponent(token || '')}`;

  function connect() {
    if (closed) return;
    onStatus?.('connecting');
    socket = new WebSocket(url);
    socket.onopen = () => {
      attempt = 0;
      onStatus?.('connected');
      heartbeat = setInterval(() => send({ type: 'heartbeat.ping', payload: { client_time: new Date().toISOString() } }), 25000);
    };
    socket.onmessage = (event) => {
      try {
        onMessage?.(JSON.parse(event.data));
      } catch {
        onMessage?.({ type: 'raw', payload: event.data });
      }
    };
    socket.onclose = () => {
      clearInterval(heartbeat);
      onStatus?.('disconnected');
      if (!closed) {
        const timeout = Math.min(30000, 1000 * 2 ** attempt);
        attempt += 1;
        setTimeout(connect, timeout);
      }
    };
    socket.onerror = () => socket.close();
  }

  function send(message) {
    if (socket?.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ ...message, sent_at: new Date().toISOString() }));
      return true;
    }
    return false;
  }

  connect();
  return {
    send,
    close() {
      closed = true;
      clearInterval(heartbeat);
      socket?.close();
    },
  };
}
