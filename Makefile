.PHONY: all backend frontend clean

# Default command: runs both servers concurrently
all:
	@make -j 2 backend frontend

# Run the Go backend
backend:
	@echo "Starting Go Backend on http://localhost:8080..."
	@cd backend && go run .

# Run the React frontend
frontend:
	@echo "Starting React Frontend on http://localhost:5173..."
	@cd frontend && npm run dev

# Clean compiled binaries
clean:
	@echo "Cleaning up build artifacts..."
	@rm -f backend/server
