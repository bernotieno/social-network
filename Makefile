run:
	cd backend && go run .
.PHONY: frontend
frontend:
	cd frontend && npm run dev
format:
	cd backend && gofmt -w -s .
	@echo "files are formatted correctly"
