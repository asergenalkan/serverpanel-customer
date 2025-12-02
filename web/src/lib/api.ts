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
  get: (id: number) => api.get(`/packages/${id}`),
  create: (data: {
    name: string;
    disk_quota?: number;
    bandwidth_quota?: number;
    max_domains?: number;
    max_databases?: number;
    max_emails?: number;
    max_ftp?: number;
    max_php_memory?: string;
    max_php_upload?: string;
    max_php_execution_time?: number;
  }) => api.post('/packages', data),
  update: (id: number, data: {
    name?: string;
    disk_quota?: number;
    bandwidth_quota?: number;
    max_domains?: number;
    max_databases?: number;
    max_emails?: number;
    max_ftp?: number;
    max_php_memory?: string;
    max_php_upload?: string;
    max_php_execution_time?: number;
  }) => api.put(`/packages/${id}`, data),
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

// PHP Management
export const phpAPI = {
  getVersions: () => api.get('/php/versions'),
  getDomainSettings: (domainId: number) => api.get(`/php/domains/${domainId}`),
  updateVersion: (domainId: number, phpVersion: string) => 
    api.put(`/php/domains/${domainId}/version`, { php_version: phpVersion }),
  updateSettings: (domainId: number, settings: {
    memory_limit: string;
    max_execution_time: number;
    max_input_time: number;
    post_max_size: string;
    upload_max_filesize: string;
    max_file_uploads: number;
    display_errors: boolean;
    error_reporting: string;
  }) => api.put(`/php/domains/${domainId}/settings`, settings),
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
  uploadWithProgress: (path: string, file: File, onProgress: (percent: number) => void) => {
    const formData = new FormData();
    formData.append('path', path);
    formData.append('files', file);
    return api.post('/files/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
      onUploadProgress: (progressEvent) => {
        if (progressEvent.total) {
          const percent = Math.round((progressEvent.loaded * 100) / progressEvent.total);
          onProgress(percent);
        }
      },
    });
  },
  download: (path: string) => `/api/v1/files/download?path=${encodeURIComponent(path)}`,
  compress: (paths: string[], destination: string) => api.post('/files/compress', { paths, destination }),
  extract: (path: string, destination: string) => api.post('/files/extract', { path, destination }),
  info: (path: string) => api.get('/files/info', { params: { path } }),
};

// DNS Management
export const dnsAPI = {
  // DNS Zones
  listZones: () => api.get('/dns/zones'),
  getZone: (domainId: number) => api.get(`/dns/zones/${domainId}`),
  resetZone: (domainId: number) => api.post(`/dns/zones/${domainId}/reset`),
  
  // DNS Records
  createRecord: (data: {
    domain_id: number;
    name: string;
    type: string;
    content: string;
    ttl?: number;
    priority?: number;
  }) => api.post('/dns/records', data),
  updateRecord: (id: number, data: {
    name: string;
    type: string;
    content: string;
    ttl?: number;
    priority?: number;
  }) => api.put(`/dns/records/${id}`, data),
  deleteRecord: (id: number) => api.delete(`/dns/records/${id}`),
};

// FTP Management
export const ftpAPI = {
  // FTP Accounts
  list: () => api.get('/ftp/accounts'),
  create: (data: { username: string; password: string; home_directory?: string; quota_mb?: number }) => 
    api.post('/ftp/accounts', data),
  update: (id: number, data: { password?: string; home_directory?: string; quota_mb?: number; active?: boolean }) => 
    api.put(`/ftp/accounts/${id}`, data),
  delete: (id: number) => api.delete(`/ftp/accounts/${id}`),
  toggle: (id: number) => api.post(`/ftp/accounts/${id}/toggle`),
  
  // FTP Server Settings (admin only)
  getSettings: () => api.get('/ftp/settings'),
  updateSettings: (settings: {
    tls_encryption: string;
    tls_cipher_suite: string;
    allow_anonymous_logins: boolean;
    allow_anonymous_uploads: boolean;
    max_idle_time: number;
    max_connections: number;
    max_connections_per_ip: number;
    allow_root_login: boolean;
    passive_port_min: number;
    passive_port_max: number;
  }) => api.put('/ftp/settings', settings),
  getStatus: () => api.get('/ftp/status'),
  restart: () => api.post('/ftp/restart'),
};

export default api;
