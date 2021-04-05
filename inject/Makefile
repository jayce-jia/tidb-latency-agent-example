go-build:
	env GOOS=linux GOARCH=amd64 go build -o 'out/webhook'

docker-build: clean go-build
	docker rmi -f github.com/jayce-jia/tidb-latency-agent-mutating-webhook:0.1 && docker build -t github.com/jayce-jia/tidb-latency-agent-mutating-webhook:0.1 .

clean:
	rm -rf ./out