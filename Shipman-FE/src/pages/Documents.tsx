import { useState, useEffect, useRef } from 'react';
import type { ChangeEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, ApiError } from '../api/client';
import type { Document } from '../api/client';
import { useAuth } from '../context/AuthContext';
import NavBar from '../components/NavBar';

export default function Documents() {
  const [documents, setDocuments] = useState<Document[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isUploading, setIsUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const navigate = useNavigate();
  const { signout } = useAuth();

  useEffect(() => {
    loadDocuments();
  }, []);

  const loadDocuments = async () => {
    try {
      const response = await api.documents.list();
      setDocuments(response.data);
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) {
        signout();
        navigate('/login');
        return;
      }
      setError('Failed to load documents');
    } finally {
      setIsLoading(false);
    }
  };

  const handleUpload = async (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setIsUploading(true);
    setError(null);

    try {
      const doc = await api.documents.upload(file);
      setDocuments([doc, ...documents]);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Upload failed');
    } finally {
      setIsUploading(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  const handleProcess = async (id: string) => {
    try {
      const updated = await api.documents.process(id);
      setDocuments(documents.map(d => d.id === id ? updated : d));
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Processing failed');
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this document?')) return;

    try {
      await api.documents.delete(id);
      setDocuments(documents.filter(d => d.id !== id));
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Delete failed');
    }
  };

  const formatFileSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  const getStatusBadge = (status: Document['status']) => {
    const styles: Record<Document['status'], string> = {
      uploaded: 'badge-info',
      processing: 'badge-warning',
      processed: 'badge-success',
      failed: 'badge-error',
    };
    return `badge ${styles[status]}`;
  };

  return (
    <div className="documents-page">
      <NavBar />
      <div className="page-content">
      <header className="page-header">
        <div>
          <h1>Charter Party Documents</h1>
          <p>Upload and analyze charter party documents with AI-powered clause extraction.</p>
        </div>
        <div className="header-actions">
          <input
            type="file"
            ref={fileInputRef}
            onChange={handleUpload}
            accept=".pdf,.doc,.docx,.txt"
            style={{ display: 'none' }}
          />
          <button
            className="btn-primary"
            onClick={() => fileInputRef.current?.click()}
            disabled={isUploading}
          >
            {isUploading ? 'Uploading...' : 'Upload Document'}
          </button>
        </div>
      </header>

      {error && (
        <div className="error-banner" onClick={() => setError(null)}>
          {error}
        </div>
      )}

      {isLoading ? (
        <div className="loading-container">
          <div className="loading-spinner" />
          <p>Loading documents...</p>
        </div>
      ) : documents.length === 0 ? (
        <div className="empty-state">
          <div className="empty-icon">📄</div>
          <h3>No documents yet</h3>
          <p>Upload a charter party document to get started with AI-powered analysis.</p>
          <button
            className="btn-primary"
            onClick={() => fileInputRef.current?.click()}
          >
            Upload Your First Document
          </button>
        </div>
      ) : (
        <div className="documents-list">
          {documents.map(doc => (
            <div key={doc.id} className="document-card">
              <div className="document-info">
                <div className="document-icon">
                  {doc.content_type === 'application/pdf' ? '📕' : '📄'}
                </div>
                <div className="document-details">
                  <h3>{doc.original_filename}</h3>
                  <div className="document-meta">
                    <span>{formatFileSize(doc.file_size)}</span>
                    <span className={getStatusBadge(doc.status)}>{doc.status}</span>
                    <span>{new Date(doc.created_at).toLocaleDateString()}</span>
                  </div>
                </div>
              </div>
              <div className="document-actions">
                {doc.status === 'uploaded' && (
                  <button
                    className="btn-secondary"
                    onClick={() => handleProcess(doc.id)}
                  >
                    Extract Text
                  </button>
                )}
                {doc.status === 'processed' && (
                  <button
                    className="btn-primary"
                    onClick={() => navigate(`/documents/${doc.id}`)}
                  >
                    View & Analyze
                  </button>
                )}
                <button
                  className="btn-danger"
                  onClick={() => handleDelete(doc.id)}
                >
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
      </div>
    </div>
  );
}
