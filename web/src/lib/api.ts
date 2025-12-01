import axios from 'axios';

const api = axios.create({
  baseURL: '/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor - add auth token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Response interceptor - handle errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

// Auth
export const authAPI = {
  login: (username: string, password: string) =>
    api.post('/auth/login', { username, password }),
  me: () => api.get('/auth/me'),
  logout: () => api.post('/auth/logout'),
};

// Dashboard
export const dashboardAPI = {
  getStats: () => api.get('/dashboard/stats'),
};

// Users
export const usersAPI = {
  list: () => api.get('/users'),
  get: (id: number) => api.get(`/users/${id}`),
  create: (data: { username: string; email: string; password: string; role: string }) =>
    api.post('/users', data),
  update: (id: number, data: Partial<{ email: string; password: string; role: string; active: boolean }>) =>
    api.put(`/users/${id}`, data),
  delete: (id: number) => api.delete(`/users/${id}`),
};

// Packages
export const packagesAPI = {
  list: () => api.get('/packages'),
  create: (data: any) => api.post('/packages', data),
  update: (id: number, data: any) => api.put(`/packages/${id}`, data),
  delete: (id: number) => api.delete(`/packages/${id}`),
};

// Domains
export const domainsAPI = {
  list: () => api.get('/domains'),
  get: (id: number) => api.get(`/domains/${id}`),
  create: (data: { name: string; document_root?: string }) => api.post('/domains', data),
  delete: (id: number) => api.delete(`/domains/${id}`),
};

// Databases
export const databasesAPI = {
  list: () => api.get('/databases'),
  create: (data: { name: string; password?: string }) => api.post('/databases', data),
  delete: (id: number) => api.delete(`/databases/${id}`),
  getSize: (id: number) => api.get(`/databases/${id}/size`),
  getPhpMyAdminURL: (databaseId: number) => api.get('/databases/phpmyadmin', { params: { database_id: databaseId } }),
};

// Database Users
export const databaseUsersAPI = {
  list: () => api.get('/database-users'),
  create: (data: { database_id: number; username: string; password: string }) => api.post('/database-users', data),
  delete: (id: number) => api.delete(`/database-users/${id}`),
};

// System
export const systemAPI = {
  getStats: () => api.get('/system/stats'),
  getServices: () => api.get('/system/services'),
  restartService: (name: string) => api.post(`/system/services/${name}/restart`),
};

// Accounts (Hosting hesapları)
export const accountsAPI = {
  list: () => api.get('/accounts'),
  get: (id: number) => api.get(`/accounts/${id}`),
  create: (data: {
    username: string;
    email: string;
    password: string;
    domain: string;
    package_id: number;
  }) => api.post('/accounts', data),
  delete: (id: number) => api.delete(`/accounts/${id}`),
  suspend: (id: number) => api.post(`/accounts/${id}/suspend`),
  unsuspend: (id: number) => api.post(`/accounts/${id}/unsuspend`),
};

// SSL Certificates
export const sslAPI = {
  list: () => api.get('/ssl'),
  get: (domainId: number) => api.get(`/ssl/${domainId}`),
  issue: (domainId: number) => api.post(`/ssl/${domainId}/issue`),
  renew: (domainId: number) => api.post(`/ssl/${domainId}/renew`),
  revoke: (domainId: number) => api.delete(`/ssl/${domainId}`),
};

// File Manager
export const filesAPI = {
  list: (path: string = '/') => api.get('/files/list', { params: { path } }),
  read: (path: string) => api.get('/files/read', { params: { path } }),
  write: (path: string, content: string) => api.post('/files/write', { path, content }),
  mkdir: (path: string) => api.post('/files/mkdir', { path }),
  delete: (paths: string[]) => api.post('/files/delete', { paths }),
  rename: (oldPath: string, newPath: string) => api.post('/files/rename', { old_path: oldPath, new_path: newPath }),
  copy: (sources: string[], destination: string) => api.post('/files/copy', { sources, destination }),
  move: (sources: string[], destination: string) => api.post('/files/move', { sources, destination }),
  upload: (path: string, files: FileList | File[]) => {
    const formData = new FormData();
    formData.append('path', path);
    Array.from(files).forEach(file => formData.append('files', file));
    return api.post('/files/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
  },
  download: (path: string) => `/api/v1/files/download?path=${encodeURIComponent(path)}`,
  compress: (paths: string[], destination: string) => api.post('/files/compress', { paths, destination }),
  extract: (path: string, destination: string) => api.post('/files/extract', { path, destination }),
  info: (path: string) => api.get('/files/info', { params: { path } }),
};

export default api;
