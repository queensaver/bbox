.PHONY: integration_test run update_dependencies
integration_test: server
	gofmt -w main.go
	go test ./...
	./server --http_server_port=8123 --debug=true & echo $$! > /tmp/server.pid
	curl -X POST -d '{"weight": 15.2244, "BHiveID": "aa:bb:cc:dd:ee:ff"}' http://localhost:8123/scale
	curl -X POST -d '{"temperature": 22.24, "BHiveID": "aa:bb:cc:dd:ee:ff", "SensorID": "xyz"}' http://localhost:8123/temperature
	kill `cat /tmp/server.pid`

server: main.go
	go build

run:
	./server

update_dependencies:
	go list -m -u all
	go get -u ./...
