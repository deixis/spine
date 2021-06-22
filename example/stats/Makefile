start:
		NODE_NAME=node-a CONFIG_URI=file://${PWD}/config.toml go run main.go

metrics:
		@curl "http://127.0.0.1:2112/metrics"
		@echo ""

test:
		@curl "http://127.0.0.1:3003/test"
		@echo ""