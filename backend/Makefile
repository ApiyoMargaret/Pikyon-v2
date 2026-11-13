.PHONY: sync-api generate-swagger generate-frontend-types

# sync-api regenerates swagger.json from Go comments and updates frontend types
sync-api: generate-swagger generate-frontend-types

# generate-swagger runs swag to generate swagger.json from code comments
generate-swagger:
	@echo "🔍 Generating swagger documentation from Go comments..."
	@which swag > /dev/null 2>&1 || export PATH=$$PATH:$$HOME/go/bin && go install github.com/swaggo/swag/cmd/swag@latest
	swag init -g cmd/api/main.go --output docs/swagger --parseDependency
	@echo "✅ swagger.json generated at docs/swagger/swagger.json"

# generate-frontend-types generates TypeScript types from swagger.json
generate-frontend-types:
	@echo "🔧 Generating frontend API types from swagger.json..."
	python3 generate_types.py
	@echo "✅ Frontend API types generated at frontend/src/api/types.py"