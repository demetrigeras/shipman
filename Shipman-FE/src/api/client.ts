const API_BASE = '/api/v1';

export interface User {
  id: string;
  email: string;
  full_name: string;
  role: 'shipowner' | 'charterer' | 'broker';
  created_at: string;
  updated_at: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export interface SignupData {
  email: string;
  password: string;
  full_name: string;
  role: 'shipowner' | 'charterer' | 'broker';
}

export interface SigninData {
  email: string;
  password: string;
}

class ApiError extends Error {
  status: number;
  
  constructor(status: number, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

async function request<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const token = localStorage.getItem('token');
  
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  if (options.headers) {
    Object.assign(headers, options.headers);
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Request failed' }));
    const msg = [error.error, error.details].filter(Boolean).join(' — ') || 'Request failed';
    throw new ApiError(response.status, msg);
  }

  return response.json();
}

export interface Document {
  id: string;
  charter_detail_id?: string;
  uploaded_by: string;
  filename: string;
  original_filename: string;
  content_type: string;
  file_size: number;
  status: 'uploaded' | 'processing' | 'processed' | 'failed';
  extracted_text?: string;
  ai_analysis?: AIAnalysis;
  created_at: string;
  updated_at: string;
}

export interface ExtractedClause {
  type: string;
  title: string;
  content: string;
  importance: 'high' | 'medium' | 'low';
  summary: string;
  key_points?: string[];
}

export interface AIAnalysis {
  clauses: ExtractedClause[];
  summary: string;
  risk_factors?: string[];
  suggestions?: string[];
}

export interface DocumentListResponse {
  data: Document[];
}

export interface AnalyzeResponse {
  document_id: string;
  analysis: AIAnalysis;
}

async function uploadFile(
  endpoint: string,
  file: File,
  charterDetailId?: string
): Promise<Document> {
  const token = localStorage.getItem('token');
  const formData = new FormData();
  formData.append('file', file);
  if (charterDetailId) {
    formData.append('charter_detail_id', charterDetailId);
  }

  const headers: Record<string, string> = {};
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE}${endpoint}`, {
    method: 'POST',
    headers,
    body: formData,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Upload failed' }));
    throw new ApiError(response.status, error.error || 'Upload failed');
  }

  return response.json();
}

export interface Deal {
  id: string;
  title: string;
  description?: string;
  document_id?: string;
  status: 'active' | 'completed' | 'cancelled';
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface DealParticipant {
  id: string;
  deal_id: string;
  user_id?: string;
  role: 'shipowner' | 'charterer' | 'broker';
  joined_at?: string;
  user?: User;
}

export interface ClauseNegotiation {
  id: string;
  deal_id: string;
  clause_type: string;
  clause_title: string;
  original_content: string;
  status: 'pending' | 'accepted' | 'rejected' | 'countered';
  created_at: string;
}

export interface ClauseProposal {
  id: string;
  negotiation_id: string;
  proposed_by: string;
  proposed_content: string;
  comment?: string;
  status: 'pending' | 'accepted' | 'rejected' | 'superseded';
  created_at: string;
  proposed_by_user?: User;
}

export interface DealVesselDetails {
  id: string;
  deal_id: string;
  filled_by: string;
  vessel_name?: string;
  imo_number?: string;
  vessel_type?: string;
  flag_state?: string;
  deadweight_tonnage?: number;
  gross_tonnage?: number;
  build_year?: number;
  class_society?: string;
  current_position?: string;
  available_from?: string;
  asking_rate?: number;
  asking_rate_currency: string;
  asking_rate_type: string;
  notes?: string;
  created_at: string;
  updated_at: string;
}

export interface DealCargoDetails {
  id: string;
  deal_id: string;
  filled_by: string;
  commodity?: string;
  quantity?: number;
  quantity_unit: string;
  load_port?: string;
  discharge_port?: string;
  laycan_from?: string;
  laycan_to?: string;
  freight_idea?: number;
  freight_currency: string;
  freight_type: string;
  special_requirements?: string;
  notes?: string;
  created_at: string;
  updated_at: string;
}

export interface DealWithParticipants {
  deal: Deal;
  participants: DealParticipant[];
  vessel_details?: DealVesselDetails | null;
  cargo_details?: DealCargoDetails | null;
}

export interface NegotiationWithProposals extends ClauseNegotiation {
  proposals: ClauseProposal[];
}

export interface Vessel {
  id: string;
  name: string;
  imo_number?: string;
  flag_state?: string;
  vessel_type?: string;
  call_sign?: string;
  deadweight_tonnage?: number;
  gross_tonnage?: number;
  build_year?: number;
  owner?: string;
  notes?: string;
  created_at: string;
  updated_at: string;
}

export interface VesselCreateData {
  name: string;
  imo_number?: string;
  flag_state?: string;
  vessel_type?: string;
  deadweight_tonnage?: number;
  gross_tonnage?: number;
  build_year?: number;
  owner?: string;
  notes?: string;
}

export const api = {
  auth: {
    signup: (data: SignupData) =>
      request<AuthResponse>('/users/signup', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    
    signin: (data: SigninData) =>
      request<AuthResponse>('/users/signin', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    
    me: () => request<User>('/users/me'),
  },

  documents: {
    upload: (file: File, charterDetailId?: string) =>
      uploadFile('/documents', file, charterDetailId),
    
    list: (limit = 20, offset = 0) =>
      request<DocumentListResponse>(`/documents?limit=${limit}&offset=${offset}`),
    
    get: (id: string) =>
      request<Document>(`/documents/${id}`),
    
    process: (id: string) =>
      request<Document>(`/documents/${id}/process`, { method: 'POST' }),
    
    analyze: (id: string) =>
      request<AnalyzeResponse>(`/documents/${id}/analyze`, { method: 'POST' }),
    
    delete: (id: string) =>
      request<{ message: string }>(`/documents/${id}`, { method: 'DELETE' }),
  },

  deals: {
    create: (data: { title: string; description?: string; document_id?: string }) =>
      request<Deal>('/deals', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    
    list: () =>
      request<{ data: Deal[] }>('/deals'),
    
    get: (id: string) =>
      request<DealWithParticipants>(`/deals/${id}`),
    
    createInvite: (id: string, email: string, role: 'shipowner' | 'charterer' | 'broker') =>
      request<{ invite_token: string; invite_link: string; expires_at: string; role: string; email_sent: boolean }>(`/deals/${id}/invite`, {
        method: 'POST',
        body: JSON.stringify({ email, role }),
      }),

    join: (token: string) =>
      request<{ message: string; deal: Deal; role: string }>('/deals/join', {
        method: 'POST',
        body: JSON.stringify({ token }),
      }),

    previewInvite: (token: string) =>
      request<{ token: string; role: string; deal_id: string; deal_title: string; expires_at: string }>(`/deals/invite/${token}`),
    
    createNegotiation: (dealId: string, data: { clause_type: string; clause_title: string; original_content: string }) =>
      request<ClauseNegotiation>(`/deals/${dealId}/negotiations`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    
    listNegotiations: (dealId: string) =>
      request<{ data: ClauseNegotiation[] }>(`/deals/${dealId}/negotiations`),
    
    getNegotiation: (dealId: string, negotiationId: string) =>
      request<NegotiationWithProposals>(`/deals/${dealId}/negotiations/${negotiationId}`),
    
    createProposal: (dealId: string, negotiationId: string, data: { proposed_content: string; comment?: string }) =>
      request<ClauseProposal>(`/deals/${dealId}/negotiations/${negotiationId}/proposals`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    
    updateProposalStatus: (dealId: string, negotiationId: string, proposalId: string, status: 'accepted' | 'rejected') =>
      request<{ message: string }>(`/deals/${dealId}/negotiations/${negotiationId}/proposals/${proposalId}`, {
        method: 'PATCH',
        body: JSON.stringify({ status }),
      }),

    upsertVesselDetails: (dealId: string, data: Partial<Omit<DealVesselDetails, 'id' | 'deal_id' | 'filled_by' | 'created_at' | 'updated_at'>>) =>
      request<DealVesselDetails>(`/deals/${dealId}/vessel`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),

    upsertCargoDetails: (dealId: string, data: Partial<Omit<DealCargoDetails, 'id' | 'deal_id' | 'filled_by' | 'created_at' | 'updated_at'>>) =>
      request<DealCargoDetails>(`/deals/${dealId}/cargo`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),

    attachDocument: (dealId: string, documentId: string) =>
      request<{ message: string }>(`/deals/${dealId}/document`, {
        method: 'PATCH',
        body: JSON.stringify({ document_id: documentId }),
      }),
  },

  marketplace: {
    listVessels: (limit = 20, offset = 0) =>
      request<{ data: Vessel[] }>(`/marketplace/vessels?limit=${limit}&offset=${offset}`),
    
    getVessel: (id: string) =>
      request<Vessel>(`/marketplace/vessels/${id}`),
    
    createVessel: (data: VesselCreateData) =>
      request<Vessel>('/marketplace/vessels', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    
    updateVessel: (id: string, data: VesselCreateData) =>
      request<Vessel>(`/marketplace/vessels/${id}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    
    deleteVessel: (id: string) =>
      request<{ message: string }>(`/marketplace/vessels/${id}`, { method: 'DELETE' }),
  },
};

export { ApiError };
