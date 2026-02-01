.PHONY: all backend frontend dev clean

all: backend frontend

# Build the Go backend
backend:
	cd backend && go build -o ../bin/hubcontrol .

# Build the React frontend
frontend:
	cd frontend && npm run build

# Run both in development mode
dev:
	@echo "Starting backend and frontend..."
	@echo "Backend: http://localhost:8080"
	@echo "Frontend: http://localhost:5173"
	@echo ""
	@echo "Run these commands in separate terminals:"
	@echo "  Terminal 1: cd backend && go run ."
	@echo "  Terminal 2: cd frontend && npm run dev"

# Run backend only
run-backend:
	cd backend && go run .

# Run frontend dev server
run-frontend:
	cd frontend && npm run dev

# Install dependencies
deps:
	cd backend && go mod tidy
	cd frontend && npm install

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf frontend/dist/

# Build and run production
prod: all
	./bin/hubcontrol
