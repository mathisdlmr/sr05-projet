import { useState } from 'react'
import type { GameState, SnapshotEG, SnapshotSiteState } from '../types'

interface Props {
  state: GameState
  send: (action: string, extra?: Record<string, string>) => void
}

const statusLabel = (s: number): string => {
  switch (s) {
    case 1: return 'request'
    case 2: return 'acknowledge'
    case 3: return 'release'
    default: return '?'
  }
}

interface ParsedAppState {
  phase?: string
  players?: Record<string, { id: string; role: string; alive: boolean }>
  votes?: Record<string, string>
}

function SiteCard({ siteId, site }: { siteId: string; site: SnapshotSiteState }) {
  // appState est un GameState sérialisé en JSON côté Go.
  let app: ParsedAppState = {}
  try {
    app = JSON.parse(site.appState) as ParsedAppState
  } catch {
    // Garde un objet vide si le parsing échoue
  }

  const vc = '[' + site.vectorClock.slice(1).join(', ') + ']'
  const players = app.players ?? {}
  const votes = app.votes ?? {}
  const playersList = Object.entries(players)
    .map(([id, p]) => `${id} (${p.role}, ${p.alive ? 'vivant' : 'mort'})`)
    .join(', ')
  const votesList = Object.entries(votes)
    .map(([voter, target]) => `${voter}→${target || '∅'}`)
    .join(', ')

  return (
    <div style={{
      border: '1px solid var(--border)',
      borderRadius: 6,
      padding: 12,
      marginBottom: 8,
      background: 'var(--card-bg)',
      color: 'var(--text)',
    }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
        <strong>Site {siteId}</strong>
        <span style={{ fontFamily: 'monospace', fontSize: '0.85em' }}>VC = {vc}</span>
      </div>
      <div style={{ fontSize: '0.85em', display: 'grid', gap: 4 }}>
        <div><strong>Phase :</strong> {app.phase ?? '?'}</div>
        <div><strong>Bilan Control :</strong> {site.controlState.bilan}</div>
        <div>
          <strong>Queue mutex :</strong>{' '}
          <code style={{ fontSize: '0.85em' }}>
            {site.controlState.queue.slice(1).map((e, i) => (
              `[${i + 1}: ${statusLabel(e.status)}@${e.timestamp}]`
            )).join(' ')}
          </code>
        </div>
        {playersList && (
          <div><strong>Joueurs :</strong> {playersList}</div>
        )}
        {votesList && (
          <div><strong>Votes :</strong> {votesList}</div>
        )}
      </div>
    </div>
  )
}

function SnapshotModal({ eg, onClose }: { eg: SnapshotEG; onClose: () => void }) {
  const siteIds = Object.keys(eg.states).sort()

  return (
    <div
      onClick={onClose}
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.5)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 1000,
        padding: 20,
      }}
    >
      <div
        onClick={(e) => e.stopPropagation()}
        style={{
          background: 'var(--bg)',
          color: 'var(--text)',
          border: '1px solid var(--border)',
          borderRadius: 8,
          padding: 24,
          maxWidth: 800,
          maxHeight: '85vh',
          overflowY: 'auto',
          width: '100%',
          boxShadow: '0 10px 40px rgba(0,0,0,0.5)',
        }}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
          <h2 style={{ margin: 0 }}>Dernière sauvegarde répartie</h2>
          <button className="btn btn-neutral" onClick={onClose}>Fermer</button>
        </div>

        <p style={{ fontSize: '0.9em', color: 'var(--text-muted)', marginTop: 0 }}>
          État global cohérent capturé par l'algorithme 11 (Lai-Yang + reconstitution).
          Date = collection des horloges vectorielles par site.
        </p>

        <h3 style={{ marginTop: 16 }}>États locaux ({siteIds.length} site{siteIds.length > 1 ? 's' : ''})</h3>
        {siteIds.map((id) => (
          <SiteCard key={id} siteId={id} site={eg.states[id]} />
        ))}

        <h3>Messages préposts en transit ({eg.preposts.length})</h3>
        {eg.preposts.length === 0 ? (
          <p style={{ fontSize: '0.9em', color: 'var(--text-muted)' }}>
            Aucun message en transit au moment de la coupe.
          </p>
        ) : (
          <ul style={{ fontFamily: 'monospace', fontSize: '0.8em', paddingLeft: 20 }}>
            {eg.preposts.map((p, i) => (
              <li key={i} style={{ wordBreak: 'break-all', marginBottom: 4 }}>{p}</li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}

export function SnapshotControl({ state, send }: Props) {
  const [showModal, setShowModal] = useState(false)
  const hasSnapshot = state.lastSnapshot !== null

  return (
    <>
      <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
        <button
          className="btn btn-primary"
          onClick={() => send('startSnapshot')}
          title="Déclenche une capture distribuée (algo 11)"
        >
          Sauvegarder
        </button>
        <button
          className="btn btn-neutral"
          onClick={() => setShowModal(true)}
          disabled={!hasSnapshot}
          title={hasSnapshot ? 'Voir la dernière sauvegarde reçue' : 'Aucune sauvegarde reçue pour l\'instant'}
        >
          👁 Voir
        </button>
        {state.snapshotRejection && (
          <span
            style={{
              fontSize: '0.85em',
              color: 'var(--wolf, #c33)',
              padding: '4px 8px',
              border: '1px solid var(--wolf, #c33)',
              borderRadius: 4,
            }}
            title={state.snapshotRejection}
          >
            ⚠ Snapshot refusé : {state.snapshotRejection}
          </span>
        )}
      </div>

      {showModal && state.lastSnapshot && (
        <SnapshotModal eg={state.lastSnapshot} onClose={() => setShowModal(false)} />
      )}
    </>
  )
}
