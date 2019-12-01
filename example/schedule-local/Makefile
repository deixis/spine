start:
		CONFIG_URI=file://${PWD}/config.toml go run main.go

schedule:
		@curl -X "POST" "http://localhost:3000/job/foo" \
				-H 'Content-Type: application/json; charset=utf-8' \
				-d '{"In": 10, "Data": "Random data"}'
		@echo ""

schedule-err:
		@curl -X "POST" "http://localhost:3000/job/err" \
				-H 'Content-Type: application/json; charset=utf-8' \
				-d '{"In": 10, "Data": "Scheduled error"}'
		@echo ""

schedule-panic:
		@curl -X "POST" "http://localhost:3000/job/panic" \
				-H 'Content-Type: application/json; charset=utf-8' \
				-d '{"In": 10, "Data": "Panic"}'
		@echo ""
