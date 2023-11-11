FROM golang:1.20
# указываем рабочую директорию
WORKDIR /app
# копируем файлы go.mod и go.sub, чтобы мы могли установить дополнительные компоненты
COPY go.mod go.sum ./
# копируем файлы приложения
COPY *.go ./
# скачиваем модули, указанные в go.mod и go.sum файлах
RUN go mod download
# собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -o /go-evolution
# запускаем приложение
CMD ["/go-evolution"]