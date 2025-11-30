import { useEffect, useState, useRef, useCallback } from 'react';
import { filesAPI } from '@/lib/api';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Card, CardContent } from '@/components/ui/Card';
import { Modal, SimpleModal } from '@/components/ui/Modal';
import Layout from '@/components/Layout';
import {
  FolderOpen,
  File,
  FileText,
  FileCode,
  FileImage,
  FileArchive,
  Upload,
  Download,
  Trash2,
  Plus,
  Edit3,
  Copy,
  Scissors,
  ClipboardPaste,
  Archive,
  FolderPlus,
  RefreshCw,
  ChevronRight,
  Home,
  X,
  Check,
  Search,
  Grid,
  List,
  ArrowUp,
} from 'lucide-react';

interface FileItem {
  name: string;
  path: string;
  is_dir: boolean;
  size: number;
  modified: string;
  permissions: string;
  extension?: string;
}

// File type icons
const getFileIcon = (file: FileItem) => {
  if (file.is_dir) return <FolderOpen className="w-5 h-5 text-yellow-500" />;
  
  const ext = file.extension?.toLowerCase() || '';
  
  if (['jpg', 'jpeg', 'png', 'gif', 'svg', 'webp', 'ico'].includes(ext)) {
    return <FileImage className="w-5 h-5 text-green-500" />;
  }
  if (['zip', 'tar', 'gz', 'rar', '7z'].includes(ext)) {
    return <FileArchive className="w-5 h-5 text-orange-500" />;
  }
  if (['js', 'ts', 'jsx', 'tsx', 'php', 'py', 'go', 'java', 'c', 'cpp', 'h', 'css', 'scss', 'html', 'xml', 'json', 'yaml', 'yml', 'sh', 'bash'].includes(ext)) {
    return <FileCode className="w-5 h-5 text-blue-500" />;
  }
  if (['txt', 'md', 'log', 'ini', 'conf', 'cfg'].includes(ext)) {
    return <FileText className="w-5 h-5 text-gray-500" />;
  }
  
  return <File className="w-5 h-5 text-gray-400" />;
};

// Format file size
const formatSize = (bytes: number): string => {
  if (bytes === 0) return '-';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
};

// Format date
const formatDate = (dateStr: string): string => {
  return new Date(dateStr).toLocaleDateString('tr-TR', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
};

export default function FileManager() {
  const [currentPath, setCurrentPath] = useState('/');
  const [files, setFiles] = useState<FileItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedFiles, setSelectedFiles] = useState<Set<string>>(new Set());
  const [viewMode, setViewMode] = useState<'list' | 'grid'>('list');
  const [searchQuery, setSearchQuery] = useState('');
  const [clipboard, setClipboard] = useState<{ paths: string[]; action: 'copy' | 'cut' } | null>(null);
  
  // Modals
  const [showNewFolder, setShowNewFolder] = useState(false);
  const [showNewFile, setShowNewFile] = useState(false);
  const [showRename, setShowRename] = useState<FileItem | null>(null);
  const [showEditor, setShowEditor] = useState<{ path: string; content: string; name: string; originalContent: string } | null>(null);
  const [showPreview, setShowPreview] = useState<{ url: string; name: string } | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  
  // Form states
  const [newFolderName, setNewFolderName] = useState('');
  const [newFileName, setNewFileName] = useState('');
  const [renameName, setRenameName] = useState('');
  const [editorContent, setEditorContent] = useState('');
  
  // Upload
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [dragOver, setDragOver] = useState(false);

  // Fetch files
  const fetchFiles = useCallback(async () => {
    setLoading(true);
    try {
      const response = await filesAPI.list(currentPath);
      if (response.data.success) {
        setFiles(response.data.data.files || []);
      }
    } catch (error) {
      console.error('Failed to fetch files:', error);
    } finally {
      setLoading(false);
    }
  }, [currentPath]);

  useEffect(() => {
    fetchFiles();
    setSelectedFiles(new Set());
  }, [fetchFiles]);

  // Navigate to path
  const navigateTo = (path: string) => {
    setCurrentPath(path);
    setSelectedFiles(new Set());
  };

  // Check if file is editable (text-based)
  const isEditableFile = (ext: string): boolean => {
    const editableExtensions = [
      'txt', 'md', 'log', 'ini', 'conf', 'cfg', 'json', 'xml', 'yaml', 'yml',
      'html', 'htm', 'css', 'scss', 'sass', 'less',
      'js', 'jsx', 'ts', 'tsx', 'mjs', 'cjs',
      'php', 'py', 'rb', 'go', 'java', 'c', 'cpp', 'h', 'hpp', 'cs',
      'sh', 'bash', 'zsh', 'fish', 'ps1', 'bat', 'cmd',
      'sql', 'htaccess', 'env', 'gitignore', 'dockerignore',
      'dockerfile', 'makefile', 'cmake', 'toml', 'lock'
    ];
    return editableExtensions.includes(ext.toLowerCase());
  };

  // Check if file is an image
  const isImageFile = (ext: string): boolean => {
    const imageExtensions = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp', 'ico'];
    return imageExtensions.includes(ext.toLowerCase());
  };

  // Load image for preview
  const loadImagePreview = async (file: FileItem) => {
    setPreviewLoading(true);
    try {
      const token = localStorage.getItem('token');
      const response = await fetch(`/api/v1/files/download?path=${encodeURIComponent(file.path)}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        const blob = await response.blob();
        const url = URL.createObjectURL(blob);
        setShowPreview({ url, name: file.name });
      }
    } catch (error) {
      console.error('Failed to load image:', error);
    } finally {
      setPreviewLoading(false);
    }
  };

  // Handle file click
  const handleFileClick = (file: FileItem) => {
    if (file.is_dir) {
      navigateTo(file.path);
    } else if (isImageFile(file.extension || '')) {
      // Show image preview
      loadImagePreview(file);
    } else if (isEditableFile(file.extension || '')) {
      // Open in editor
      openFile(file);
    } else {
      // Download other files - use fetch with auth
      const token = localStorage.getItem('token');
      fetch(`/api/v1/files/download?path=${encodeURIComponent(file.path)}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      })
        .then(res => res.blob())
        .then(blob => {
          const url = URL.createObjectURL(blob);
          const a = document.createElement('a');
          a.href = url;
          a.download = file.name;
          document.body.appendChild(a);
          a.click();
          document.body.removeChild(a);
          URL.revokeObjectURL(url);
        });
    }
  };

  // Toggle file selection
  const toggleSelection = (file: FileItem, e: React.MouseEvent) => {
    e.stopPropagation();
    const newSelection = new Set(selectedFiles);
    if (newSelection.has(file.path)) {
      newSelection.delete(file.path);
    } else {
      newSelection.add(file.path);
    }
    setSelectedFiles(newSelection);
  };

  // Select all
  const selectAll = () => {
    if (selectedFiles.size === files.length) {
      setSelectedFiles(new Set());
    } else {
      setSelectedFiles(new Set(files.map(f => f.path)));
    }
  };

  // Open file in editor
  const openFile = async (file: FileItem) => {
    try {
      const response = await filesAPI.read(file.path);
      if (response.data.success) {
        const content = response.data.data.content;
        setShowEditor({
          path: file.path,
          content: content,
          name: file.name,
          originalContent: content,
        });
        setEditorContent(content);
      }
    } catch (error: any) {
      alert(error.response?.data?.error || 'Dosya açılamadı');
    }
  };

  // Save file
  const saveFile = async () => {
    if (!showEditor) return;
    try {
      await filesAPI.write(showEditor.path, editorContent);
      setShowEditor(null);
      fetchFiles();
    } catch (error: any) {
      alert(error.response?.data?.error || 'Dosya kaydedilemedi');
    }
  };

  // Create folder
  const createFolder = async () => {
    if (!newFolderName.trim()) return;
    try {
      const path = currentPath === '/' ? `/${newFolderName}` : `${currentPath}/${newFolderName}`;
      await filesAPI.mkdir(path);
      setShowNewFolder(false);
      setNewFolderName('');
      fetchFiles();
    } catch (error: any) {
      alert(error.response?.data?.error || 'Klasör oluşturulamadı');
    }
  };

  // Create file
  const createFile = async () => {
    if (!newFileName.trim()) return;
    try {
      const path = currentPath === '/' ? `/${newFileName}` : `${currentPath}/${newFileName}`;
      await filesAPI.write(path, '');
      setShowNewFile(false);
      setNewFileName('');
      fetchFiles();
    } catch (error: any) {
      alert(error.response?.data?.error || 'Dosya oluşturulamadı');
    }
  };

  // Rename
  const handleRename = async () => {
    if (!showRename || !renameName.trim()) return;
    try {
      const dir = showRename.path.substring(0, showRename.path.lastIndexOf('/')) || '/';
      const newPath = dir === '/' ? `/${renameName}` : `${dir}/${renameName}`;
      await filesAPI.rename(showRename.path, newPath);
      setShowRename(null);
      setRenameName('');
      fetchFiles();
    } catch (error: any) {
      alert(error.response?.data?.error || 'Yeniden adlandırılamadı');
    }
  };

  // Delete
  const handleDelete = async () => {
    if (selectedFiles.size === 0) return;
    if (!confirm(`${selectedFiles.size} öğe silinecek. Emin misiniz?`)) return;
    
    try {
      await filesAPI.delete(Array.from(selectedFiles));
      setSelectedFiles(new Set());
      fetchFiles();
    } catch (error: any) {
      alert(error.response?.data?.error || 'Silinemedi');
    }
  };

  // Copy/Cut
  const handleCopy = () => {
    setClipboard({ paths: Array.from(selectedFiles), action: 'copy' });
  };

  const handleCut = () => {
    setClipboard({ paths: Array.from(selectedFiles), action: 'cut' });
  };

  // Paste
  const handlePaste = async () => {
    if (!clipboard) return;
    try {
      if (clipboard.action === 'copy') {
        await filesAPI.copy(clipboard.paths, currentPath);
      } else {
        await filesAPI.move(clipboard.paths, currentPath);
        setClipboard(null);
      }
      fetchFiles();
    } catch (error: any) {
      alert(error.response?.data?.error || 'Yapıştırılamadı');
    }
  };

  // Upload
  const handleUpload = async (fileList: FileList | File[]) => {
    if (!fileList.length) return;
    try {
      await filesAPI.upload(currentPath, fileList);
      fetchFiles();
    } catch (error: any) {
      alert(error.response?.data?.error || 'Yükleme başarısız');
    }
  };

  // Download
  const handleDownload = () => {
    selectedFiles.forEach(path => {
      const url = filesAPI.download(path);
      const a = document.createElement('a');
      a.href = url;
      a.download = path.split('/').pop() || 'download';
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
    });
  };

  // Compress
  const handleCompress = async () => {
    if (selectedFiles.size === 0) return;
    const name = prompt('Arşiv adı:', 'archive.zip');
    if (!name) return;
    
    try {
      const dest = currentPath === '/' ? `/${name}` : `${currentPath}/${name}`;
      await filesAPI.compress(Array.from(selectedFiles), dest);
      fetchFiles();
    } catch (error: any) {
      alert(error.response?.data?.error || 'Sıkıştırılamadı');
    }
  };

  // Extract
  const handleExtract = async (file: FileItem) => {
    try {
      const dest = currentPath;
      await filesAPI.extract(file.path, dest);
      fetchFiles();
    } catch (error: any) {
      alert(error.response?.data?.error || 'Çıkarılamadı');
    }
  };

  // Drag & Drop
  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(true);
  };

  const handleDragLeave = () => {
    setDragOver(false);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    if (e.dataTransfer.files.length) {
      handleUpload(e.dataTransfer.files);
    }
  };

  // Breadcrumb
  const breadcrumbs = currentPath.split('/').filter(Boolean);

  // Filtered files
  const filteredFiles = files.filter(f => 
    f.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <Layout>
      <div 
        className={`space-y-4 ${dragOver ? 'ring-2 ring-blue-500 ring-offset-4 rounded-lg' : ''}`}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
      >
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Dosya Yöneticisi</h1>
            <p className="text-muted-foreground text-sm">
              Dosyalarınızı yönetin, düzenleyin ve yükleyin
            </p>
          </div>
        </div>

        {/* Toolbar */}
        <Card>
          <CardContent className="p-3">
            <div className="flex flex-wrap items-center gap-2">
              {/* Navigation */}
              <Button 
                variant="ghost" 
                size="sm"
                onClick={() => navigateTo('/')}
                title="Ana Dizin"
              >
                <Home className="w-4 h-4" />
              </Button>
              <Button 
                variant="ghost" 
                size="sm"
                onClick={() => {
                  const parent = currentPath.substring(0, currentPath.lastIndexOf('/')) || '/';
                  navigateTo(parent);
                }}
                disabled={currentPath === '/'}
                title="Üst Dizin"
              >
                <ArrowUp className="w-4 h-4" />
              </Button>
              <Button variant="ghost" size="sm" onClick={fetchFiles} title="Yenile">
                <RefreshCw className="w-4 h-4" />
              </Button>

              <div className="w-px h-6 bg-slate-200 mx-1" />

              {/* Create */}
              <Button 
                variant="ghost" 
                size="sm"
                onClick={() => setShowNewFolder(true)}
                title="Yeni Klasör"
              >
                <FolderPlus className="w-4 h-4" />
              </Button>
              <Button 
                variant="ghost" 
                size="sm"
                onClick={() => setShowNewFile(true)}
                title="Yeni Dosya"
              >
                <Plus className="w-4 h-4" />
              </Button>
              <Button 
                variant="outline" 
                size="sm"
                onClick={() => fileInputRef.current?.click()}
                className="gap-1"
              >
                <Upload className="w-4 h-4" />
                <span className="hidden sm:inline">Yükle</span>
              </Button>
              <input
                ref={fileInputRef}
                type="file"
                multiple
                className="hidden"
                onChange={(e) => e.target.files && handleUpload(e.target.files)}
              />

              <div className="w-px h-6 bg-slate-200 mx-1" />

              {/* Actions */}
              <Button 
                variant="ghost" 
                size="sm"
                onClick={handleCopy}
                disabled={selectedFiles.size === 0}
                title="Kopyala"
              >
                <Copy className="w-4 h-4" />
              </Button>
              <Button 
                variant="ghost" 
                size="sm"
                onClick={handleCut}
                disabled={selectedFiles.size === 0}
                title="Kes"
              >
                <Scissors className="w-4 h-4" />
              </Button>
              <Button 
                variant="ghost" 
                size="sm"
                onClick={handlePaste}
                disabled={!clipboard}
                title="Yapıştır"
              >
                <ClipboardPaste className="w-4 h-4" />
              </Button>
              <Button 
                variant="ghost" 
                size="sm"
                onClick={handleDownload}
                disabled={selectedFiles.size === 0}
                title="İndir"
              >
                <Download className="w-4 h-4" />
              </Button>
              <Button 
                variant="ghost" 
                size="sm"
                onClick={handleCompress}
                disabled={selectedFiles.size === 0}
                title="Sıkıştır"
              >
                <Archive className="w-4 h-4" />
              </Button>
              <Button 
                variant="ghost" 
                size="sm"
                onClick={handleDelete}
                disabled={selectedFiles.size === 0}
                className="text-red-500 hover:text-red-700"
                title="Sil"
              >
                <Trash2 className="w-4 h-4" />
              </Button>

              <div className="flex-1" />

              {/* Search & View */}
              <div className="relative">
                <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                <Input
                  type="text"
                  placeholder="Ara..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-8 h-8 w-40"
                />
              </div>
              <div className="flex border rounded-md">
                <Button
                  variant={viewMode === 'list' ? 'secondary' : 'ghost'}
                  size="sm"
                  className="rounded-r-none"
                  onClick={() => setViewMode('list')}
                >
                  <List className="w-4 h-4" />
                </Button>
                <Button
                  variant={viewMode === 'grid' ? 'secondary' : 'ghost'}
                  size="sm"
                  className="rounded-l-none"
                  onClick={() => setViewMode('grid')}
                >
                  <Grid className="w-4 h-4" />
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Breadcrumb */}
        <div className="flex items-center gap-1 text-sm text-muted-foreground px-1">
          <button 
            onClick={() => navigateTo('/')}
            className="hover:text-blue-600 flex items-center gap-1"
          >
            <Home className="w-4 h-4" />
            <span>home</span>
          </button>
          {breadcrumbs.map((part, i) => (
            <span key={i} className="flex items-center gap-1">
              <ChevronRight className="w-4 h-4" />
              <button
                onClick={() => navigateTo('/' + breadcrumbs.slice(0, i + 1).join('/'))}
                className="hover:text-blue-600"
              >
                {part}
              </button>
            </span>
          ))}
        </div>

        {/* File List */}
        <Card>
          <CardContent className="p-0">
            {loading ? (
              <div className="flex items-center justify-center h-64">
                <RefreshCw className="w-6 h-6 animate-spin text-blue-600" />
              </div>
            ) : filteredFiles.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-64 text-muted-foreground">
                <FolderOpen className="w-12 h-12 mb-2 opacity-50" />
                <p>Bu klasör boş</p>
                <p className="text-sm">Dosya yüklemek için sürükle bırak yapın</p>
              </div>
            ) : viewMode === 'list' ? (
              <table className="w-full">
                <thead className="bg-muted/50 border-b">
                  <tr>
                    <th className="w-10 px-3 py-2">
                      <input
                        type="checkbox"
                        checked={selectedFiles.size === files.length && files.length > 0}
                        onChange={selectAll}
                        className="rounded"
                      />
                    </th>
                    <th className="text-left px-3 py-2 text-sm font-medium">Ad</th>
                    <th className="text-left px-3 py-2 text-sm font-medium w-24">Boyut</th>
                    <th className="text-left px-3 py-2 text-sm font-medium w-40">Değiştirilme</th>
                    <th className="text-left px-3 py-2 text-sm font-medium w-28">İzinler</th>
                    <th className="w-20"></th>
                  </tr>
                </thead>
                <tbody>
                  {filteredFiles.map((file) => (
                    <tr 
                      key={file.path}
                      className={`border-b hover:bg-muted/50 cursor-pointer ${
                        selectedFiles.has(file.path) ? 'bg-primary/10' : ''
                      }`}
                      onClick={() => handleFileClick(file)}
                    >
                      <td className="px-3 py-2" onClick={(e) => toggleSelection(file, e)}>
                        <input
                          type="checkbox"
                          checked={selectedFiles.has(file.path)}
                          onChange={() => {}}
                          className="rounded"
                        />
                      </td>
                      <td className="px-3 py-2">
                        <div className="flex items-center gap-2">
                          {getFileIcon(file)}
                          <span className="font-medium">{file.name}</span>
                        </div>
                      </td>
                      <td className="px-3 py-2 text-sm text-muted-foreground">
                        {file.is_dir ? '-' : formatSize(file.size)}
                      </td>
                      <td className="px-3 py-2 text-sm text-muted-foreground">
                        {formatDate(file.modified)}
                      </td>
                      <td className="px-3 py-2 text-sm font-mono text-muted-foreground">
                        {file.permissions}
                      </td>
                      <td className="px-3 py-2">
                        <div className="flex items-center gap-1">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={(e) => {
                              e.stopPropagation();
                              setShowRename(file);
                              setRenameName(file.name);
                            }}
                            title="Yeniden Adlandır"
                          >
                            <Edit3 className="w-4 h-4" />
                          </Button>
                          {file.extension === 'zip' && (
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={(e) => {
                                e.stopPropagation();
                                handleExtract(file);
                              }}
                              title="Çıkar"
                            >
                              <Archive className="w-4 h-4" />
                            </Button>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            ) : (
              <div className="grid grid-cols-4 sm:grid-cols-6 md:grid-cols-8 lg:grid-cols-10 gap-2 p-4">
                {filteredFiles.map((file) => (
                  <div
                    key={file.path}
                    className={`flex flex-col items-center p-3 rounded-lg cursor-pointer hover:bg-muted ${
                      selectedFiles.has(file.path) ? 'bg-primary/10' : ''
                    }`}
                    onClick={() => handleFileClick(file)}
                    onContextMenu={(e) => {
                      e.preventDefault();
                      toggleSelection(file, e);
                    }}
                  >
                    <div className="w-12 h-12 flex items-center justify-center">
                      {file.is_dir ? (
                        <FolderOpen className="w-10 h-10 text-yellow-500" />
                      ) : (
                        getFileIcon(file)
                      )}
                    </div>
                    <span className="text-xs text-center mt-1 truncate w-full">
                      {file.name}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Status Bar */}
        <div className="flex items-center justify-between text-sm text-muted-foreground px-1">
          <span>{files.length} öğe</span>
          {selectedFiles.size > 0 && (
            <span>{selectedFiles.size} seçili</span>
          )}
          {clipboard && (
            <span className="text-blue-600">
              {clipboard.paths.length} öğe {clipboard.action === 'copy' ? 'kopyalandı' : 'kesildi'}
            </span>
          )}
        </div>

        {/* New Folder Modal */}
        <SimpleModal
          isOpen={showNewFolder}
          onClose={() => setShowNewFolder(false)}
          title="Yeni Klasör"
        >
          <Input
            value={newFolderName}
            onChange={(e) => setNewFolderName(e.target.value)}
            placeholder="Klasör adı"
            autoFocus
            onKeyDown={(e) => e.key === 'Enter' && createFolder()}
          />
          <div className="flex justify-end gap-2 mt-4">
            <Button variant="ghost" onClick={() => setShowNewFolder(false)}>İptal</Button>
            <Button onClick={createFolder}>Oluştur</Button>
          </div>
        </SimpleModal>

        {/* New File Modal */}
        <SimpleModal
          isOpen={showNewFile}
          onClose={() => setShowNewFile(false)}
          title="Yeni Dosya"
        >
          <Input
            value={newFileName}
            onChange={(e) => setNewFileName(e.target.value)}
            placeholder="Dosya adı (örn: index.html)"
            autoFocus
            onKeyDown={(e) => e.key === 'Enter' && createFile()}
          />
          <div className="flex justify-end gap-2 mt-4">
            <Button variant="ghost" onClick={() => setShowNewFile(false)}>İptal</Button>
            <Button onClick={createFile}>Oluştur</Button>
          </div>
        </SimpleModal>

        {/* Rename Modal */}
        <SimpleModal
          isOpen={!!showRename}
          onClose={() => setShowRename(null)}
          title="Yeniden Adlandır"
        >
          <Input
            value={renameName}
            onChange={(e) => setRenameName(e.target.value)}
            placeholder="Yeni ad"
            autoFocus
            onKeyDown={(e) => e.key === 'Enter' && handleRename()}
          />
          <div className="flex justify-end gap-2 mt-4">
            <Button variant="ghost" onClick={() => setShowRename(null)}>İptal</Button>
            <Button onClick={handleRename}>Kaydet</Button>
          </div>
        </SimpleModal>

        {/* Editor Modal */}
        <Modal
          isOpen={!!showEditor}
          onClose={() => setShowEditor(null)}
          title={showEditor ? `${showEditor.name} - ${showEditor.path}` : ''}
          size="xl"
          hasUnsavedChanges={showEditor ? editorContent !== showEditor.originalContent : false}
        >
          <div className="flex flex-col h-[70vh]">
            <div className="flex-1 p-4">
              <textarea
                value={editorContent}
                onChange={(e) => setEditorContent(e.target.value)}
                className="w-full h-full font-mono text-sm p-4 border rounded-lg resize-none focus:outline-none focus:ring-2 focus:ring-blue-500 bg-background text-foreground"
                spellCheck={false}
              />
            </div>
            <div className="flex items-center justify-between p-4 border-t">
              <div className="text-sm text-muted-foreground">
                {showEditor && editorContent !== showEditor.originalContent && (
                  <span className="text-orange-500">● Kaydedilmemiş değişiklikler</span>
                )}
              </div>
              <div className="flex gap-2">
                <Button variant="ghost" onClick={() => setShowEditor(null)}>İptal</Button>
                <Button onClick={saveFile}>
                  <Check className="w-4 h-4 mr-1" />
                  Kaydet
                </Button>
              </div>
            </div>
          </div>
        </Modal>

        {/* Image Preview Modal */}
        {showPreview && (
          <div 
            className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-4"
            onClick={() => {
              URL.revokeObjectURL(showPreview.url);
              setShowPreview(null);
            }}
          >
            <div className="relative max-w-4xl max-h-[90vh]" onClick={e => e.stopPropagation()}>
              <Button
                variant="ghost"
                size="sm"
                className="absolute -top-10 right-0 text-white hover:bg-white/20"
                onClick={() => {
                  URL.revokeObjectURL(showPreview.url);
                  setShowPreview(null);
                }}
              >
                <X className="w-6 h-6" />
              </Button>
              <img
                src={showPreview.url}
                alt={showPreview.name}
                className="max-w-full max-h-[85vh] object-contain rounded-lg shadow-2xl"
              />
              <div className="absolute -bottom-10 left-0 right-0 text-center text-white">
                <span className="bg-black/50 px-3 py-1 rounded-full text-sm">
                  {showPreview.name}
                </span>
              </div>
            </div>
          </div>
        )}

        {/* Loading overlay for preview */}
        {previewLoading && (
          <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50">
            <div className="text-white text-center">
              <RefreshCw className="w-8 h-8 animate-spin mx-auto mb-2" />
              <p>Yükleniyor...</p>
            </div>
          </div>
        )}

        {/* Drag overlay */}
        {dragOver && (
          <div className="fixed inset-0 bg-primary/20 flex items-center justify-center z-40 pointer-events-none">
            <div className="bg-card rounded-xl p-8 shadow-xl border">
              <Upload className="w-16 h-16 text-primary mx-auto mb-4" />
              <p className="text-xl font-semibold text-center">Dosyaları buraya bırakın</p>
            </div>
          </div>
        )}
      </div>
    </Layout>
  );
}
