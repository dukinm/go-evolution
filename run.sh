export IMAGE_PREFIX="go-evolution-"
export PG_POSTFIX="database"
export GO_POSTFIX="app"
export NETWORK_POSTFIX="network"
export APP_CONTAINER_NAME="${IMAGE_PREFIX}${GO_POSTFIX}"
export DB_CONTAINER_NAME="${IMAGE_PREFIX}${PG_POSTFIX}"
export NETWORK_NAME="${IMAGE_PREFIX}${NETWORK_POSTFIX}"
export POSTGRES_USER="go-evolution-user"
export POSTGRES_PASSWORD="bAQ9BxTyHgYyv8l"
export POSTGRES_DB="go-evolution-database"
export POSTGRES_PORT="5432"
export APP_PORT="60112"

docker network ls|grep "${NETWORK_NAME}" > /dev/null || docker network create --driver bridge "${NETWORK_NAME}" # проверяем существует ли сетевое подключение, если не существует, то создаем новое с драйвером bridge

docker build -f docker-images/pg-sql/Dockerfile -t "${DB_CONTAINER_NAME}" . # собираем контейнер с базой данных
docker build -f docker-images/go/Dockerfile -t "${APP_CONTAINER_NAME}" . # собираем контейнер с приложением

docker ps -a|grep "${APP_CONTAINER_NAME}" > /dev/null && docker stop "${APP_CONTAINER_NAME}" && docker rm "${APP_CONTAINER_NAME}" # проверяем существует ли контейнер приложения, если существует, то удаляем его
docker run -d --restart unless-stopped --name "${APP_CONTAINER_NAME}" --memory=512m -e POSTGRES_HOST="${DB_CONTAINER_NAME}" -e POSTGRES_PORT="${POSTGRES_PORT}" -e POSTGRES_USER="${POSTGRES_USER}" -e POSTGRES_PASSWORD="${POSTGRES_PASSWORD}" -e POSTGRES_DB="${POSTGRES_DB}" -p "${APP_PORT}":80 "${APP_CONTAINER_NAME}" # запускаем новый контейнер приложения, добавляем ограничения по памяти - 512 мб, пробрасываем переменные окружения для подключения к базе данных, атрибут d означает, что контейнер запускается в фоновом режиме(detached)
docker network inspect "${NETWORK_NAME}"|grep "${APP_CONTAINER_NAME}" > /dev/null || docker network connect "${NETWORK_NAME}" "${APP_CONTAINER_NAME}" # если контейнер не подключен к созданной ранее сети - подключаем его

docker ps -a|grep "${DB_CONTAINER_NAME}" > /dev/null && docker stop "${DB_CONTAINER_NAME}" && docker rm "${DB_CONTAINER_NAME}" #
docker run -d --restart unless-stopped --name "${DB_CONTAINER_NAME}" -v "${pwd}/postgres-data:/var/lib/postgresql/data" --memory=1024m -e POSTGRES_USER="${POSTGRES_USER}" -e POSTGRES_PASSWORD="${POSTGRES_PASSWORD}" -e POSTGRES_DB="${POSTGRES_DB}" -p "${POSTGRES_PORT}":"${POSTGRES_PORT}" "${DB_CONTAINER_NAME}" # запускаем новый контейнер приложения, добавляем ограничения по памяти - 1024 мб, пробрасываем переменные окружения для подключения к базе данных, а также том, который позволит нам сохранять данные при замене контейнера, pwd возвращает нам путь к текущей директории, атрибут d означает, что контейнер запускается в фоновом режиме(detached)
docker network inspect "${NETWORK_NAME}"|grep "${DB_CONTAINER_NAME}" > /dev/null || docker network connect "${NETWORK_NAME}" "${DB_CONTAINER_NAME}" # если контейнер не подключен к созданной ранее сети - подключаем его