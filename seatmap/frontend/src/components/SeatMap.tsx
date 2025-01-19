// src/components/SeatMap.js
import React, { useEffect, useState } from "react";
import { getSeatsByEvent, reserveSeat, bookSeat } from "../api.ts";
import Seat from "./Seat.tsx";

function SeatMap({ eventID }) {
  const [seats, setSeats] = useState([]);
  const [error, setError] = useState(null);

  useEffect(() => {
    loadSeats();
  }, [eventID]);

  async function loadSeats() {
    try {
      setError(null);
      const data = await getSeatsByEvent(eventID);
      setSeats(data);
    } catch (err) {
      setError(err.message);
    }
  }

  async function handleReserve(seatID) {
    try {
      await reserveSeat(seatID);
      // reload seat states from server
      await loadSeats();
    } catch (err) {
      setError(err.message);
    }
  }

  async function handleBook(seatID) {
    try {
      await bookSeat(seatID);
      // reload seat states from server
      await loadSeats();
    } catch (err) {
      setError(err.message);
    }
  }

  // Render seats in a simple grid
  return (
    <div>
      {error && <div style={{ color: "red" }}>{error}</div>}

      <h2>Event ID: {eventID}</h2>
      <div style={{ display: "flex", flexWrap: "wrap", width: "400px" }}>
        {seats.map((seat) => (
          <Seat
            key={seat.id}
            seat={seat}
            onReserve={() => handleReserve(seat.id)}
            onBook={() => handleBook(seat.id)}
          />
        ))}
      </div>
    </div>
  );
}

export default SeatMap;
