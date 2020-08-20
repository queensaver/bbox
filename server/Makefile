.PHONY: integration_test
integration_test: server
	./server --http_server_port=8123 --debug=true & echo $$! > /tmp/server.pid
	curl -X POST -d '{"weight": 15.2244, "BBoxID": "aa:bb:cc:dd:ee:ff"}' http://localhost:8123/scale
	kill `cat /tmp/server.pid`

server: main.go
	go build

.PHONY: run
run:
	./server


