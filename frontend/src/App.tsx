import { useState } from 'react'
import { useGame } from './useGame'
import type { GameState } from './types'
import { Lobby } from './components/Lobby'
import { NightPhase } from './components/NightPhase'
import { VotePhase } from './components/VotePhase'
import { EndPhase } from './components/EndPhase'
import { SnapshotControl } from './components/SnapshotControl'

function roleLabel(role: string): string {
  switch (role) {
    case 'WOLF':     return '🐺 Loup'
    case 'WITCH':    return '🧙 Sorcière'
    case 'VILLAGER': return '🏡 Villageois'
    default:         return '❓ Inconnu'
  }
}

function QuitButton({ send }: { send: (action: string) => void }) {
  const [confirming, setConfirming] = useState(false)

  if (!confirming) {
    return (
      <button className="btn btn-danger" style={{ fontSize: '0.75rem', padding: '4px 10px' }} onClick={() => setConfirming(true)}>
        Quitter
      </button>
    )
  }

  return (
    <span style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '0.8rem' }}>
      Confirmer ?
      <button className="btn btn-danger" style={{ fontSize: '0.75rem', padding: '4px 10px' }} onClick={() => send('quit')}>
        Oui
      </button>
      <button className="btn btn-neutral" style={{ fontSize: '0.75rem', padding: '4px 10px' }} onClick={() => setConfirming(false)}>
        Non
      </button>
    </span>
  )
}

function Header({ state, send }: { state: GameState; send: (action: string, extra?: Record<string, string>) => void }) {
  const statusClass =
    state.wsStatus === 'connected' ? 'connected' :
    state.wsStatus === 'disconnected' ? 'disconnected' : 'connecting'

  return (
    <header className="header">
      <span className="header-title">🐺 Loup-Garou</span>
      <div className="header-info">
        {state.myId && (
          <span className="header-player">
            <strong>{state.myId}</strong>
            {state.myRole !== '?' && (
              <span className={`role-badge ${state.myRole}`}>
                {roleLabel(state.myRole)}
              </span>
            )}
            {!state.myAlive && <span style={{ color: 'var(--wolf)' }}>💀</span>}
          </span>
        )}
        <span>
          <span className={`status-dot ${statusClass}`} />
          {' '}
          {state.wsStatus === 'connected' ? 'Connecté' :
           state.wsStatus === 'disconnected' ? 'Déconnecté' : 'Connexion...'}
        </span>
        <SnapshotControl state={state} send={send} />
        {state.wsStatus === 'connected' && <QuitButton send={send} />}
      </div>
    </header>
  )
}

export default function App() {
  const [state, send] = useGame()

  if (state.wsStatus === 'disconnected') {
    return (
      <div className="fullscreen-state">
        <div style={{ fontSize: '3rem' }}>🔌</div>
        <h2>Connexion perdue</h2>
        <p>Le serveur est inaccessible.</p>
        <button className="btn btn-neutral" onClick={() => window.location.reload()}>
          Réessayer
        </button>
      </div>
    )
  }

  if (state.wsStatus === 'connecting' || !state.myId) {
    return (
      <div className="fullscreen-state">
        <div style={{ fontSize: '3rem', animation: 'pulse 1s infinite' }}>🌙</div>
        <h2>Connexion au serveur...</h2>
      </div>
    )
  }

  const isSpectator = state.myRole === '?' && state.phase !== 'LOBBY'

  return (
    <div className="app">
      <Header state={state} send={send} />
      {isSpectator && (
        <div style={{
          background: 'var(--surface)',
          borderBottom: '2px solid var(--gold)',
          padding: '8px 16px',
          textAlign: 'center',
          fontSize: '0.875rem',
          color: 'var(--gold)',
        }}>
          👁 Mode spectateur — vous observez la partie sans y participer
        </div>
      )}
      <main className="main">
        <div className="panel">
          {state.phase === 'LOBBY' && <Lobby state={state} send={send} />}
          {(state.phase === 'NIGHT' || state.phase === 'WITCH') && (
            <NightPhase state={state} send={send} />
          )}
          {state.phase === 'VOTE' && <VotePhase state={state} send={send} />}
          {state.phase === 'END' && <EndPhase state={state} send={send} />}
        </div>
      </main>
    </div>
  )
}
