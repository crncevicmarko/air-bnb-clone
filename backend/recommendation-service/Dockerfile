FROM golang:alpine as build_container
WORKDIR /app
COPY ./recommendation-service/go.mod ./recommendation-service/go.sum ./
COPY /utility ../utility
RUN go mod download
COPY ./recommendation-service/ .
RUN go build -o server

FROM alpine
WORKDIR /usr/bin
COPY --from=build_container /app/server .
EXPOSE 8000
ENTRYPOINT ["server"]