cloudrunbasecommand := gcloud run deploy --project=mattbutterfield --region=us-central1 --platform=managed
gobuild := go build

build: build-server build-worker

build-server:
	$(gobuild) -o bin/server cmd/server/main.go

build-worker:
	$(gobuild) -o bin/worker cmd/worker/main.go

deploy: docker-build docker-push deploy-server deploy-worker

deploy-server: docker-build-server docker-push-server
	$(cloudrunbasecommand) --image=gcr.io/mattbutterfield/mattbutterfield.com mattbutterfield

deploy-worker: docker-build-worker docker-push-worker
	$(cloudrunbasecommand) --image=gcr.io/mattbutterfield/mattbutterfield.com-worker mattbutterfield-worker

docker-build:
	docker-compose build

docker-build-server:
	docker-compose build server

docker-build-worker:
	docker-compose build worker

docker-push:
	docker-compose push

docker-push-server:
	docker-compose push server

docker-push-worker:
	docker-compose push worker

db:
	createdb mattbutterfield

fmt:
	go fmt ./...
	npx eslint app/static/js/ --fix

run-server:
	DB_SOCKET="host=localhost dbname=mattbutterfield" USE_LOCAL_FS=true go run cmd/server/main.go

test:
	dropdb --if-exists mattbutterfield_test && createdb mattbutterfield_test && psql -d mattbutterfield_test -f schema.sql
	DB_SOCKET="host=localhost dbname=mattbutterfield_test" go test -v ./app/...

update-deps:
	go get -u ./...
	go mod tidy
	npm upgrade
