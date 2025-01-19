// src/api.js

const API_BASE_URL = "http://localhost:8080"; // adjust as needed

export async function getSeatsByEvent(eventID) {
  const res = await fetch(`${API_BASE_URL}/events/${eventID}/seats`);
  if (!res.ok) {
    throw new Error("Failed to fetch seats");
  }
  return res.json();
}

export async function reserveSeat(seatID, duration = 300) {
  const res = await fetch(`${API_BASE_URL}/seats/${seatID}/reserve?duration=${duration}`, {
    method: "POST",
  });
  if (!res.ok) {
    throw new Error(await res.text());
  }
  return res.text();
}

export async function bookSeat(seatID) {
  const res = await fetch(`${API_BASE_URL}/seats/${seatID}/book`, {
    method: "POST",
  });
  if (!res.ok) {
    throw new Error(await res.text());
  }
  return res.text();
}
