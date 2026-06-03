"use client";

import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  type ReactNode,
} from "react";
import type { AuthUser, RegisterRequest, UpdateProfileRequest, ChangePasswordRequest } from "@/lib/types";
import { authService } from "@/lib/services";
import { setAccessToken } from "@/lib/api";

interface AuthState {
  user: AuthUser | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  loginWithGoogle: (credential: string) => Promise<void>;
  register: (data: RegisterRequest) => Promise<void>;
  logout: () => void;
  updateProfile: (data: UpdateProfileRequest) => Promise<void>;
  changePassword: (data: ChangePasswordRequest) => Promise<void>;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthState | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [loading, setLoading] = useState(true);

const fetchUser = useCallback(async () => {
  try {
    const u = await authService.me();
    setUser(u);
    // Set a role cookie readable by Next.js middleware (not HttpOnly)
    // Security note: this is for routing only — BFF enforces real auth via JWT
    document.cookie = `role=${u.role}; path=/; SameSite=Strict`;
  } catch {
    setUser(null);
    document.cookie = "role=; path=/; max-age=0; SameSite=Strict";
  } finally {
    setLoading(false);
  }
}, []);

  useEffect(() => {
    // What: เรียก fetchUser() ทุกครั้งที่ mount — ไม่ต้องตรวจ localStorage
    // Why:  access_token อยู่ใน memory (หายหลัง reload) แต่ refresh_token อยู่ใน cookie
    //       api.ts จะ tryRefresh() อัตโนมัติเมื่อได้รับ 401 → ได้ access_token ใหม่
    fetchUser();
  }, [fetchUser]);

  const login = async (email: string, password: string) => {
    const res = await authService.login({ email, password });
    // What: เก็บ access_token ใน memory, refresh_token ถูก set เป็น HttpOnly cookie โดย server
    setAccessToken(res.access_token);
    await fetchUser();
  };

  const register = async (data: RegisterRequest) => {
    await authService.register(data);
    await login(data.email, data.password);
  };

  const logout = () => {
    authService.logout().catch(() => {});
    // What: ล้าง access_token จาก memory — cookie ถูกลบโดย server ผ่าน Set-Cookie: MaxAge=-1
    setAccessToken(null);
    setUser(null);
    document.cookie = "role=; path=/; max-age=0; SameSite=Strict"; // ← เพิ่ม
  };

  const updateProfile = async (data: UpdateProfileRequest) => {
    // Why: updateProfile เรียก /users/update/:id — user endpoint เท่านั้น admin ใช้ไม่ได้
    if (!user || user.role !== "user") return;
    await authService.updateProfile(user.id, data);
    await fetchUser();
  };

  const changePassword = async (data: ChangePasswordRequest) => {
    // Why: changePassword เรียก /users/chgpass/:id — user endpoint เท่านั้น admin ใช้ไม่ได้
    if (!user || user.role !== "user") return;
    await authService.changePassword(user.id, data);
  };

  const loginWithGoogle = async (credential: string) => {
    const res = await authService.googleLogin(credential);
    setAccessToken(res.access_token);
    await fetchUser();
  };

  return (
    <AuthContext.Provider value={{ user, loading, login, loginWithGoogle, register, logout, updateProfile, changePassword, refreshUser: fetchUser }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be inside AuthProvider");
  return ctx;
}
