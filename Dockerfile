# syntax=docker/dockerfile:1

# ---- build stage ----------------------------------------------------------
FROM golang:1.27-alpine AS build
RUN apk add --no-cache build-base upx tzdata
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make STATIC=1 pack
RUN mkdir -p /out/data

# ---- runtime stage --------------------------------------------------------
FROM scratch
WORKDIR /app
COPY --from=build /src/dist/todoList /app/todoList
COPY --from=build /src/assets /app/assets
COPY --from=build /out/data /app/data
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo

ENV PORT=8080 \
    DB_NAME=data/todoList.db \
    TZ=UTC

EXPOSE 8080
VOLUME ["/app/data"]
ENTRYPOINT ["/app/todoList"]
