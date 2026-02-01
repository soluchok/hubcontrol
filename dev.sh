#!/bin/bash

# USB Hub Control - Development Runner
# This script runs both the backend and frontend in development mode

cleanup() {
    echo "Shutting down..."
    kill $BACKEND_PID 2>/dev/null
    kill $FRONTEND_PID 2>/dev/null
    exit 0
}

trap cleanup SIGINT SIGTERM

echo "Starting USB Hub Control..."
echo ""

# Start backend
cd backend
go run . &
BACKEND_PID=$!
cd ..

# Wait for backend to start
sleep 2

# Start frontend dev server
cd frontend
npm run dev &
FRONTEND_PID=$!
cd ..

echo ""
echo "========================================="
echo "  USB Hub Control is running!"
echo "========================================="
echo ""
echo "  Frontend (dev):  http://localhost:5173"
echo "  Backend API:     http://localhost:8080"
echo ""
echo "  Press Ctrl+C to stop both servers"
echo "========================================="
echo ""

# Wait for either process to exit
wait
