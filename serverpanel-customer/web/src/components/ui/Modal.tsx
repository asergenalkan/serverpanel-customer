import { useEffect, useCallback, useState, ReactNode } from 'react';
import { X, AlertTriangle } from 'lucide-react';
import { Button } from './Button';

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  children: ReactNode;
  size?: 'sm' | 'md' | 'lg' | 'xl' | 'full';
  hasUnsavedChanges?: boolean;
  onConfirmClose?: () => void;
}

// Unsaved changes confirmation modal
function UnsavedChangesModal({ 
  isOpen, 
  onDiscard, 
  onCancel 
}: { 
  isOpen: boolean; 
  onDiscard: () => void; 
  onCancel: () => void;
}) {
  useEffect(() => {
    if (!isOpen) return;
    
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onCancel();
      }
    };
    
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onCancel]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[60]">
      <div className="bg-card rounded-lg p-6 w-96 shadow-xl border animate-in fade-in zoom-in-95 duration-200">
        <div className="flex items-start gap-3">
          <div className="p-2 rounded-full bg-orange-100 dark:bg-orange-500/20">
            <AlertTriangle className="w-5 h-5 text-orange-600 dark:text-orange-400" />
          </div>
          <div className="flex-1">
            <h3 className="text-lg font-semibold">Kaydedilmemiş Değişiklikler</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Kaydedilmemiş değişiklikleriniz var. Çıkmak istediğinize emin misiniz?
            </p>
          </div>
        </div>
        <div className="flex justify-end gap-2 mt-6">
          <Button variant="ghost" onClick={onCancel}>
            Vazgeç
          </Button>
          <Button variant="destructive" onClick={onDiscard}>
            Değişiklikleri Sil
          </Button>
        </div>
      </div>
    </div>
  );
}

export function Modal({ 
  isOpen, 
  onClose, 
  title, 
  children, 
  size = 'md',
  hasUnsavedChanges = false,
}: ModalProps) {
  const [showUnsavedWarning, setShowUnsavedWarning] = useState(false);

  const handleClose = useCallback(() => {
    if (hasUnsavedChanges) {
      setShowUnsavedWarning(true);
    } else {
      onClose();
    }
  }, [hasUnsavedChanges, onClose]);

  const handleConfirmDiscard = useCallback(() => {
    setShowUnsavedWarning(false);
    onClose();
  }, [onClose]);

  const handleCancelDiscard = useCallback(() => {
    setShowUnsavedWarning(false);
  }, []);

  // ESC key handler
  useEffect(() => {
    if (!isOpen) return;
    
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !showUnsavedWarning) {
        handleClose();
      }
    };
    
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, handleClose, showUnsavedWarning]);

  // Prevent body scroll when modal is open
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }
    return () => {
      document.body.style.overflow = '';
    };
  }, [isOpen]);

  if (!isOpen) return null;

  const sizeClasses = {
    sm: 'max-w-sm',
    md: 'max-w-md',
    lg: 'max-w-2xl',
    xl: 'max-w-5xl',
    full: 'max-w-[95vw] h-[90vh]',
  };

  return (
    <>
      <div 
        className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
        onClick={handleClose}
      >
        <div 
          className={`bg-card rounded-lg shadow-xl border w-full ${sizeClasses[size]} animate-in fade-in zoom-in-95 duration-200 flex flex-col`}
          onClick={(e) => e.stopPropagation()}
        >
          {title && (
            <div className="flex items-center justify-between p-4 border-b">
              <h3 className="text-lg font-semibold">{title}</h3>
              <Button 
                variant="ghost" 
                size="icon" 
                onClick={handleClose}
                className="w-8 h-8"
              >
                <X className="w-4 h-4" />
              </Button>
            </div>
          )}
          <div className={`${title ? '' : 'pt-4'} ${size === 'full' ? 'flex-1 overflow-auto' : ''}`}>
            {children}
          </div>
        </div>
      </div>
      
      <UnsavedChangesModal
        isOpen={showUnsavedWarning}
        onDiscard={handleConfirmDiscard}
        onCancel={handleCancelDiscard}
      />
    </>
  );
}

// Simple modal for quick confirmations
interface SimpleModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
}

export function SimpleModal({ isOpen, onClose, title, children }: SimpleModalProps) {
  useEffect(() => {
    if (!isOpen) return;
    
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };
    
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <div 
      className="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
      onClick={onClose}
    >
      <div 
        className="bg-card rounded-lg p-6 w-96 shadow-xl border animate-in fade-in zoom-in-95 duration-200"
        onClick={(e) => e.stopPropagation()}
      >
        <h3 className="text-lg font-semibold mb-4">{title}</h3>
        {children}
      </div>
    </div>
  );
}
