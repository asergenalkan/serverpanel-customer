import { useState, useEffect, useRef } from 'react';
import { X, CheckCircle, XCircle, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/Button';

interface TaskModalProps {
  isOpen: boolean;
  onClose: () => void;
  taskId: string | null;
  taskName: string;
  onComplete?: (success: boolean) => void;
}

export default function TaskModal({ isOpen, onClose, taskId, taskName, onComplete }: TaskModalProps) {
  const [logs, setLogs] = useState<string[]>([]);
  const [status, setStatus] = useState<'running' | 'completed' | 'failed'>('running');
  const logsEndRef = useRef<HTMLDivElement>(null);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (!isOpen || !taskId) return;

    // Reset state
    setLogs([]);
    setStatus('running');

    // Connect to WebSocket with token
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const token = localStorage.getItem('token');
    const wsUrl = `${protocol}//${window.location.host}/api/v1/ws/tasks/${taskId}?token=${token}`;
    
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      console.log('WebSocket connected');
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        
        if (data.type === 'log') {
          // Check for status message
          if (data.data.startsWith('__STATUS__')) {
            const newStatus = data.data.replace('__STATUS__', '') as 'completed' | 'failed';
            setStatus(newStatus);
            if (onComplete) {
              onComplete(newStatus === 'completed');
            }
          } else {
            setLogs(prev => [...prev, data.data]);
          }
        } else if (data.type === 'status') {
          setStatus(data.status);
          if (onComplete) {
            onComplete(data.status === 'completed');
          }
        }
      } catch (e) {
        console.error('Failed to parse WebSocket message:', e);
      }
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    ws.onclose = () => {
      console.log('WebSocket closed');
    };

    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [isOpen, taskId, onComplete]);

  // Auto-scroll to bottom
  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [logs]);

  if (!isOpen) return null;

  const getStatusIcon = () => {
    switch (status) {
      case 'running':
        return <Loader2 className="w-6 h-6 animate-spin text-blue-500" />;
      case 'completed':
        return <CheckCircle className="w-6 h-6 text-green-500" />;
      case 'failed':
        return <XCircle className="w-6 h-6 text-red-500" />;
    }
  };

  const getStatusText = () => {
    switch (status) {
      case 'running':
        return 'İşlem devam ediyor...';
      case 'completed':
        return 'İşlem tamamlandı!';
      case 'failed':
        return 'İşlem başarısız!';
    }
  };

  const getStatusColor = () => {
    switch (status) {
      case 'running':
        return 'text-blue-500';
      case 'completed':
        return 'text-green-500';
      case 'failed':
        return 'text-red-500';
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black/50 backdrop-blur-sm"
        onClick={status !== 'running' ? onClose : undefined}
      />
      
      {/* Modal */}
      <div className="relative bg-background border rounded-lg shadow-xl w-full max-w-2xl mx-4 max-h-[80vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b">
          <div className="flex items-center gap-3">
            {getStatusIcon()}
            <div>
              <h2 className="font-semibold">{taskName}</h2>
              <p className={`text-sm ${getStatusColor()}`}>{getStatusText()}</p>
            </div>
          </div>
          {status !== 'running' && (
            <Button variant="ghost" size="sm" onClick={onClose}>
              <X className="w-5 h-5" />
            </Button>
          )}
        </div>

        {/* Logs */}
        <div className="flex-1 overflow-auto p-4 bg-black/90 font-mono text-sm">
          <div className="space-y-0.5">
            {logs.length === 0 ? (
              <p className="text-gray-500">Bağlanıyor...</p>
            ) : (
              logs.map((log, index) => (
                <div 
                  key={index} 
                  className={`${
                    log.startsWith('✅') ? 'text-green-400' :
                    log.startsWith('❌') ? 'text-red-400' :
                    log.startsWith('🚀') || log.startsWith('📦') || log.startsWith('🔧') || log.startsWith('🔄') || log.startsWith('🗑️') || log.startsWith('🛑') ? 'text-blue-400' :
                    log.includes('error') || log.includes('Error') || log.includes('ERROR') ? 'text-red-400' :
                    log.includes('warning') || log.includes('Warning') || log.includes('WARNING') ? 'text-yellow-400' :
                    'text-gray-300'
                  }`}
                >
                  {log || '\u00A0'}
                </div>
              ))
            )}
            <div ref={logsEndRef} />
          </div>
        </div>

        {/* Footer */}
        <div className="p-4 border-t flex justify-end">
          {status === 'running' ? (
            <p className="text-sm text-muted-foreground">
              İşlem tamamlanana kadar bu pencereyi kapatmayın...
            </p>
          ) : (
            <Button onClick={onClose}>
              Kapat
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
