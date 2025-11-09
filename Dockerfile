FROM golang:1.25-alpine3.22 AS builder
WORKDIR /app
RUN apk add --no-cache gcc musl-dev
COPY ./runner .
RUN go mod tidy && go mod download
RUN CGO_ENABLED=1 go build -o monocron-runner .

FROM alpine:3.22
WORKDIR /app
RUN apk add --no-cache sqlite-libs
COPY --from=builder /app/monocron-runner .
EXPOSE 5050
CMD [ "./monocron-runner" ]