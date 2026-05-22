const RAW_API_BASE = (import.meta.env?.VITE_API_BASE_URL ?? '/api/v1') as string;
const API_BASE = RAW_API_BASE.replace(/\/+$/, '');

export interface ExtractedTerms {
  vessel_name?: string;
  imo_number?: string;
  vessel_type?: string;
  dwt?: number;
  flag_state?: string;
  hire_rate?: number;
  freight_rate?: number;
  cargo_type?: string;
  cargo_quantity?: number;
  load_port?: string;
  discharge_port?: string;
  laytime_allowed_hours?: number;
  demurrage_rate?: number;
  despatch_rate?: number;
  currency?: string;
  raw_summary?: string;
}

export interface Voyage {
  id: string;
  deal_id?: string;
  owner_user_id?: string;
  document_id?: string;
  charter_type?: string;
  voyage_number?: string;
  vessel_name?: string;
  imo_number?: string;
  vessel_type?: string;
  dwt?: number;
  flag_state?: string;
  departure_port?: string;
  arrival_port?: string;
  planned_departure_at?: string;
  planned_arrival_at?: string;
  actual_departure_at?: string;
  actual_arrival_at?: string;
  hire_rate?: number;
  freight_rate?: number;
  cargo_quantity?: number;
  cargo_type?: string;
  laytime_allowed_hours?: number;
  demurrage_rate?: number;
  despatch_rate?: number;
  demurrage_currency: string;
  payment_frequency?: string;
  first_payment_date?: string;
  total_contract_value?: number;
  commission_rate?: number;
  bunker_cost?: number;
  port_costs?: number;
  insurance_cost?: number;
  counterparty_name?: string;
  counterparty_email?: string;
  parties?: VoyageParties;
  status: string;
  notes?: string;
  created_at: string;
  updated_at: string;
}

export interface VoyageParty {
  user_id?: string;
  email?: string;
  full_name?: string;
}

export interface VoyageParties {
  owner?: VoyageParty;
  counterparty?: VoyageParty;
  broker?: VoyageParty;
}

export interface ShipPosition {
  id: string;
  voyage_id: string;
  recorded_at: string;
  latitude: number;
  longitude: number;
  speed_knots?: number;
  heading?: number;
  distance_logged_nm?: number;
  fuel_remaining_mt?: number;
  source: string;
  remarks?: string;
  created_at: string;
}

export interface LaytimeEntry {
  id: string;
  voyage_id?: string;
  port_name: string;
  activity: string;
  started_at: string;
  ended_at?: string;
  hours_counted?: number;
  remarks?: string;
  created_at: string;
}

export interface LaytimeSummary {
  total_hours_used: number;
  total_hours_allowed: number;
  balance_hours: number;
  demurrage_hours: number;
  despatch_hours: number;
  demurrage_amount?: number;
  despatch_amount?: number;
  currency: string;
}

export interface User {
  id: string;
  email: string;
  full_name: string;
  role: 'shipowner' | 'charterer' | 'broker';
  coinsub_merchant_id?: string;
  wallet_address?: string;
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
  shipowner_user_id?: string;
  charterer_user_id?: string;
  broker_user_id?: string;
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
  status: 'pending' | 'open' | 'accepted' | 'rejected' | 'countered';
  sort_order: number;
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
  pending_invites: DealInviteSummary[];
  vessel_details?: DealVesselDetails | null;
  cargo_details?: DealCargoDetails | null;
}

export interface DealInviteSummary {
  id: string;
  deal_id: string;
  role: 'shipowner' | 'charterer' | 'broker';
  invited_email: string;
  created_at: string;
  expires_at: string;
}

export interface NegotiationWithProposals extends ClauseNegotiation {
  proposals: ClauseProposal[];
}

export interface VoyagePayment {
  id: string;
  voyage_id: string;
  created_by: string;
  payment_type: 'hire' | 'freight' | 'demurrage' | 'despatch' | 'bunker' | 'port_charges' | 'other';
  description?: string;
  amount: number;
  currency: string;
  recipient_email?: string;
  coinsub_session_id?: string;
  coinsub_payment_id?: string;
  coinsub_checkout_url?: string;
  coinsub_tx_hash?: string;
  coinsub_agreement_id?: string;
  status: 'draft' | 'pending' | 'completed' | 'failed' | 'cancelled';
  paid_at?: string;
  created_at: string;
  updated_at: string;
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

    cancelInvite: (dealId: string, inviteId: string) =>
      request<{ status: string }>(`/deals/${dealId}/invites/${inviteId}`, {
        method: 'DELETE',
      }),

    join: (token: string) =>
      request<{ message: string; deal: Deal; role: string }>('/deals/join', {
        method: 'POST',
        body: JSON.stringify({ token }),
      }),

    previewInvite: (token: string) =>
      request<{ token: string; role: string; deal_id: string; deal_title: string; invited_email: string; expires_at: string }>(`/deals/invite/${token}`),
    
    createNegotiation: (dealId: string, data: { clause_type: string; clause_title: string; original_content: string; sort_order?: number }) =>
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
      request<{ message: string; deal_completed: boolean }>(`/deals/${dealId}/negotiations/${negotiationId}/proposals/${proposalId}`, {
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

  voyages: {
    list: () => request<Voyage[]>('/voyages'),
    get: (id: string) => request<Voyage>(`/voyages/${id}`),
    create: (data: Partial<Voyage> & { deal_id?: string }) =>
      request<Voyage>('/voyages', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: Partial<Voyage> & { clear_document?: boolean }) =>
      request<Voyage>(`/voyages/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    delete: (id: string) =>
      request<{ message: string }>(`/voyages/${id}`, { method: 'DELETE' }),

    listPositions: (id: string) => request<ShipPosition[]>(`/voyages/${id}/positions`),
    addPosition: (id: string, data: Partial<ShipPosition> & { recorded_at: string }) =>
      request<ShipPosition>(`/voyages/${id}/positions`, { method: 'POST', body: JSON.stringify(data) }),
    getLivePosition: (id: string) =>
      request<{ source: string; position: ShipPosition | null; hint?: string }>(`/voyages/${id}/position/live`),

    listLaytime: (id: string) => request<LaytimeEntry[]>(`/voyages/${id}/laytime`),
    addLaytime: (id: string, data: Partial<LaytimeEntry> & { port_name: string; activity: string; started_at: string }) =>
      request<LaytimeEntry>(`/voyages/${id}/laytime`, { method: 'POST', body: JSON.stringify(data) }),
    updateLaytime: (id: string, entryId: string, data: Partial<LaytimeEntry>) =>
      request<LaytimeEntry>(`/voyages/${id}/laytime/${entryId}`, { method: 'PATCH', body: JSON.stringify(data) }),
    deleteLaytime: (id: string, entryId: string) =>
      request<{ message: string }>(`/voyages/${id}/laytime/${entryId}`, { method: 'DELETE' }),
    getLaytimeSummary: (id: string) => request<LaytimeSummary>(`/voyages/${id}/laytime/summary`),

    attachDocument: (id: string, documentId: string) =>
      request<{ message: string; document_id: string }>(`/voyages/${id}/attach-document`, {
        method: 'POST', body: JSON.stringify({ document_id: documentId }),
      }),
    extractTerms: (id: string, documentId?: string) =>
      request<ExtractedTerms>(`/voyages/${id}/extract-terms`, {
        method: 'POST',
        body: JSON.stringify(documentId ? { document_id: documentId } : {}),
      }),
    /** Scan charter party text without creating a voyage (uses processed document). */
    extractTermsPreview: (documentId: string) =>
      request<ExtractedTerms>('/voyages/extract-terms-preview', {
        method: 'POST',
        body: JSON.stringify({ document_id: documentId }),
      }),

    listPayments: (id: string) => request<VoyagePayment[]>(`/voyages/${id}/payments`),
    createPayment: (id: string, data: { payment_type: string; name?: string; description?: string; amount: number; currency?: string }) =>
      request<VoyagePayment>(`/voyages/${id}/payments`, { method: 'POST', body: JSON.stringify(data) }),
    checkoutPayment: (voyageId: string, paymentId: string, opts?: { recurring?: boolean; interval?: string; frequency?: string; duration?: string }) =>
      request<{ checkout_url: string; session_id: string }>(`/voyages/${voyageId}/payments/${paymentId}/checkout`, {
        method: 'POST', body: JSON.stringify(opts ?? {}),
      }),
    transferPayment: (voyageId: string, paymentId: string, toAddress: string, chainId?: number, token?: string) =>
      request<{ message: string; fee: number }>(`/voyages/${voyageId}/payments/${paymentId}/transfer`, {
        method: 'POST', body: JSON.stringify({ to_address: toAddress, chain_id: chainId, token }),
      }),
    deletePayment: (voyageId: string, paymentId: string) =>
      request<{ message: string }>(`/voyages/${voyageId}/payments/${paymentId}`, { method: 'DELETE' }),
    markPaid: (voyageId: string, paymentId: string) =>
      request<VoyagePayment>(`/voyages/${voyageId}/payments/${paymentId}/mark-paid`, { method: 'POST' }),

    createInvite: (id: string, inviteEmail: string, role: string) =>
      request<{ invite_token: string; invite_link: string; expires_at: string; role: string; email_sent: boolean }>(
        `/voyages/${id}/invite`,
        { method: 'POST', body: JSON.stringify({ email: inviteEmail, role }) },
      ),
    previewInvite: (token: string) =>
      request<{ token: string; type: string; role: string; voyage_id: string; fixture_title: string; invited_email: string; expires_at: string }>(
        `/voyages/invite/${token}`,
      ),
    joinVoyage: (token: string) =>
      request<{ message: string; voyage_id: string; role: string }>(
        '/voyages/join',
        { method: 'POST', body: JSON.stringify({ token }) },
      ),
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

  admin: {
    coinsubStatus: () =>
      request<{ enabled: boolean; merchant_id: string; webhook_url: string }>('/admin/coinsub/status'),

    registerWebhook: (webhookUrl: string) =>
      request<{ message: string; webhook_id: number; signing_secret: string; status: string; note: string }>(
        '/admin/coinsub/register-webhook',
        { method: 'POST', body: JSON.stringify({ webhook_url: webhookUrl }) },
      ),
  },

  payments: {
    // Mint a fresh single-use RocketRamp embed code for the given recipient.
    // Prefer `embed_url` (full popup URL); `embed_base_url`+`embed_code` is
    // kept around for older callers that build the URL themselves.
    createEmbedCode: (recipientEmail: string, memo?: string) =>
      request<{ embed_code: string; embed_base_url: string; embed_url: string; test_mode: boolean }>(
        '/payments/embed-code',
        {
          method: 'POST',
          body: JSON.stringify({ recipient_email: recipientEmail, memo: memo ?? '' }),
        },
      ),

    embedConfig: () =>
      request<{ enabled: boolean; test_mode: boolean; embed_base_url: string }>('/payments/embed-config'),
  },
};

export { ApiError };
