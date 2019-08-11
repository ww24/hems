FROM alpine:3.10

WORKDIR /hems
ADD hems_linux_amd64 /hems

ENTRYPOINT [ "./hems_linux_amd64" ]
