import { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { api, ApiError } from '../api/client';
import type { Document, AIAnalysis, ExtractedClause } from '../api/client';
import NavBar from '../components/NavBar';

const IMPORTANCE_LABELS: Record<string, string> = {
  high: '🔴 High',
  medium: '🟡 Medium',
  low: '🟢 Low',
};

const CLAUSE_TYPE_LABELS: Record<string, string> = {
  hire_rate: 'Hire Rate', duration: 'Duration', delivery: 'Delivery',
  redelivery: 'Redelivery', payment_terms: 'Payment Terms', laytime: 'Laytime',
  demurrage: 'Demurrage', off_hire: 'Off-Hire', arbitration: 'Arbitration',
  cargo: 'Cargo', bunkers: 'Bunkers', insurance: 'Insurance',
  termination: 'Termination', maintenance: 'Maintenance', other: 'Other',
};

/**
 * PDF text extractors often put each word (or short phrase) on its own line.
 * Strategy: collapse all lines into one stream, then split only at numbered
 * clauses ("1.", "2.", "1)", etc.). All-caps chunks become headings.
 */
function reflowText(raw: string): string {
  const lines = raw.split('\n').map(l => l.trim()).filter(Boolean);

  const chunks: string[] = [];
  let buf = '';

  for (const line of lines) {
    // A new clause/section starts at patterns like "1.", "2.", "14.", "1)"
    // The (\s|$) ensures "10.4" or "1.5" decimals don't trigger this.
    const startsClause = /^\d{1,3}[\.\)](\s|$)/.test(line);

    if (startsClause && buf) {
      chunks.push(buf.trim());
      buf = line;
    } else {
      buf = buf ? buf + ' ' + line : line;
    }
  }
  if (buf) chunks.push(buf.trim());

  return chunks
    .map(chunk => {
      // All-caps chunk (e.g. "BIMCO STANDARD TIME CHARTER PARTY…") → heading
      const isAllCaps =
        chunk.length > 0 &&
        chunk === chunk.toUpperCase() &&
        /[A-Z]/.test(chunk) &&
        chunk.length < 160;
      return isAllCaps ? '\x01' + chunk : chunk;
    })
    .join('\n');
}

export default function DocumentViewer() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const textRef = useRef<HTMLDivElement>(null);

  const [document, setDocument] = useState<Document | null>(null);
  const [analysis, setAnalysis] = useState<AIAnalysis | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [scanState, setScanState] = useState<'idle' | 'extracting' | 'analyzing' | 'done'>('idle');
  const [selectedClause, setSelectedClause] = useState<ExtractedClause | null>(null);
  const [activeTab, setActiveTab] = useState<'clauses' | 'summary' | 'risks'>('clauses');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (id) loadDocument(id);
  }, [id]);

  const loadDocument = async (docId: string) => {
    try {
      let doc = await api.documents.get(docId);

      // Auto-extract text for free if not done yet (just PDF→text, no AI)
      if (doc.status === 'uploaded') {
        doc = await api.documents.process(docId);
      }

      setDocument(doc);
      if (doc.ai_analysis) {
        setAnalysis(doc.ai_analysis);
        setScanState('done');
      } else if (doc.status === 'processing') {
        setScanState('extracting');
      }
    } catch (e) {
      if (e instanceof ApiError && e.status === 404) {
        navigate('/documents');
        return;
      }
      setError('Failed to load document');
    } finally {
      setIsLoading(false);
    }
  };

  // Single "Scan" action: process (if needed) + analyze
  const handleScan = async () => {
    if (!id || !document) return;
    setError(null);

    try {
      setScanState('extracting');

      // If not yet extracted, process first (free — no AI)
      if (document.status === 'uploaded') {
        const processed = await api.documents.process(id);
        setDocument(processed);
      }

      setScanState('analyzing');
      const result = await api.documents.analyze(id);
      setAnalysis(result.analysis);
      if (result.document) setDocument(result.document);
      setScanState('done');
    } catch (e) {
      setScanState('idle');
      if (e instanceof ApiError && e.status === 503) {
        setError('AI not configured — add a DeepSeek or OpenAI key in config.local.yaml to use this feature.');
      } else {
        setError(e instanceof ApiError ? e.message : 'Scan failed');
      }
    }
  };

  // Highlight text in the document panel when a clause is selected
  const highlightClause = (clause: ExtractedClause) => {
    if (!textRef.current || !clause.content) return;
    const el = textRef.current;
    const text = el.innerText;
    const idx = text.indexOf(clause.content.slice(0, 60));
    if (idx !== -1) {
      // Rough scroll: find element position
      const range = window.document.createRange();
      const walker = window.document.createTreeWalker(el, NodeFilter.SHOW_TEXT);
      let charCount = 0;
      let node: Node | null;
      while ((node = walker.nextNode())) {
        const nodeLen = node.textContent?.length ?? 0;
        if (charCount + nodeLen >= idx) {
          range.setStart(node, idx - charCount);
          range.collapse(true);
          const rect = range.getBoundingClientRect();
          const parentRect = el.getBoundingClientRect();
          el.scrollTop += rect.top - parentRect.top - 120;
          break;
        }
        charCount += nodeLen;
      }
    }
  };

  const handleSelectClause = (clause: ExtractedClause) => {
    const next = selectedClause === clause ? null : clause;
    setSelectedClause(next);
    if (next) highlightClause(next);
  };

  const formatDocumentText = (text: string) => {
    const reflowed = reflowText(text);
    return reflowed
      .split('\n')
      .map((line, i) => {
        if (!line) return <div key={i} className="doc-spacer" />;
        if (line.startsWith('\x01')) return <h3 key={i} className="doc-heading">{line.slice(1)}</h3>;
        return <p key={i} className="doc-paragraph">{line}</p>;
      });
  };

  if (isLoading) {
    return (
      <div className="loading-container" style={{ paddingTop: '5rem' }}>
        <div className="loading-spinner" />
        <p>Loading document…</p>
      </div>
    );
  }

  if (!document) {
    return (
      <div className="error-container">
        <p>Document not found</p>
        <button className="btn-primary" onClick={() => navigate('/documents')}>Back to Documents</button>
      </div>
    );
  }

  const canScan = document.status === 'uploaded' || document.status === 'processed';
  const isScanning = scanState === 'extracting' || scanState === 'analyzing';

  return (
    <div className="doc-viewer-shell">
      <NavBar backTo="/documents" backLabel="Documents" />

      {/* ── Toolbar ── */}
      <div className="doc-toolbar">
        <div className="doc-toolbar-left">
          <span className="doc-filename">📄 {document.original_filename}</span>
          <span className={`badge badge-${document.status}`}>{document.status}</span>
        </div>
        <div className="doc-toolbar-right">
          {error && (
            <span className="doc-error-inline" onClick={() => setError(null)} title="Click to dismiss">
              ⚠ {error}
            </span>
          )}
          {canScan && !analysis && (
            <button className="btn-scan" onClick={handleScan} disabled={isScanning}>
              {isScanning ? (
                <><span className="btn-spinner" /> {scanState === 'extracting' ? 'Extracting text…' : 'Scanning clauses…'}</>
              ) : (
                '🔍 Scan for Negotiation Points'
              )}
            </button>
          )}
          {analysis && (
            <button className="btn-secondary btn-sm" onClick={handleScan} disabled={isScanning}>
              {isScanning ? 'Re-scanning…' : '↺ Re-scan'}
            </button>
          )}
        </div>
      </div>

      {/* ── Two panels ── */}
      <div className="doc-panels">

        {/* LEFT — paper document */}
        <div className="doc-paper-panel" ref={textRef}>
          {document.extracted_text ? (
            <div className="doc-paper">
              <div className="doc-paper-inner">
                {formatDocumentText(document.extracted_text)}
              </div>
            </div>
          ) : (
            <div className="doc-paper-empty">
              <div style={{ fontSize: '3rem', marginBottom: '1rem' }}>📄</div>
              <p>Text will appear here after scanning.</p>
              {canScan && (
                <button className="btn-primary" style={{ marginTop: '1rem' }} onClick={handleScan} disabled={isScanning}>
                  {isScanning ? 'Scanning…' : '🔍 Scan Document'}
                </button>
              )}
            </div>
          )}
        </div>

        {/* RIGHT — AI analysis */}
        <div className="doc-analysis-panel">
          {isScanning ? (
            <div className="analysis-scanning">
              <div className="loading-spinner" style={{ margin: '2rem auto' }} />
              <p className="analysis-scanning-label">
                {scanState === 'extracting' ? 'Extracting document text…' : 'AI is reading your charter party…'}
              </p>
              <p className="hint" style={{ textAlign: 'center' }}>
                {scanState === 'analyzing' ? 'Identifying key negotiation clauses, risks, and suggestions. This takes about 30–60 seconds.' : ''}
              </p>
            </div>
          ) : analysis ? (
            <>
              <div className="analysis-tabs">
                <button
                  className={`analysis-tab${activeTab === 'clauses' ? ' analysis-tab--active' : ''}`}
                  onClick={() => setActiveTab('clauses')}
                >
                  Clauses <span className="tab-count">{analysis.clauses.length}</span>
                </button>
                <button
                  className={`analysis-tab${activeTab === 'summary' ? ' analysis-tab--active' : ''}`}
                  onClick={() => setActiveTab('summary')}
                >
                  Summary
                </button>
                <button
                  className={`analysis-tab${activeTab === 'risks' ? ' analysis-tab--active' : ''}`}
                  onClick={() => setActiveTab('risks')}
                >
                  Risks {analysis.risk_factors?.length ? <span className="tab-count">{analysis.risk_factors.length}</span> : null}
                </button>
              </div>

              <div className="analysis-tab-content">

                {activeTab === 'clauses' && (
                  <div className="analysis-clauses">
                    {analysis.clauses.length === 0 ? (
                      <p className="hint" style={{ padding: '1rem' }}>No clauses extracted.</p>
                    ) : (
                      analysis.clauses.map((clause, i) => (
                        <div
                          key={i}
                          className={`analysis-clause${selectedClause === clause ? ' analysis-clause--active' : ''} clause-imp-${clause.importance}`}
                          onClick={() => handleSelectClause(clause)}
                        >
                          <div className="analysis-clause-header">
                            <span className="analysis-clause-type">{CLAUSE_TYPE_LABELS[clause.type] ?? clause.type}</span>
                            <span className="analysis-clause-imp">{IMPORTANCE_LABELS[clause.importance] ?? clause.importance}</span>
                          </div>
                          <h4 className="analysis-clause-title">{clause.title}</h4>
                          <p className="analysis-clause-summary">{clause.summary}</p>

                          {selectedClause === clause && (
                            <div className="analysis-clause-expanded">
                              <p className="analysis-clause-excerpt">{clause.content}</p>
                              {clause.key_points && clause.key_points.length > 0 && (
                                <div className="analysis-clause-points">
                                  <strong>Negotiation points:</strong>
                                  <ul>
                                    {clause.key_points.map((pt, j) => <li key={j}>{pt}</li>)}
                                  </ul>
                                </div>
                              )}
                            </div>
                          )}
                        </div>
                      ))
                    )}
                  </div>
                )}

                {activeTab === 'summary' && (
                  <div className="analysis-summary-tab">
                    <p>{analysis.summary || 'No summary available.'}</p>
                    {analysis.suggestions && analysis.suggestions.length > 0 && (
                      <>
                        <h4 style={{ marginTop: '1.5rem', marginBottom: '0.5rem' }}>Negotiation Suggestions</h4>
                        <ul className="analysis-suggestions">
                          {analysis.suggestions.map((s, i) => <li key={i}>{s}</li>)}
                        </ul>
                      </>
                    )}
                  </div>
                )}

                {activeTab === 'risks' && (
                  <div className="analysis-risks-tab">
                    {analysis.risk_factors && analysis.risk_factors.length > 0 ? (
                      <ul className="analysis-risks">
                        {analysis.risk_factors.map((r, i) => (
                          <li key={i} className="risk-item">
                            <span className="risk-icon">⚠</span>
                            {r}
                          </li>
                        ))}
                      </ul>
                    ) : (
                      <p className="hint" style={{ padding: '1rem' }}>No risk factors identified.</p>
                    )}
                  </div>
                )}

              </div>
            </>
          ) : (
            <div className="analysis-empty">
              <div style={{ fontSize: '2.5rem', marginBottom: '1rem' }}>🤖</div>
              <h3>AI Analysis</h3>
              <p>Click <strong>Scan for Negotiation Points</strong> to have AI read your charter party and highlight:</p>
              <ul>
                <li>Key commercial clauses (hire, demurrage, laytime…)</li>
                <li>Risk factors</li>
                <li>Negotiation suggestions</li>
              </ul>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
