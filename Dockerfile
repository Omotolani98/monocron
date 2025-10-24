FROM --platform=$BUILDPLATFORM golang:1.25-alpine3.22
WORKDIR /app

COPY ./runner .
RUN go mod tidy
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o monocron-runner .

EXPOSE 5050
CMD [ "./monocron-runner" ]