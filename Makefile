COMPOSE ?= docker compose

PROD ?= docker compose -f compose.yaml -f compose.prod.yaml

.PHONY: up-prod down-prod ps-prod backup restore loadtest up down logs ps migrate createadmin resetpassword seed seeddemo sqlc test test-go test-face test-face-model test-e2e test-web web-dev

up:
	$(COMPOSE) up -d --build --wait
down:
	$(COMPOSE) down
logs:
	$(COMPOSE) logs -f app face-service
ps:
	$(COMPOSE) ps
migrate:
	$(COMPOSE) exec app /app migrate up
createadmin:
	$(COMPOSE) exec app /app createadmin --username admin
# make resetpassword USER=student1 [KEEP=1]: generate a new random password (printed once)
resetpassword:
	$(COMPOSE) exec app /app resetpassword --username $(USER) $(if $(KEEP),--keep,)
seed:
	$(COMPOSE) exec app /app seed
# make seeddemo [ARGS="--reset --days 14"]: sample attendance attempts for student1..3
seeddemo:
	$(COMPOSE) exec app /app seeddemo $(ARGS)
sqlc:
	$(COMPOSE) --profile tools run --rm tools sqlc generate
test: test-go test-face test-web
test-go:
	$(COMPOSE) --profile test run --rm go-test
test-face:
	$(COMPOSE) build face-service
	$(COMPOSE) --profile test run --rm face-test
test-face-model:
	$(COMPOSE) build face-service
	$(COMPOSE) --profile test-model run --rm face-test-model
test-e2e:
	$(COMPOSE) --profile e2e run --rm go-e2e
test-web:
	$(COMPOSE) --profile test run --rm web-test
web-dev:
	cd web && bun run dev

# ---- production (compose.prod.yaml) ----
up-prod:
	$(PROD) up -d --build --wait
down-prod:
	$(PROD) down
ps-prod:
	$(PROD) ps
# dump the database and archive uploads into ./backups
backup:
	@mkdir -p backups
	$(PROD) --profile backup run --rm backup
# make restore RESTORE_DB=db-....dump RESTORE_UPLOADS=uploads-....tgz CONFIRM=yes  (stop the app first)
restore:
	$(PROD) --profile restore run --rm restore
# concurrent /verify against the face-service: LOAD_LEVELS=1,2,4,8 make loadtest
loadtest:
	$(COMPOSE) --profile loadtest run --rm loadtest
