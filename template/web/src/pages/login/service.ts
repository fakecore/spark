import { http, request } from '@/lib/http';

export interface LoginParams {
  username: string;
  password: string;
}

export interface LoginResult {
  token: string;
  expires: number;
}

export interface UserProfile {
  id: string;
  name: string;
  nickname: string;
}

export function login(params: LoginParams) {
  return request<LoginResult>(http.post('system/auth/login', { json: params }));
}

export function logout() {
  return request(http.post('system/auth/logout'));
}

export function getUserProfile() {
  return request<UserProfile>(http.get('system/user/profile'));
}

export function changePassword(params: { oldPassword: string; newPassword: string; verifyCode?: string }) {
  return request(http.put('system/user/password/change', { json: params }));
}

export function changePasswordById(params: { userId: string; newPassword: string }) {
  return request(http.post('system/user/password/admin-change', { json: params }));
}
