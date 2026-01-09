import { useTheme } from '@/contexts/ThemeContext';
import { Sun, Moon, Monitor } from 'lucide-react';
import { useRef, useState, useEffect } from 'react';
import { cn } from '@/lib/utils';

export function ThemeDropdown() {
  const { theme, setTheme } = useTheme();
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const themes = [
    { value: 'light' as const, label: 'Açık', icon: Sun },
    { value: 'dark' as const, label: 'Koyu', icon: Moon },
    { value: 'system' as const, label: 'Sistem', icon: Monitor },
  ];

  const currentTheme = themes.find(t => t.value === theme) || themes[2];

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className={cn(
          "inline-flex items-center justify-center rounded-md text-sm font-medium",
          "h-9 w-9",
          "transition-colors duration-200",
          "hover:bg-accent hover:text-accent-foreground",
          "focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
        )}
        title="Tema seç"
      >
        <currentTheme.icon className="h-4 w-4" />
      </button>

      {isOpen && (
        <div className="absolute right-0 top-full z-50 mt-2 min-w-[160px] overflow-hidden rounded-lg border bg-background shadow-lg">
          <div className="p-1">
            {themes.map(({ value, label, icon: Icon }) => (
              <button
                key={value}
                onClick={() => {
                  setTheme(value);
                  setIsOpen(false);
                }}
                className={cn(
                  "relative flex w-full cursor-pointer items-center rounded-md px-3 py-2 text-sm",
                  "transition-colors duration-150",
                  "hover:bg-muted/50",
                  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-muted-foreground/20",
                  theme === value && "bg-muted text-foreground font-medium"
                )}
              >
                <Icon className="mr-3 h-4 w-4 text-muted-foreground" />
                <span className="flex-1 text-left">{label}</span>
                {theme === value && (
                  <div className="h-2 w-2 rounded-full bg-foreground" />
                )}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
