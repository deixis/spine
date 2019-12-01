start:
		@make -j2 start1 start2

start1:
		cd server && \
		NODE_NAME=node-a GRPC_PORT=8000 CONFIG_URI=file://${PWD}/server/config.toml go run main.go

start2:
		cd server && \
		NODE_NAME=node-b GRPC_PORT=8001 CONFIG_URI=file://${PWD}/server/config.toml go run main.go

call:
		cd client && \
		CONFIG_URI=file://${PWD}/client/config.toml go run main.go