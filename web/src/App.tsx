import { useState } from 'react'
import { Leaderboard } from './components/Leaderboard'
import { ConnectionStatus } from './components/ConnectionStatus'
import { Trophy, Zap } from 'lucide-react'
import './App.css'

function App() {
  const [leaderboardId] = useState('global_scores')

  return (
    <div className="app">
      <div className="background-effects">
        <div className="gradient-orb orb-1" />
        <div className="gradient-orb orb-2" />
        <div className="gradient-orb orb-3" />
        <div className="grid-overlay" />
      </div>

      <header className="header">
        <div className="logo">
          <Trophy className="logo-icon" />
          <h1>LEADERBOARD</h1>
          <Zap className="logo-accent" />
        </div>
        <p className="subtitle">Real-time rankings powered by Nakama</p>
        <ConnectionStatus />
      </header>

      <main className="main-content">
        <Leaderboard leaderboardId={leaderboardId} />
      </main>

      <footer className="footer">
        <p>Scores ingested via Kafka • Real-time updates via WebSocket</p>
      </footer>
    </div>
  )
}

export default App
