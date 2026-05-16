# Architecture & System Design

## Overview
The Wisdom Portal is a web-based interface for interacting with the Wisdom neural atlas and expert registry.

## Tech Stack
- **Frontend:** React (Vite)
- **Styling:** Tailwind CSS
- **Icons:** Lucide React
- **State Management:** React Context API (WisdomContext)
- **Communication:** WebSockets for real-time events, REST for identity and data.

## Key Components
- **WisdomProvider:** Manages authentication, user state, and WebSocket connectivity.
- **MissionControlView:** Main dashboard for system oversight.
- **GraphView:** Visual representation of knowledge connections.
- **ExpertRegistry:** Management of domain-specific experts.
