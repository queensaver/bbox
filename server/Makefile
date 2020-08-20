.PHONY: integration_test run update_dependencies
integration_test: server
	./server --http_server_port=8123 --debug=true & echo $$! > /tmp/server.pid
	curl -X POST -d '{"weight": 15.2244, "BBoxID": "aa:bb:cc:dd:ee:ff"}' http://localhost:8123/scale
	kill `cat /tmp/server.pid`

server: main.go
	go build

run:
	./server

update_dependencies:
	go list -m -u all
