FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -o /out/tests .

FROM alpine:3.21
COPY --from=build /out/tests /tests
ENTRYPOINT ["/tests"]
CMD ["e2e"]
