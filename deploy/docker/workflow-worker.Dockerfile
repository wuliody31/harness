FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/workflow-worker ./cmd/workflow-worker
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/workflow-worker /workflow-worker
ENTRYPOINT ["/workflow-worker"]
