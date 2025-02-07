// src/components/Seat.js
import React from "react";

function Seat({ seat, onReserve, onBook }) {
  // Choose color/style based on seat status
  let bgColor = "#fff";
  if (seat.status === "available") bgColor = "green";
  if (seat.status === "reserved") bgColor = "orange";
  if (seat.status === "booked") bgColor = "red";

  return (
    <div
      style={{
        width: "40px",
        height: "40px",
        margin: "5px",
        backgroundColor: bgColor,
        display: "flex",
        justifyContent: "center",
        alignItems: "center",
        cursor: seat.status === "available" ? "pointer" : "default",
        border: "1px solid #000",
      }}
    >
      <div style={{ fontSize: "0.8rem", textAlign: "center" }}>
        {seat.seatRow}-{seat.seatNumber}
      </div>
      {seat.status === "available" && (
        <button 
          style={{position: "absolute", opacity: 0}} 
          onClick={() => console.log("Reserve seat", seat.id) as any || onReserve(seat.id)}
          aria-label="Reserve seat"
        />
      )}
      {seat.status === "reserved" && (
        <button 
          style={{position: "absolute", opacity: 0}}
          onClick={() => console.log("Book seat", seat.id) as any || onBook(seat.id)}
          aria-label="Book seat"
        />
      )}
    </div>
  );
}

export default Seat;
