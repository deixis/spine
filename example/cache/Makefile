start:
		@make -j7 start1 start2 start3 start4 start5 start6 start7

start1:
		NODE_NAME=node-a CACHE_PORT=3000 CONFIG_URI=file://${PWD}/config.toml go run main.go

start2:
		NODE_NAME=node-b CACHE_PORT=3001 CONFIG_URI=file://${PWD}/config.toml go run main.go

start3:
		NODE_NAME=node-c CACHE_PORT=3002 CONFIG_URI=file://${PWD}/config.toml go run main.go

start4:
		NODE_NAME=node-d CACHE_PORT=3003 CONFIG_URI=file://${PWD}/config.toml go run main.go

start5:
		NODE_NAME=node-e CACHE_PORT=3004 CONFIG_URI=file://${PWD}/config.toml go run main.go

start6:
		NODE_NAME=node-f CACHE_PORT=3005 CONFIG_URI=file://${PWD}/config.toml go run main.go

start7:
		NODE_NAME=node-g CACHE_PORT=3006 CONFIG_URI=file://${PWD}/config.toml go run main.go

pull:
		@curl "http://127.0.0.1:3000/cache/A"
		@echo ""
