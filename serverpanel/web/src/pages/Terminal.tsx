import React, { useEffect, useRef, useState } from 'react';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import '@xterm/xterm/css/xterm.css';
import Layout from '../components/Layout';
import { Terminal as TerminalIcon, Maximize2, Minimize2, X } from 'lucide-react';

const TerminalPage: React.FC = () => {
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<XTerm | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const [connected, setConnected] = useState(false);
  const [fullscreen, setFullscreen] = useState(false);

  useEffect(() => {
    if (!terminalRef.current) return;

    // Create terminal
    const term = new XTerm({
      cursorBlink: true,
      cursorStyle: 'block',
      fontSize: 14,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      theme: {
        background: '#1a1b26',
        foreground: '#a9b1d6',
        cursor: '#c0caf5',
        cursorAccent: '#1a1b26',
        selectionBackground: '#33467c',
        black: '#32344a',
        red: '#f7768e',
        green: '#9ece6a',
        yellow: '#e0af68',
        blue: '#7aa2f7',
        magenta: '#ad8ee6',
        cyan: '#449dab',
        white: '#787c99',
        brightBlack: '#444b6a',
        brightRed: '#ff7a93',
        brightGreen: '#b9f27c',
        brightYellow: '#ff9e64',
        brightBlue: '#7da6ff',
        brightMagenta: '#bb9af7',
        brightCyan: '#0db9d7',
        brightWhite: '#acb0d0',
      },
      allowProposedApi: true,
    });

    // Addons
    const fitAddon = new FitAddon();
    const webLinksAddon = new WebLinksAddon();

    term.loadAddon(fitAddon);
    term.loadAddon(webLinksAddon);

    // Open terminal
    term.open(terminalRef.current);
    fitAddon.fit();

    xtermRef.current = term;
    fitAddonRef.current = fitAddon;

    // Connect WebSocket
    const token = localStorage.getItem('token');
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/ws/terminal?token=${token}`;

    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.binaryType = 'arraybuffer';

    ws.onopen = () => {
      setConnected(true);
      // Send initial size
      const dims = fitAddon.proposeDimensions();
      if (dims) {
        ws.send(new Uint8Array([0x01, ...new TextEncoder().encode(`${dims.rows},${dims.cols}`)]));
      }
    };

    ws.onmessage = (event) => {
      if (event.data instanceof ArrayBuffer) {
        term.write(new Uint8Array(event.data));
      } else {
        term.write(event.data);
      }
    };

    ws.onclose = () => {
      setConnected(false);
      term.write('\r\n\x1b[31mBağlantı kapandı.\x1b[0m\r\n');
    };

    ws.onerror = () => {
      setConnected(false);
      term.write('\r\n\x1b[31mBağlantı hatası.\x1b[0m\r\n');
    };

    // Send input to WebSocket
    term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(new TextEncoder().encode(data));
      }
    });

    // Handle resize
    const handleResize = () => {
      fitAddon.fit();
      const dims = fitAddon.proposeDimensions();
      if (dims && ws.readyState === WebSocket.OPEN) {
        ws.send(new Uint8Array([0x01, ...new TextEncoder().encode(`${dims.rows},${dims.cols}`)]));
      }
    };

    window.addEventListener('resize', handleResize);

    // Cleanup
    return () => {
      window.removeEventListener('resize', handleResize);
      ws.close();
      term.dispose();
    };
  }, []);

  // Handle fullscreen toggle
  useEffect(() => {
    if (fitAddonRef.current) {
      setTimeout(() => {
        fitAddonRef.current?.fit();
      }, 100);
    }
  }, [fullscreen]);

  const reconnect = () => {
    window.location.reload();
  };

  return (
    <Layout>
      <div className={`${fullscreen ? 'fixed inset-0 z-50 bg-background p-4' : ''}`}>
        {/* Header */}
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center space-x-3">
            <div className="p-2 bg-green-500/10 rounded-lg">
              <TerminalIcon className="w-6 h-6 text-green-500" />
            </div>
            <div>
              <h1 className="text-2xl font-bold">Terminal</h1>
              <p className="text-sm text-muted-foreground">
                Sunucu komut satırı erişimi
              </p>
            </div>
          </div>

          <div className="flex items-center space-x-2">
            {/* Connection status */}
            <div className={`flex items-center space-x-2 px-3 py-1 rounded-full text-sm ${
              connected ? 'bg-green-500/10 text-green-500' : 'bg-red-500/10 text-red-500'
            }`}>
              <div className={`w-2 h-2 rounded-full ${connected ? 'bg-green-500' : 'bg-red-500'}`} />
              <span>{connected ? 'Bağlı' : 'Bağlantı Yok'}</span>
            </div>

            {!connected && (
              <button
                onClick={reconnect}
                className="px-3 py-1 bg-orange-500 text-white rounded-lg hover:bg-orange-600 text-sm"
              >
                Yeniden Bağlan
              </button>
            )}

            {/* Fullscreen toggle */}
            <button
              onClick={() => setFullscreen(!fullscreen)}
              className="p-2 hover:bg-muted rounded-lg"
              title={fullscreen ? 'Küçült' : 'Tam Ekran'}
            >
              {fullscreen ? (
                <Minimize2 className="w-5 h-5" />
              ) : (
                <Maximize2 className="w-5 h-5" />
              )}
            </button>

            {fullscreen && (
              <button
                onClick={() => setFullscreen(false)}
                className="p-2 hover:bg-muted rounded-lg"
                title="Kapat"
              >
                <X className="w-5 h-5" />
              </button>
            )}
          </div>
        </div>

        {/* Terminal Container */}
        <div className={`bg-[#1a1b26] rounded-lg border border-border overflow-hidden ${
          fullscreen ? 'h-[calc(100vh-120px)]' : 'h-[600px]'
        }`}>
          <div 
            ref={terminalRef} 
            className="w-full h-full p-2"
            style={{ backgroundColor: '#1a1b26' }}
          />
        </div>

        {/* Info */}
        {!fullscreen && (
          <div className="mt-4 p-4 bg-muted/50 rounded-lg">
            <h3 className="font-medium mb-2">💡 Kısayollar</h3>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm text-muted-foreground">
              <div><kbd className="px-2 py-1 bg-muted rounded">Ctrl+C</kbd> İptal</div>
              <div><kbd className="px-2 py-1 bg-muted rounded">Ctrl+L</kbd> Temizle</div>
              <div><kbd className="px-2 py-1 bg-muted rounded">Ctrl+A</kbd> Satır başı</div>
              <div><kbd className="px-2 py-1 bg-muted rounded">Ctrl+E</kbd> Satır sonu</div>
            </div>
          </div>
        )}
      </div>
    </Layout>
  );
};

export default TerminalPage;
