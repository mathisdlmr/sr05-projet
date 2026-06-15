import { useEffect, useReducer, useRef } from 'react'
import type { GameState, ServerEvent } from './types'

const initial: GameState = {
  wsStatus: 'connecting',
  phase: 'LOBBY',
  myId: '',
  myRole: '?',
  myAlive: true,
  players: {},
  votes: {},
  wolfVotes: {},
  killWolf: '',
  winner: '',
  joined: false,
  lastSnapshot: null,
  snapshotRejection: null,
  lastNotice: null,
}

function roleLabel(role: string): string {
  switch (role) {
    case 'WOLF':     return 'Loup-Garou'
    case 'WITCH':    return 'Sorcière'
    case 'VILLAGER': return 'Villageois'
    default:         return role
  }
}

type Action = ServerEvent | { type: 'wsOpen' } | { type: 'wsClose' }

function reducer(state: GameState, action: Action): GameState {
  switch (action.type) {
    case 'wsOpen':
      return { ...state, wsStatus: 'connected' }

    case 'wsClose':
      return { ...state, wsStatus: 'disconnected' }

    case 'init': {
      const isNightWolf = action.phase === 'NIGHT' && action.myRole === 'WOLF'
      return {
        ...state,
        wsStatus: 'connected',
        phase: action.phase,
        myId: action.myId,
        myRole: action.myRole,
        myAlive: action.myAlive,
        players: action.players,
        votes: action.phase === 'VOTE' ? action.votes : {},
        wolfVotes: isNightWolf ? action.votes : {},
        killWolf: action.killWolf ?? '',
        winner: '',
        joined: action.myId in action.players,
      }
    }

    case 'playerJoined':
      if (action.playerId in state.players) return state
      return {
        ...state,
        players: {
          ...state.players,
          [action.playerId]: { id: action.playerId, role: '?', alive: true },
        },
      }

    case 'playerLeft': {
      const players = { ...state.players }
      let message = `${action.playerId} a quitté la partie.`
      if (action.role) {
        if (players[action.playerId]) {
          players[action.playerId] = { ...players[action.playerId], alive: false, role: action.role }
        }
        message = `${action.playerId} a quitté la partie — il/elle était ${roleLabel(action.role)}.`
      } else {
        delete players[action.playerId]
        message = `${action.playerId} a quitté le lobby.`
      }
      return { ...state, players, lastNotice: { id: Date.now(), message } }
    }

    case 'actionRejected':
      return { ...state, lastNotice: { id: Date.now(), message: action.message } }

    case 'gameRestart':
      return {
        ...initial,
        wsStatus: state.wsStatus,
        myId: state.myId,
      }

    case 'gameStart':
      return {
        ...state,
        phase: 'NIGHT',
        myRole: action.myRole,
        players: action.players,
        votes: {},
        wolfVotes: {},
        joined: true,
      }

    case 'wolfVoted': {
      if (action.target === undefined) return state
      return {
        ...state,
        wolfVotes: { ...state.wolfVotes, [action.voter]: action.target },
      }
    }

    case 'phaseChange':
      return {
        ...state,
        phase: action.phase,
        killWolf: action.killWolf ?? '',
        votes: {},
        wolfVotes: action.phase === 'NIGHT' ? {} : state.wolfVotes,
      }

    case 'nightKills': {
      const players = { ...state.players }
      for (const id of action.killed) {
        if (players[id]) players[id] = { ...players[id], alive: false }
      }
      return {
        ...state,
        players,
        myAlive: players[state.myId]?.alive ?? state.myAlive,
        phase: action.nextPhase === 'VOTE' ? 'VOTE' : state.phase,
        killWolf: '',
        votes: {},
        wolfVotes: {},
      }
    }

    case 'voted':
      return {
        ...state,
        votes: { ...state.votes, [action.voter]: action.target },
      }

    case 'voteEliminated': {
      const players = { ...state.players }
      if (players[action.playerId]) {
        players[action.playerId] = { ...players[action.playerId], alive: false }
      }
      return {
        ...state,
        players,
        myAlive: players[state.myId]?.alive ?? state.myAlive,
        votes: {},
        wolfVotes: {},
        phase: action.nextPhase === 'NIGHT' ? 'NIGHT' : state.phase,
      }
    }

    case 'gameEnd':
      return {
        ...state,
        phase: 'END',
        winner: action.winner,
        players: action.players,
      }

    case 'snapshot_received':
      return {
        ...state,
        lastSnapshot: action.eg,
        snapshotRejection: null,
      }

    case 'snapshot_rejected':
      return {
        ...state,
        snapshotRejection: action.reason || 'snapshot refusé',
      }

    default:
      return state
  }
}

export function useGame(): [GameState, (action: string, extra?: Record<string, string>) => void] {
  const [state, dispatch] = useReducer(reducer, initial)
  const wsRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    const wsProtocol = location.protocol === 'https:' ? 'wss' : 'ws'
    const ws = new WebSocket(`${wsProtocol}://${location.host}/ws`)
    wsRef.current = ws

    ws.onopen = () => dispatch({ type: 'wsOpen' })
    ws.onclose = () => dispatch({ type: 'wsClose' })

    const handleUnload = () => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ action: 'quit' }))
      }
    }
    window.addEventListener('beforeunload', handleUnload)
    ws.onmessage = (e: MessageEvent<string>) => {
      try {
        dispatch(JSON.parse(e.data) as Action)
      } catch {
        console.error('Failed to parse server event:', e.data)
      }
    }

    return () => {
      window.removeEventListener('beforeunload', handleUnload)
      ws.close()
    }
  }, [])

  const send = (action: string, extra: Record<string, string> = {}) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ action, ...extra }))
    }
  }

  return [state, send]
}
