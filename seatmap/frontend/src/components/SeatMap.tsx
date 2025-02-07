// src/SeatMap.js
import React, { useEffect, useState } from "react";
import { getSeatsByEvent, reserveSeat, bookSeat } from "../api.ts";
import Seat from "./Seat.tsx";

function SeatMap({ eventId }) {
  console.log("SeatMap", eventId);
  const [seats, setSeats] = useState([]);
  const [error, setError] = useState("");

  // 1. Load seats initially
  useEffect(() => {
    fetchSeats();
    // eslint-disable-next-line
  }, [eventId]);

  async function fetchSeats() {
    try {
      setError("");
      const data = await getSeatsByEvent(eventId);
      setSeats(data);
    } catch (err) {
      setError(err.message);
    }
  }

  // 2. Subscribe to SSE for real-time seat updates
  useEffect(() => {
    const sseUrl = `http://localhost:8080/events/${eventId}/seats/stream`;
    const eventSource = new EventSource(sseUrl);

    // Whenever the server pushes a new seat map
    eventSource.onmessage = (event) => {
      try {
        console.log("SSE message", event.data);
        const updatedSeats = JSON.parse(event.data);
        console.log("updatedSeats", updatedSeats);
        // sort seats by seatRow and seatNumber
        updatedSeats.sort((a, b) => a.row - b.row || a.number - b.number);
        setSeats(updatedSeats);
      } catch (err) {
        console.error("Failed to parse SSE data:", err);
      }
    };

    // For error handling
    eventSource.onerror = (err) => {
      console.error("SSE error:", err);
      // optionally setError("SSE connection failed");
    };

    // Cleanup when component unmounts
    return () => {
      eventSource.close();
    };
  }, [eventId]);

  // 3. Reserve a seat
  async function handleReserve(seatID) {
    console.log("handleReserve", seatID);
    try {
      setError("");
      await reserveSeat(seatID);
      // *No need to manually refresh seats here*
      // SSE broadcast from the server will update us
    } catch (err) {
      setError(err.message);
    }
  }

  // 4. Book a seat
  async function handleBook(seatID) {
    try {
      setError("");
      await bookSeat(seatID);
      // *No need to manually refresh seats here*
      // SSE broadcast from the server will update us
    } catch (err) {
      setError(err.message);
    }
  }

  return (
    <div>
      <h2>Seat Map (Event {eventId})</h2>
      {error && <p style={{ color: "red" }}>{error}</p>}

      <div style={{ display: "flex", flexWrap: "wrap", maxWidth: "400px" }}>
        {seats.map((seat) => (
          <Seat
            key={seat.id}
            seat={seat}
            onReserve={handleReserve}
            onBook={handleBook}
          />
        ))}
      </div>
    </div>
  );
}

export default SeatMap;
