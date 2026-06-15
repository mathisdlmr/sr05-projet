import type { GameState } from '../types'

interface Props {
  state: GameState
  send: (action: string, extra?: Record<string, string>) => void
}

export function Lobby({ state, send }: Props) {
  const playerList = Object.values(state.players)
  const nbJoined = playerList.length
  const notEnoughPlayers = nbJoined < 3

  return (
    <>
      <div className="phase-banner">
        <h1>Salle d'attente</h1>
        <p>En attente que tous les joueureuses rejoignent la partie.</p>
      </div>

      <div className="card">
        <h2>joueureuses connecté.e.s</h2>
        <p className="player-count">{nbJoined} joueur{nbJoined !== 1 ? 's' : ''} dans le lobby</p>
        {playerList.length === 0 ? (
          <p style={{ color: 'var(--text-muted)', fontSize: '0.875rem' }}>
            Personne n'a encore rejoint…
          </p>
        ) : (
          <div className="players-grid">
            {playerList.map(p => (
              <div
                key={p.id}
                className={`player-card${p.id === state.myId ? ' me' : ''}`}
              >
                <div className="player-id">{p.id}</div>
                {p.id === state.myId && (
                  <div style={{ fontSize: '0.75rem', color: 'var(--gold)' }}>vous</div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="card">
        <h2>Démarrer</h2>
        <p style={{ color: 'var(--text-muted)', marginBottom: '16px', fontSize: '0.875rem' }}>
          La partie démarre quand tous les joueureuses attendus sont présent.e.s.
        </p>
        {notEnoughPlayers && (
          <p style={{ color: 'var(--gold)', marginBottom: '16px', fontSize: '0.875rem' }}>
            Il faut au moins 3 joueureuses pour démarrer ({nbJoined}/3).
            Faites rejoindre un nouveau site avec :{' '}
            <code>./scripts/join_site.sh &lt;id_dun_site_pas_utilise&gt; &lt;id_dun_site_en_partie&gt;</code>
          </p>
        )}
        <button
          className="btn btn-success btn-lg"
          disabled={notEnoughPlayers}
          onClick={() => send('start')}
        >
          Démarrer la partie
        </button>
      </div>
    </>
  )
}
