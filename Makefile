
.PHONY: build-rpi
build-rpi:
	GOOS=linux GOARM=7 GOARCH=arm go build -o hems_linux_arm .

.PHONY: docker
docker:
	docker build -t .
