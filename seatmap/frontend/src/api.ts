// src/api.js

const API_BASE_URL = "http://localhost:8080"; // adjust as needed

export async function getSeatsByEvent(eventID) {
  console.log("getSeatsByEvent", eventID);
  const res = await fetch(`${API_BASE_URL}/events/${eventID}/seats`);
  if (!res.ok) {
    throw new Error("Failed to fetch seats");
  }
  const seats = await res.json();
  // sort seats by seatRow and seatNumber
  seats.sort((a, b) => a.row - b.row || a.number - b.number);
  console.log("getSeatsByEvent", seats);
  return seats;
}

export async function reserveSeat(seatID, duration = 300) {
  console.log("reserveSeat", seatID, duration);
  const res = await fetch(`${API_BASE_URL}/seats/${seatID}/reserve?duration=${duration}`, {
    method: "POST",
  });
  if (!res.ok) {
    throw new Error(await res.text());
  }
  return res.text();
}

export async function bookSeat(seatID) {
  console.log("bookSeat", seatID);
  const res = await fetch(`${API_BASE_URL}/seats/${seatID}/book`, {
    method: "POST",
  });
  if (!res.ok) {
    throw new Error(await res.text());
  }
  return res.text();
}
