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

  return (
    <div className="app">
      <Header state={state} send={send} />
      <main className="main">
        <div className="panel">
          {state.phase === 'LOBBY' && <Lobby state={state} send={send} />}
          {(state.phase === 'NIGHT' || state.phase === 'WITCH') && (
            <NightPhase state={state} send={send} />
          )}
          {state.phase === 'VOTE' && <VotePhase state={state} send={send} />}
          {state.phase === 'END' && <EndPhase state={state} />}
        </div>
      </main>
    </div>
  )
}
