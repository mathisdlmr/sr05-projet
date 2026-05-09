import type { GameState } from '../types'

interface Props {
  state: GameState
}

function roleLabel(role: string): { label: string; emoji: string } {
  switch (role) {
    case 'WOLF':     return { label: 'Loup', emoji: '🐺' }
    case 'WITCH':    return { label: 'Sorcière', emoji: '🧙' }
    case 'VILLAGER': return { label: 'Villageois', emoji: '🏡' }
    default:         return { label: '?', emoji: '❓' }
  }
}

export function EndPhase({ state }: Props) {
  const isWolvesWin = state.winner === 'WOLVES'
  const players = Object.values(state.players)

  const wolves    = players.filter(p => p.role === 'WOLF')
  const villagers = players.filter(p => p.role === 'VILLAGER')
  const witches   = players.filter(p => p.role === 'WITCH')

  return (
    <>
      <div className={`winner-banner ${state.winner}`}>
        <div style={{ fontSize: '3rem', marginBottom: '12px' }}>
          {isWolvesWin ? '🐺' : '🏡'}
        </div>
        <h1 style={{ color: isWolvesWin ? 'var(--wolf)' : 'var(--villager)' }}>
          {isWolvesWin ? 'Les loups ont gagné !' : 'Le village a gagné !'}
        </h1>
        <p style={{ color: 'var(--text-muted)', marginTop: '8px' }}>
          {isWolvesWin
            ? 'Les loups-garous ont dévoré le village.'
            : 'Le village a éliminé tous les loups-garous.'}
        </p>
      </div>

      <div className="card">
        <h2>Révélation des rôles</h2>
        <div className="players-grid" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(140px, 1fr))' }}>
          {players.map(p => {
            const { label, emoji } = roleLabel(p.role)
            const isMe = p.id === state.myId
            return (
              <div
                key={p.id}
                className={`player-card${!p.alive ? ' dead' : ''}${isMe ? ' me' : ''}`}
              >
                <div style={{ fontSize: '1.5rem', marginBottom: '6px' }}>{emoji}</div>
                <div className="player-id">{p.id}</div>
                <div>
                  <span className={`role-badge ${p.role}`}>{label}</span>
                </div>
                <div className={`player-status ${p.alive ? 'alive' : 'dead'}`} style={{ marginTop: '6px' }}>
                  {p.alive ? '● Vivant' : '✕ Mort'}
                </div>
                {isMe && (
                  <div style={{ fontSize: '0.7rem', color: 'var(--gold)', marginTop: '4px' }}>vous</div>
                )}
              </div>
            )
          })}
        </div>
      </div>

      <div className="card">
        <h2>Équipes</h2>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          <div>
            <span style={{ color: 'var(--wolf)', fontWeight: 600 }}>🐺 Loups : </span>
            {wolves.map(p => p.id).join(', ') || '—'}
          </div>
          {witches.length > 0 && (
            <div>
              <span style={{ color: 'var(--witch)', fontWeight: 600 }}>🧙 Sorcière : </span>
              {witches.map(p => p.id).join(', ')}
            </div>
          )}
          <div>
            <span style={{ color: 'var(--villager)', fontWeight: 600 }}>🏡 Villageois : </span>
            {villagers.map(p => p.id).join(', ') || '—'}
          </div>
        </div>
      </div>

      <div className="btn-row" style={{ justifyContent: 'center', marginTop: '8px' }}>
        <button
          className="btn btn-primary btn-lg"
          onClick={() => window.location.reload()}
        >
          🔄 Rejouer
        </button>
      </div>
    </>
  )
}
