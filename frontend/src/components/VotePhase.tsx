import { useState } from 'react'
import type { GameState } from '../types'

interface Props {
  state: GameState
  send: (action: string, extra?: Record<string, string>) => void
}

export function VotePhase({ state, send }: Props) {
  const [selected, setSelected] = useState('')

  const myVote = state.votes[state.myId]
  const alreadyVoted = myVote !== undefined
  const canVote = state.myAlive && !alreadyVoted

  const alivePlayers = Object.values(state.players).filter(p => p.alive && p.id !== state.myId)
  const allPlayers = Object.values(state.players)

  // Compte des votes pour chaque joueur
  const tally: Record<string, number> = {}
  for (const target of Object.values(state.votes)) {
    tally[target] = (tally[target] ?? 0) + 1
  }

  function handleVote() {
    if (selected && canVote) {
      send('vote', { target: selected })
    }
  }

  return (
    <>
      {!state.myAlive && (
        <div className="dead-banner">
          💀 Vous êtes mort — vous observez le vote du village.
        </div>
      )}

      <div className="phase-banner">
        <h1>⚖️ Vote du village</h1>
        <p>Qui est le loup-garou parmi vous ?</p>
      </div>

      {state.myAlive && (
        <div className="card">
          <h2>Voter</h2>
          {alreadyVoted ? (
            <p style={{ color: 'var(--text-muted)', marginBottom: '12px', fontSize: '0.875rem' }}>
              ✓ Vous avez voté pour{' '}
              <strong style={{ color: 'var(--gold)' }}>{myVote}</strong>.
            </p>
          ) : (
            <p style={{ color: 'var(--text-muted)', marginBottom: '12px', fontSize: '0.875rem' }}>
              Sélectionnez le joueur que vous suspectez d'être un loup.
            </p>
          )}

          <div className="players-grid">
            {alivePlayers.map(p => (
              <div
                key={p.id}
                className={`player-card selectable${selected === p.id && !alreadyVoted ? ' selected' : ''}`}
                onClick={() => canVote && setSelected(p.id)}
              >
                <div className="player-id">{p.id}</div>
                {tally[p.id] && (
                  <div style={{ fontSize: '0.75rem', color: 'var(--gold)', marginTop: '4px' }}>
                    {tally[p.id]} vote{(tally[p.id] ?? 0) > 1 ? 's' : ''}
                  </div>
                )}
              </div>
            ))}
          </div>

          <div className="btn-row">
            <button
              className="btn btn-danger btn-lg"
              disabled={!selected || !canVote}
              onClick={handleVote}
            >
              {alreadyVoted ? '✓ Vote envoyé' : `Voter pour ${selected || '…'}`}
            </button>
          </div>
        </div>
      )}

      <div className="card">
        <h2>Votes ({Object.keys(state.votes).length}/{allPlayers.filter(p => p.alive).length})</h2>
        {Object.keys(state.votes).length === 0 ? (
          <p style={{ color: 'var(--text-muted)', fontSize: '0.875rem' }}>
            En attente des premiers votes…
          </p>
        ) : (
          <ul className="vote-list">
            {Object.entries(state.votes).map(([voter, target]) => (
              <li key={voter} className="vote-item">
                <strong>{voter}</strong>
                <span className="arrow">→</span>
                <strong style={{ color: 'var(--wolf)' }}>{target}</strong>
              </li>
            ))}
          </ul>
        )}
      </div>

      {!state.myAlive && Object.keys(state.votes).length > 0 && (
        <div className="card">
          <h2>Décompte</h2>
          <ul className="vote-list">
            {Object.entries(tally)
              .sort(([, a], [, b]) => b - a)
              .map(([id, count]) => (
                <li key={id} className="vote-item">
                  <strong>{id}</strong>
                  <span className="arrow">—</span>
                  <span>{count} vote{count > 1 ? 's' : ''}</span>
                </li>
              ))}
          </ul>
        </div>
      )}
    </>
  )
}
