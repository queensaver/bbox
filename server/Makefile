server: main.go
	go build

.PHONY: run
run:
	./server

.PHONY: integration_test
integration_test:
	curl -X POST -d '{"weight": 15.2244, "BBoxID": "aa:bb:cc:dd:ee:ff"}' http://localhost:8333/scale
