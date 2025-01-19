// src/App.js
import React from "react";
import SeatMap from "./components/SeatMap.tsx";

function App() {
  return (
    <div className="App">
      <h1>Ticketmaster-like Seat Map Demo</h1>
      <SeatMap eventID={1} />
    </div>
  );
}

export default App;