export type Phase = 'LOBBY' | 'NIGHT' | 'WITCH' | 'VOTE' | 'END'
export type Role = 'WOLF' | 'VILLAGER' | 'WITCH' | '?'

export interface Player {
  id: string
  role: Role
  alive: boolean
}

export interface GameState {
  wsStatus: 'connecting' | 'connected' | 'disconnected'
  phase: Phase
  myId: string
  myRole: Role
  myAlive: boolean
  players: Record<string, Player>
  votes: Record<string, string>        // village votes: voter -> target
  wolfVotes: Record<string, string>    // wolf votes: voter -> target (wolves only)
  killWolf: string                     // for witch in WITCH phase
  winner: string
  joined: boolean
  // myRole is always set: '?' means role not yet revealed (LOBBY)
}

// ── Événements serveur -> navigateur ──────────────────────────────────────────

export type ServerEvent =
  | { type: 'init'; phase: Phase; myId: string; myRole: Role; myAlive: boolean; players: Record<string, Player>; votes: Record<string, string>; killWolf?: string }
  | { type: 'playerJoined'; playerId: string }
  | { type: 'gameStart'; myRole: Role; players: Record<string, Player> }
  | { type: 'wolfVoted'; voter: string; target?: string }
  | { type: 'phaseChange'; phase: Phase; killWolf?: string }
  | { type: 'nightKills'; killed: string[]; nextPhase: 'VOTE' | 'END' }
  | { type: 'voted'; voter: string; target: string }
  | { type: 'voteEliminated'; playerId: string; nextPhase: 'NIGHT' | 'END' }
  | { type: 'gameEnd'; winner: string; players: Record<string, Player> }
