.PHONY: all dev backend frontend redis clean

# Default command: runs both servers concurrently (assumes Redis is already handled or not needed)
all:
	@make -j 2 backend frontend

# The ultimate dev command: ensures Redis is running, then starts backend and frontend
dev: redis
	@echo "Starting Habit Coach ecosystem..."
	@make -j 2 backend frontend

# Starts Redis Stack via Docker. Handles creation if it doesn't exist, or starts it if stopped.
redis:
	@echo "Checking Redis Stack..."
	@if [ ! "$$(docker ps -a -q -f name=redis-stack)" ]; then \
		echo "Creating and starting new Redis Stack container..."; \
		docker run -d --name redis-stack -p 6379:6379 -p 8001:8001 redis/redis-stack:latest; \
	elif [ ! "$$(docker ps -q -f name=redis-stack)" ]; then \
		echo "Starting existing Redis Stack container..."; \
		docker start redis-stack; \
	else \
		echo "Redis Stack is already running."; \
	fi

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
