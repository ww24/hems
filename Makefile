
.PHONY: build-rpi0
build-rpi0:
	GOOS=linux GOARM=6 GOARCH=arm go build -o hems_linux_armv6 .

.PHONY: build-rpi2
build-rpi2:
	GOOS=linux GOARM=7 GOARCH=arm go build -o hems_linux_armv7 .

.PHONY: build-rpi4
build-rpi4:
	GOOS=linux GOARCH=arm64 go build -o hems_linux_arm64 .

.PHONY: docker
docker:
	docker build -t .
