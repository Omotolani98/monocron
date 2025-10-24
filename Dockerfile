FROM golang:1.25-alpine3.22
WORKDIR /app

COPY ./runner .
RUN go mod tidy
RUN go build -o monocron-runner .

EXPOSE 5050
CMD [ "./monocron-runner" ]