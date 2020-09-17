FROM golang:1.13 as builder
COPY / /node-mock
RUN cd /node-mock && make node-mock

FROM ubuntu:bionic
LABEL description="Node mock"

COPY --from=builder /node-mock/_output/node-mock /node-mock
COPY /assets/photo.jpeg /assets/photo.jpeg

RUN chmod +x /node-mock
ENTRYPOINT ["/node-mock"]
