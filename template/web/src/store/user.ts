import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export interface UserInfo {
  id: string;
  name: string;
  nickname: string;
}

interface UserState {
  userInfo: UserInfo | null;
  setUserInfo: (info: UserInfo) => void;
  clearUser: () => void;
}

export const useUserStore = create<UserState>()(
  persist(
    (set) => ({
      userInfo: null,
      setUserInfo: (info) => set({ userInfo: info }),
      clearUser: () => set({ userInfo: null }),
    }),
    { name: 'user-storage' },
  ),
);
