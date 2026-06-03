"use client";

import { createContext, useCallback, useContext, useRef, useState, type ReactNode } from "react";

type ToastType = "success" | "error" | "info";

interface Toast {
  id: number;
  message: string;
  type: ToastType;
}

interface ToastState {
  toast: (message: string, type?: ToastType) => void;
}

const ToastContext = createContext<ToastState | undefined>(undefined);

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const counter = useRef(0);

  const toast = useCallback((message: string, type: ToastType = "info") => {
    const id = ++counter.current;
    setToasts((prev) => [...prev, { id, message, type }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 3500);
  }, []);

  const iconMap: Record<ToastType, string> = {
    success: "check_circle",
    error: "error",
    info: "info",
  };
  const colorMap: Record<ToastType, string> = {
    success: "bg-[#1e4620] text-white",
    error: "bg-[#410002] text-white",
    info: "bg-surface-container-highest text-on-surface",
  };

  return (
    <ToastContext.Provider value={{ toast }}>
      {children}
      {/* Toast container */}
      <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-[9999] flex flex-col gap-2 items-center pointer-events-none">
        {toasts.map((t) => (
          <div
            key={t.id}
            className={`flex items-center gap-2.5 px-4 py-3 rounded-xl shadow-lg text-sm font-medium animate-fade-in-up ${colorMap[t.type]}`}
          >
            <span
              className="material-symbols-outlined text-[18px]"
              style={{ fontVariationSettings: "'FILL' 1" }}
            >
              {iconMap[t.type]}
            </span>
            {t.message}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast() {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error("useToast must be inside ToastProvider");
  return ctx.toast;
}
