import { useState } from 'react'
import type { GameState, Player } from '../types'

interface Props {
  state: GameState
  send: (action: string, extra?: Record<string, string>) => void
}

// ── Vue loup : choisir une victime ────────────────────────────────────────────

function WolfView({ state, send }: Props) {
  const [selected, setSelected] = useState('')

  const myVote = state.wolfVotes[state.myId]
  const alreadyVoted = myVote !== undefined

  const alivePlayers = Object.values(state.players)
  const targets = alivePlayers.filter(
    p => p.alive && p.id !== state.myId && p.role !== 'WOLF'
  )
  const wolves = alivePlayers.filter(p => p.role === 'WOLF')
  const wolfVoters = wolves.filter(p => p.id in state.wolfVotes)

  function handleVote() {
    if (selected && !alreadyVoted) {
      send('wolfkill', { target: selected })
    }
  }

  return (
    <>
      {!state.myAlive && (
        <div className="dead-banner">💀 Vous êtes mort — vous observez silencieusement.</div>
      )}

      <div className="phase-banner">
        <h1>🌙 La nuit tombe</h1>
        <p>Les loups-garous se réveillent et choisissent une victime.</p>
      </div>

      {state.myAlive && (
        <div className="card">
          <h2>Choisir une victime</h2>
          {alreadyVoted ? (
            <p style={{ color: 'var(--text-muted)', marginBottom: '12px', fontSize: '0.875rem' }}>
              ✓ Vous avez voté pour <strong style={{ color: 'var(--wolf)' }}>{myVote}</strong>.
            </p>
          ) : (
            <p style={{ color: 'var(--text-muted)', marginBottom: '12px', fontSize: '0.875rem' }}>
              Sélectionnez votre victime.
            </p>
          )}

          <div className="players-grid">
            {targets.map(p => (
              <div
                key={p.id}
                className={`player-card selectable${selected === p.id && !alreadyVoted ? ' selected' : ''}${!p.alive ? ' dead' : ''}`}
                onClick={() => !alreadyVoted && p.alive && setSelected(p.id)}
              >
                <div className="player-id">{p.id}</div>
                <div className={`player-status ${p.alive ? 'alive' : 'dead'}`}>
                  {p.alive ? '● Vivant' : '✕ Mort'}
                </div>
              </div>
            ))}
          </div>

          <div className="btn-row">
            <button
              className="btn btn-danger btn-lg"
              disabled={!selected || alreadyVoted}
              onClick={handleVote}
            >
              {alreadyVoted ? '✓ Vote envoyé' : `Voter pour ${selected || '…'}`}
            </button>
          </div>
        </div>
      )}

      <div className="card">
        <h2>Votes des loups</h2>
        <div className="wolf-votes">
          {wolves.map(w => (
            <span
              key={w.id}
              className={`wolf-vote-chip${w.id in state.wolfVotes ? ' voted' : ''}`}
            >
              {w.id}
              {w.id in state.wolfVotes
                ? ` → ${state.wolfVotes[w.id]}`
                : ' (en attente…)'}
            </span>
          ))}
        </div>
        <p style={{ marginTop: '10px', fontSize: '0.8rem', color: 'var(--text-muted)' }}>
          {wolfVoters.length}/{wolves.length} loup{wolves.length > 1 ? 's' : ''} ont voté
        </p>
      </div>
    </>
  )
}

// ── Vue sorcière : sauver / empoisonner / passer ──────────────────────────────

function WitchView({ state, send }: Props) {
  const [poisonTarget, setPoisonTarget] = useState('')

  const alivePlayers = Object.values(state.players).filter(p => p.alive && p.id !== state.myId)

  return (
    <>
      <div className="phase-banner">
        <h1>🧙 La sorcière se réveille</h1>
        <p>Vous pouvez utiliser vos potions cette nuit.</p>
      </div>

      <div className="card">
        <h2>Cette nuit</h2>
        {state.killWolf ? (
          <p style={{ fontSize: '1rem', marginBottom: '6px' }}>
            Les loups ont désigné{' '}
            <strong style={{ color: 'var(--wolf)' }}>{state.killWolf}</strong> comme victime.
          </p>
        ) : (
          <p style={{ color: 'var(--text-muted)' }}>Personne n'a été désigné par les loups.</p>
        )}
      </div>

      {state.killWolf && (
        <div className="card">
          <h2>Potion de vie</h2>
          <button
            className="btn btn-success btn-lg"
            onClick={() => send('witchsave')}
          >
            💚 Sauver {state.killWolf}
          </button>
        </div>
      )}

      <div className="card">
        <h2>Potion de mort</h2>
        <p style={{ color: 'var(--text-muted)', marginBottom: '12px', fontSize: '0.875rem' }}>
          Empoisonnez un joueur de votre choix.
        </p>
        <div className="players-grid">
          {alivePlayers.map((p: Player) => (
            <div
              key={p.id}
              className={`player-card selectable${poisonTarget === p.id ? ' selected' : ''}`}
              onClick={() => setPoisonTarget(p.id)}
            >
              <div className="player-id">{p.id}</div>
              <div className="player-status alive">● Vivant</div>
            </div>
          ))}
        </div>
        <div className="btn-row">
          <button
            className="btn btn-danger btn-lg"
            disabled={!poisonTarget}
            onClick={() => send('witchkill', { target: poisonTarget })}
          >
            ☠️ Empoisonner {poisonTarget || '…'}
          </button>
        </div>
      </div>

      <div className="divider">ou</div>

      <button
        className="btn btn-neutral btn-lg"
        style={{ width: '100%' }}
        onClick={() => send('witchskip')}
      >
        Passer (ne rien faire)
      </button>
    </>
  )
}

// ── Vue d'attente (non-loup, non-sorcière) ────────────────────────────────────

function WaitingView({ phase }: { phase: string }) {
  return (
    <div className="waiting">
      <div className="waiting-icon">
        {phase === 'WITCH' ? '🧙' : '🌙'}
      </div>
      <h2>{phase === 'WITCH' ? 'La sorcière décide…' : 'La nuit tombe…'}</h2>
      <p>
        {phase === 'WITCH'
          ? 'Attendez que la sorcière ait pris sa décision.'
          : 'Fermez les yeux et attendez le lever du jour.'}
      </p>
    </div>
  )
}

// ── Composant principal NIGHT / WITCH ─────────────────────────────────────────

export function NightPhase({ state, send }: Props) {
  const { phase, myRole, myAlive } = state

  // La sorcière en phase WITCH
  if (phase === 'WITCH' && myRole === 'WITCH' && myAlive) {
    return <WitchView state={state} send={send} />
  }

  // Un loup vivant en phase NUIT
  if (phase === 'NIGHT' && myRole === 'WOLF') {
    return <WolfView state={state} send={send} />
  }

  // Tout le monde : vue d'attente (avec bannière mort si applicable)
  return (
    <>
      {!myAlive && (
        <div className="dead-banner">💀 Vous êtes mort — vous observez la nuit en silence.</div>
      )}
      <WaitingView phase={phase} />
    </>
  )
}
