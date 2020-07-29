
.PHONY: build-rpi
build-rpi:
	GOOS=linux GOARM=7 GOARCH=arm go build -o hems_linux_arm .

.PHONY: build-rpi4
build-rpi4:
	GOOS=linux GOARCH=arm64 go build -o hems_linux_arm64 .

.PHONY: docker
docker:
	docker build -t .
