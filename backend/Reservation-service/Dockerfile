FROM golang:alpine as build_container
WORKDIR /app
COPY ./Reservation-service/go.mod ./Reservation-service/go.sum ./
COPY /utility ../utility
RUN go mod download
COPY ./Reservation-service/ .
RUN go build -o server

FROM alpine
WORKDIR /usr/bin
COPY --from=build_container /app/server .
RUN mkdir cert
COPY ./Reservation-service/cert/reservation-service.crt /cert/reservation-service.crt
COPY ./Reservation-service/cert/reservation-service.key /cert/reservation-service.key
EXPOSE 8000
ENTRYPOINT ["server","https"]